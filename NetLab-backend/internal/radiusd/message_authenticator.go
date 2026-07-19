package radiusd

import (
	"crypto/hmac"
	"crypto/md5"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2869"
)

// Message-Authenticator（RFC 3579 §3.2）校验模式，由 RADIUS_MESSAGE_AUTH_MODE
// 配置。这是 RADIUS/UDP 在 CVE-2024-3596（BlastRADIUS）下的加固开关。
const (
	// MsgAuthModeDisabled 不校验入站报文，出站响应不签名。
	MsgAuthModeDisabled = "disabled"
	// MsgAuthModeWarn 出站响应一律签名；入站报文带 MA 则校验（错误即丢弃），
	// 缺失仅告警。
	MsgAuthModeWarn = "warn"
	// MsgAuthModeEnforce 在 warn 基础上额外丢弃缺失 MA 的 Access-Request。
	MsgAuthModeEnforce = "enforce"
)

// messageAuthResult 是入站 MA 校验结果。
type messageAuthResult int

const (
	// msgAuthAbsent 请求未携带 Message-Authenticator。
	msgAuthAbsent messageAuthResult = iota
	// msgAuthValid 校验通过。
	msgAuthValid
	// msgAuthInvalid 校验失败（错误、畸形或重复出现）。
	msgAuthInvalid
)

// messageAuthenticatorMode 返回配置的校验模式（未知值回落 warn）。
func (s *RadiusService) messageAuthenticatorMode() string {
	switch s.cfg().MessageAuthMode {
	case MsgAuthModeDisabled:
		return MsgAuthModeDisabled
	case MsgAuthModeEnforce:
		return MsgAuthModeEnforce
	default:
		return MsgAuthModeWarn
	}
}

// computeMessageAuthenticator 计算 RFC 3579 HMAC-MD5 Message-Authenticator：
// 计算时将报文中所有 MA 属性值视为 16 个零字节。输入缓冲被复制，不会修改原报文。
func computeMessageAuthenticator(wire, secret []byte) []byte {
	buf := make([]byte, len(wire))
	copy(buf, wire)

	// 属性区从 20 字节头部之后开始；遍历 TLV，将 MA 属性值就地清零。
	for i := 20; i+2 <= len(buf); {
		attrLen := int(buf[i+1])
		if attrLen < 2 || i+attrLen > len(buf) {
			break
		}
		if buf[i] == byte(rfc2869.MessageAuthenticator_Type) && attrLen == 18 {
			for j := i + 2; j < i+attrLen; j++ {
				buf[j] = 0
			}
		}
		i += attrLen
	}

	mac := hmac.New(md5.New, secret)
	mac.Write(buf)
	return mac.Sum(nil)
}

// verifyMessageAuthenticator 校验入站 Access-Request 的 MA 属性。
func (s *RadiusService) verifyMessageAuthenticator(r *radius.Packet, secret []byte) messageAuthResult {
	if len(secret) == 0 {
		return msgAuthInvalid
	}

	values, err := rfc2869.MessageAuthenticator_Gets(r)
	if err != nil {
		return msgAuthInvalid
	}
	switch len(values) {
	case 0:
		return msgAuthAbsent
	case 1:
		// 单个属性，继续校验
	default:
		// RFC 3579 §3.2 至多允许一个 MA 属性。
		return msgAuthInvalid
	}

	received := values[0]
	if len(received) != md5.Size {
		return msgAuthInvalid
	}

	wire, err := r.MarshalBinary()
	if err != nil {
		return msgAuthInvalid
	}

	expected := computeMessageAuthenticator(wire, secret)
	if hmac.Equal(received, expected) {
		return msgAuthValid
	}
	return msgAuthInvalid
}

