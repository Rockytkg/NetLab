package radiusd

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/rfc3576"

	"netlab-backend/internal/model"
)

// DefaultCoAPort 是 RFC 5176 §2.1 规定的动态授权默认 UDP 端口。
const DefaultCoAPort = 3799

const (
	// defaultCoATimeout 是单次 CoA/Disconnect 交换的超时时间。
	defaultCoATimeout = 5 * time.Second
	// defaultCoARetries 是首次超时后的额外重传次数（RFC 5176 §2.3 建议
	// 重传同一 Identifier 以便 NAS 去重）。
	defaultCoARetries = 2
)

// ErrSessionNotFound 表示按 Acct-Session-Id 未找到在线会话。
var ErrSessionNotFound = errors.New("coa: online session not found")

// ErrNoTarget 表示 CoA/Disconnect 请求缺少目标 NAS 地址。
var ErrNoTarget = errors.New("coa: target NAS address is required")

// errResponseDiscarded 标记应答因 Message-Authenticator 校验失败被静默丢弃
// （RFC 5176 §3.4），驱动重传循环继续等待可信应答。
var errResponseDiscarded = errors.New("coa: reply discarded: invalid Message-Authenticator (RFC 5176 §3.4)")

// CoAAction 标识动态授权操作类型。
type CoAAction string

const (
	// CoAActionDisconnect 对应 Disconnect-Request（RFC 5176 §2.1）。
	CoAActionDisconnect CoAAction = "disconnect"
	// CoAActionCoA 对应 CoA-Request（RFC 5176 §2.2）。
	CoAActionCoA CoAAction = "coa"
)

// CoATarget 标识接收 CoA/Disconnect 请求的 NAS（动态授权服务端）。
type CoATarget struct {
	Addr   string
	Secret string
	Port   int // <=0 时使用 DefaultCoAPort
}

// endpoint 返回 "host:port" 形式的目的地。
func (t CoATarget) endpoint() string {
	port := t.Port
	if port <= 0 {
		port = DefaultCoAPort
	}
	return net.JoinHostPort(t.Addr, strconv.Itoa(port))
}

// CoATargetFromNas 由 NAS 记录构建 CoATarget。
func CoATargetFromNas(nas *model.RadiusNas) CoATarget {
	return CoATarget{Addr: nas.Ipaddr, Secret: nas.Secret, Port: nas.CoaPort}
}

// SessionIdentity 携带 RFC 5176 §3 的会话识别属性；空字段不下发。
type SessionIdentity struct {
	Username       string
	NasIP          string
	NasIdentifier  string
	AcctSessionID  string
	FramedIP       string
	CallingStation string
	NasPort        *uint32
	NasPortID      string
}

// SessionIdentityFromOnline 由在线会话记录提取识别属性。
func SessionIdentityFromOnline(o *model.RadiusOnline) SessionIdentity {
	id := SessionIdentity{
		Username:       o.Username,
		NasIP:          o.NasAddr,
		AcctSessionID:  o.AcctSessionId,
		FramedIP:       o.FramedIpaddr,
		CallingStation: o.MacAddr,
		NasPortID:      o.NasPortId,
	}
	if o.NasPort > 0 && o.NasPort <= math.MaxUint32 {
		p := uint32(o.NasPort)
		id.NasPort = &p
	}
	return id
}

// CoAResult 是单次 CoA/Disconnect 交换的结构化结果。
type CoAResult struct {
	Action         CoAAction
	Target         string
	Username       string
	AcctSessionID  string
	Identifier     int
	Success        bool
	ResponseCode   string
	ErrorCause     int
	ErrorCauseText string
	Attempts       int
	RTT            time.Duration
	TimedOut       bool
	Err            string
	SentAt         time.Time
}

// CoAService 实现 RFC 5176 动态授权客户端（DAC）。
type CoAService struct {
	*RadiusService
	timeout  time.Duration
	retries  int
	exchange func(ctx context.Context, packet *radius.Packet, addr string) (*radius.Packet, error)
}