// verifyResponseMessageAuthenticator 校验 NAS 在 CoA/Disconnect 应答（ACK/NAK）
// 中可选携带的 MA（RFC 5176 §3.4）：摘要以对应请求的 Request Authenticator
// 覆盖 Authenticator 字段后计算。缺席视为通过（响应上 MA 是可选的）。
func verifyResponseMessageAuthenticator(resp *radius.Packet, reqAuth [16]byte, secret []byte) messageAuthResult {
	if resp == nil {
		return msgAuthAbsent
	}

	values, err := rfc2869.MessageAuthenticator_Gets(resp)
	if err != nil {
		return msgAuthInvalid
	}
	switch len(values) {
	case 0:
		return msgAuthAbsent
	case 1:
	default:
		return msgAuthInvalid
	}

	if len(secret) == 0 {
		return msgAuthInvalid
	}

	received := values[0]
	if len(received) != md5.Size {
		return msgAuthInvalid
	}

	wire, err := resp.MarshalBinary()
	if err != nil || len(wire) < 20 {
		return msgAuthInvalid
	}
	copy(wire[4:20], reqAuth[:])

	expected := computeMessageAuthenticator(wire, secret)
	if hmac.Equal(received, expected) {
		return msgAuthValid
	}
	return msgAuthInvalid
}

// addResponseMessageAuthenticator 在非 EAP 的 Access-Accept/Reject 写出前签名
// Message-Authenticator（使响应 Authenticator 覆盖该属性）。禁用模式或无可用
// 密钥时为空操作。
func (s *RadiusService) addResponseMessageAuthenticator(resp *radius.Packet, secret string) {
	if resp == nil {
		return
	}
	if s.messageAuthenticatorMode() == MsgAuthModeDisabled {
		return
	}
	if secret == "" || secret == unknownNasSecret {
		return
	}

	if err := rfc2869.MessageAuthenticator_Set(resp, make([]byte, md5.Size)); err != nil {
		s.logger.Error("重置 message-authenticator 失败", zap.Error(err))
		return
	}
	wire, err := resp.MarshalBinary()
	if err != nil {
		s.logger.Error("序列化响应以计算 message-authenticator 失败", zap.Error(err))
		return
	}
	mac := computeMessageAuthenticator(wire, []byte(secret))
	if err := rfc2869.MessageAuthenticator_Set(resp, mac); err != nil {
		s.logger.Error("设置 message-authenticator 失败", zap.Error(err))
	}
}

// messageAuthDecision 是 enforceMessageAuthenticator 的纯策略核心：
// discard 表示报文必须静默丢弃；warnMissing 表示缺失但可容忍，应记录告警。
func messageAuthDecision(mode string, result messageAuthResult) (discard, warnMissing bool) {
	if mode == MsgAuthModeDisabled {
		return false, false
	}
	switch result {
	case msgAuthInvalid:
		// 错误意味着共享密钥不匹配或报文被篡改，两种模式都丢弃（RFC 3579 §3.2）。
		return true, false
	case msgAuthAbsent:
		if mode == MsgAuthModeEnforce {
			return true, false
		}
		return false, true
	default:
		return false, false
	}
}

// enforceMessageAuthenticator 按配置模式校验入站 Access-Request，返回是否应
// 静默丢弃（pipeline 停止且不写任何响应）。
func (s *AuthService) enforceMessageAuthenticator(ctx *AuthPipelineContext) (discard bool) {
	mode := s.messageAuthenticatorMode()
	if mode == MsgAuthModeDisabled || ctx.NAS == nil {
		return false
	}

	result := s.verifyMessageAuthenticator(ctx.Request.Packet, []byte(ctx.NAS.Secret))
	drop, warnMissing := messageAuthDecision(mode, result)
	if drop {
		reason := "invalid message-authenticator"
		if result == msgAuthAbsent {
			reason = "missing message-authenticator"
		}
		s.logger.Warn("radius access-request 被丢弃",
			zap.String("reason", reason),
			zap.String("username", ctx.Username),
			zap.String("nasip", ctx.RemoteIP),
		)
		return true
	}
	if warnMissing {
		s.logger.Warn("radius access-request 缺少 message-authenticator",
			zap.String("username", ctx.Username),
			zap.String("nasip", ctx.RemoteIP),
			zap.String("mode", mode),
		)
	}
	return false
}