// NewCoAService 构造 CoA 服务（5s 超时、2 次重传）。
func NewCoAService(radiusService *RadiusService) *CoAService {
	client := &radius.Client{
		Retry:           0, // 重传由本服务自行管理
		MaxPacketErrors: 10,
	}
	return &CoAService{
		RadiusService: radiusService,
		timeout:       defaultCoATimeout,
		retries:       defaultCoARetries,
		exchange:      client.Exchange,
	}
}

// Disconnect 发送 Disconnect-Request（仅带会话识别属性，RFC 5176 §3）。
func (s *CoAService) Disconnect(ctx context.Context, target CoATarget, id SessionIdentity) (*CoAResult, error) {
	if target.Addr == "" {
		return nil, ErrNoTarget
	}
	packet := radius.New(radius.CodeDisconnectRequest, []byte(target.Secret))
	if err := applyCoAIdentity(packet, id); err != nil {
		return nil, fmt.Errorf("coa: build disconnect request: %w", err)
	}
	if err := signCoAPacket(packet, target.Secret); err != nil {
		return nil, fmt.Errorf("coa: sign disconnect request: %w", err)
	}
	return s.send(ctx, CoAActionDisconnect, target, id, packet), nil
}

// DisconnectSession 按 Acct-Session-Id 定位在线会话并下发 Disconnect-Request。
func (s *CoAService) DisconnectSession(ctx context.Context, acctSessionID string) (*CoAResult, error) {
	target, id, err := s.resolveSession(ctx, acctSessionID)
	if err != nil {
		return nil, err
	}
	return s.Disconnect(ctx, target, id)
}

// CoAAttributeSetter 向 CoA-Request 写入一个授权变更属性（RFC 5176 §2.2）。
type CoAAttributeSetter func(*radius.Packet) error

// WithSessionTimeout 设置 Session-Timeout（#27，秒）。
func WithSessionTimeout(seconds uint32) CoAAttributeSetter {
	return func(p *radius.Packet) error {
		return rfc2865.SessionTimeout_Set(p, rfc2865.SessionTimeout(seconds))
	}
}

// WithFilterID 设置 Filter-Id（#11），用于下发 NAS 侧预定义的过滤器/策略名。
func WithFilterID(filterID string) CoAAttributeSetter {
	return func(p *radius.Packet) error {
		return rfc2865.FilterID_SetString(p, filterID)
	}
}

// CoA 发送 CoA-Request（会话识别属性 + 授权变更属性，RFC 5176 §2.2/§3）。
func (s *CoAService) CoA(ctx context.Context, target CoATarget, id SessionIdentity, setters ...CoAAttributeSetter) (*CoAResult, error) {
	if target.Addr == "" {
		return nil, ErrNoTarget
	}
	if len(setters) == 0 {
		return nil, errors.New("coa: at least one authorization attribute is required")
	}
	packet := radius.New(radius.CodeCoARequest, []byte(target.Secret))
	if err := applyCoAIdentity(packet, id); err != nil {
		return nil, fmt.Errorf("coa: build coa request: %w", err)
	}
	for _, setter := range setters {
		if setter == nil {
			continue
		}
		if err := setter(packet); err != nil {
			return nil, fmt.Errorf("coa: apply attribute: %w", err)
		}
	}
	if err := signCoAPacket(packet, target.Secret); err != nil {
		return nil, fmt.Errorf("coa: sign coa request: %w", err)
	}
	return s.send(ctx, CoAActionCoA, target, id, packet), nil
}

// CoASession 按 Acct-Session-Id 定位在线会话并下发 CoA-Request。
func (s *CoAService) CoASession(ctx context.Context, acctSessionID string, setters ...CoAAttributeSetter) (*CoAResult, error) {
	target, id, err := s.resolveSession(ctx, acctSessionID)
	if err != nil {
		return nil, err
	}
	return s.CoA(ctx, target, id, setters...)
}

// resolveSession 加载在线会话与其 NAS，构建目标与识别属性。
func (s *CoAService) resolveSession(ctx context.Context, acctSessionID string) (CoATarget, SessionIdentity, error) {
	online, err := s.SessionRepo.GetBySessionID(ctx, acctSessionID)
	if err != nil {
		return CoATarget{}, SessionIdentity{}, fmt.Errorf("coa: load session %s: %w", acctSessionID, err)
	}
	if online == nil {
		return CoATarget{}, SessionIdentity{}, fmt.Errorf("%w: %s", ErrSessionNotFound, acctSessionID)
	}
	nas, err := s.GetNas(online.NasAddr, "")
	if err != nil {
		return CoATarget{}, SessionIdentity{}, fmt.Errorf("coa: resolve nas for session %s: %w", acctSessionID, err)
	}
	return CoATargetFromNas(nas), SessionIdentityFromOnline(online), nil
}

// send 在有限超时/重传预算内发送报文并把应答归类为 CoAResult。
func (s *CoAService) send(ctx context.Context, action CoAAction, target CoATarget, id SessionIdentity, packet *radius.Packet) *CoAResult {
	endpoint := target.endpoint()
	result := &CoAResult{
		Action:        action,
		Target:        endpoint,
		Username:      id.Username,
		AcctSessionID: id.AcctSessionID,
		Identifier:    int(packet.Identifier),
		SentAt:        time.Now(),
	}

	var lastErr error
	reqAuth := coaRequestAuthenticator(packet)
	for attempt := 0; attempt <= s.retries; attempt++ {
		if err := ctx.Err(); err != nil {
			lastErr = err
			break
		}
		result.Attempts++

		attemptCtx, cancel := context.WithTimeout(ctx, s.timeout)
		start := time.Now()
		resp, err := s.exchange(attemptCtx, packet, endpoint)
		rtt := time.Since(start)
		cancel()

		if err == nil {
			// 应答携带 MA 时必须验签，否则丢弃并继续重传（RFC 5176 §3.4）。
			if verifyResponseMessageAuthenticator(resp, reqAuth, []byte(target.Secret)) == msgAuthInvalid {
				lastErr = errResponseDiscarded
				continue
			}
			result.RTT = rtt
			classifyCoAResponse(result, resp)
			s.logCoAResult(result)
			return result
		}

		lastErr = err
		// 仅超时重传；其他传输错误立即返回。
		if !isTimeoutErr(err) {
			break
		}
	}

	if lastErr != nil {
		result.Err = lastErr.Error()
	}
	result.TimedOut = isTimeoutErr(lastErr)
	s.logCoAResult(result)
	return result
}

// classifyCoAResponse 把 NAS 应答映射到结果（含 NAK 的 Error-Cause）。
func classifyCoAResponse(result *CoAResult, resp *radius.Packet) {
	if resp == nil {
		return
	}
	result.ResponseCode = resp.Code.String()
	switch resp.Code {
	case radius.CodeDisconnectACK, radius.CodeCoAACK:
		result.Success = true
	case radius.CodeDisconnectNAK, radius.CodeCoANAK:
		result.Success = false
		if cause, err := rfc3576.ErrorCause_Lookup(resp); err == nil {
			result.ErrorCause = int(cause)
			result.ErrorCauseText = cause.String()
		}
	default:
		result.Success = false
	}
}

// applyCoAIdentity 把会话识别属性写入请求包。
func applyCoAIdentity(packet *radius.Packet, id SessionIdentity) error {
	if id.Username != "" {
		if err := rfc2865.UserName_SetString(packet, id.Username); err != nil {
			return fmt.Errorf("set User-Name: %w", err)
		}
	}
	if id.NasIP != "" {
		if ip := net.ParseIP(id.NasIP); ip != nil {
			if err := rfc2865.NASIPAddress_Set(packet, ip); err != nil {
				return fmt.Errorf("set NAS-IP-Address: %w", err)
			}
		}
	}
	if id.NasIdentifier != "" {
		if err := rfc2865.NASIdentifier_Set(packet, []byte(id.NasIdentifier)); err != nil {
			return fmt.Errorf("set NAS-Identifier: %w", err)
		}
	}
	if id.AcctSessionID != "" {
		if err := rfc2866.AcctSessionID_SetString(packet, id.AcctSessionID); err != nil {
			return fmt.Errorf("set Acct-Session-Id: %w", err)
		}
	}
	if id.FramedIP != "" {
		if ip := net.ParseIP(id.FramedIP); ip != nil {
			if err := rfc2865.FramedIPAddress_Set(packet, ip); err != nil {
				return fmt.Errorf("set Framed-IP-Address: %w", err)
			}
		}
	}
	if id.CallingStation != "" {
		if err := rfc2865.CallingStationID_SetString(packet, id.CallingStation); err != nil {
			return fmt.Errorf("set Calling-Station-Id: %w", err)
		}
	}
	if id.NasPort != nil {
		if err := rfc2865.NASPort_Set(packet, rfc2865.NASPort(*id.NasPort)); err != nil {
			return fmt.Errorf("set NAS-Port: %w", err)
		}
	}
	if id.NasPortID != "" {
		if err := rfc2869.NASPortID_Set(packet, []byte(id.NasPortID)); err != nil {
			return fmt.Errorf("set NAS-Port-Id: %w", err)
		}
	}
	return nil
}

// signCoAPacket 为 CoA/Disconnect 请求签名 Message-Authenticator（RFC 5176 §3.4）：
// 计算时 Request Authenticator 与 MA 值均视为 16 个零字节。
func signCoAPacket(packet *radius.Packet, secret string) error {
	packet.Authenticator = [16]byte{}
	if err := rfc2869.MessageAuthenticator_Set(packet, make([]byte, 16)); err != nil {
		return fmt.Errorf("set Message-Authenticator placeholder: %w", err)
	}
	wire, err := packet.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal request for Message-Authenticator: %w", err)
	}
	if err := rfc2869.MessageAuthenticator_Set(packet, computeMessageAuthenticator(wire, []byte(secret))); err != nil {
		return fmt.Errorf("set Message-Authenticator: %w", err)
	}
	return nil
}

// coaRequestAuthenticator 预计算传输层将写入线路的 Request Authenticator，
// 用于校验应答的可选 MA。
func coaRequestAuthenticator(packet *radius.Packet) [16]byte {
	var auth [16]byte
	wire, err := packet.Encode()
	if err != nil || len(wire) < 20 {
		return auth
	}
	copy(auth[:], wire[4:20])
	return auth
}

// isTimeoutErr 判定是否为超时错误（仅超时值得重传）。
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

// logCoAResult 输出一次 CoA 交换的结构化日志。
func (s *CoAService) logCoAResult(result *CoAResult) {
	fields := []zap.Field{
		zap.String("action", string(result.Action)),
		zap.String("target", result.Target),
		zap.String("username", result.Username),
		zap.String("acct_session_id", result.AcctSessionID),
		zap.Int("identifier", result.Identifier),
		zap.Int("attempts", result.Attempts),
		zap.Duration("rtt", result.RTT),
	}
	switch {
	case result.Success:
		s.logger.Info("radius coa 请求已确认", append(fields, zap.String("response", result.ResponseCode))...)
	case result.TimedOut:
		s.logger.Warn("radius coa 请求超时", append(fields, zap.String("error", result.Err))...)
	default:
		s.logger.Warn("radius coa 请求被拒", append(fields,
			zap.String("response", result.ResponseCode),
			zap.Int("error_cause", result.ErrorCause),
			zap.String("error_cause_text", result.ErrorCauseText),
			zap.String("error", result.Err),
		)...)
	}
}
