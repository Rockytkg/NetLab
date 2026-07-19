package radiusd

import (
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"netlab-backend/internal/model"
	radiuserrors "netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/eap"
)

// 认证流水线各 stage 名称。
const (
	StageRequestMetadata = "request_metadata"
	StageNasLookup       = "nas_lookup"
	StageMsgAuth         = "message_authenticator"
	StageRateLimit       = "auth_rate_limit"
	StageVendorParsing   = "vendor_parsing"
	StageBypass          = "bypass_check"
	StageLoadUser        = "load_user"
	StageEAPDispatch     = "eap_dispatch"
	StagePluginAuth      = "plugin_auth"
)

// registerDefaultStages 注册默认认证流水线。
func (s *AuthService) registerDefaultStages() {
	s.pipeline.
		Use(newStage(StageRequestMetadata, s.stageRequestMetadata)).
		Use(newStage(StageNasLookup, s.stageNasLookup)).
		Use(newStage(StageMsgAuth, s.stageMessageAuthenticator)).
		Use(newStage(StageRateLimit, s.stageRateLimit)).
		Use(newStage(StageVendorParsing, s.stageVendorParsing)).
		Use(newStage(StageBypass, s.stageBypassCheck)).
		Use(newStage(StageLoadUser, s.stageLoadUser)).
		Use(newStage(StageEAPDispatch, s.stageEAPDispatch)).
		Use(newStage(StagePluginAuth, s.stagePluginAuth))
}

// stageRequestMetadata 解析请求元数据：用户名、MAC、源 IP、是否 EAP。
func (s *AuthService) stageRequestMetadata(ctx *AuthPipelineContext) error {
	r := ctx.Request

	if s.eapHelper != nil {
		ctx.EAPMethod = resolveEAPMethod(s.cfg().EAPMethod)
		if _, err := eap.ParseEAPMessage(r.Packet); err == nil {
			ctx.IsEAP = true
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr.String())
	if err != nil {
		ctx.RemoteIP = r.RemoteAddr.String()
	} else {
		ctx.RemoteIP = host
	}

	ctx.NasIdentifier = rfc2865.NASIdentifier_GetString(r.Packet)
	ctx.Username = rfc2865.UserName_GetString(r.Packet)
	ctx.CallingStationID = rfc2865.CallingStationID_GetString(r.Packet)

	if ctx.Username == "" {
		return radiuserrors.NewAuthErrorWithStage(
			radiuserrors.MetricsRadiusRejectNotExists,
			"username is empty",
			StageRequestMetadata,
		)
	}
	return nil
}

// stageNasLookup 匹配 NAS 并以 NAS 密钥预设响应包。
func (s *AuthService) stageNasLookup(ctx *AuthPipelineContext) error {
	nas, err := s.GetNas(ctx.RemoteIP, ctx.NasIdentifier)
	if err != nil {
		return err
	}
	ctx.NAS = nas
	ctx.Request.Secret = []byte(nas.Secret)
	ctx.Response = ctx.Request.Response(radius.CodeAccessAccept)
	return nil
}

// stageMessageAuthenticator 按配置模式校验 Message-Authenticator（BlastRADIUS 加固）。
func (s *AuthService) stageMessageAuthenticator(ctx *AuthPipelineContext) error {
	if s.enforceMessageAuthenticator(ctx) {
		ctx.Stop()
	}
	return nil
}

// stageRateLimit 按用户名限流（EAP 多轮挑战豁免）。
func (s *AuthService) stageRateLimit(ctx *AuthPipelineContext) error {
	if ctx.IsEAP {
		return nil
	}
	if err := s.CheckAuthRateLimit(ctx.Username); err != nil {
		return err
	}
	ctx.RateLimitChecked = true
	return nil
}

// stageVendorParsing 解析厂商属性（MAC/VLAN）并判定 MAC 认证。
func (s *AuthService) stageVendorParsing(ctx *AuthPipelineContext) error {
	if ctx.NAS == nil {
		return fmt.Errorf("nas should not be nil before vendor parsing")
	}
	ctx.VendorRequest = s.ParseVendor(ctx.Request, ctx.NAS.VendorCode)
	ctx.IsMacAuth = ctx.VendorRequest.MacAddr != "" && ctx.VendorRequest.MacAddr == ctx.Username
	return nil
}

// stageBypassCheck 免认证检查：命中启用规则（MAC/IP）时直接 Access-Accept，
// 跳过用户加载与密码校验（EAP 流程同样短路），并记录一条 bypass 认证日志。
func (s *AuthService) stageBypassCheck(ctx *AuthPipelineContext) error {
	rules := s.GetBypassRules()
	if len(rules) == 0 {
		return nil
	}
	rule, ok := matchBypassRule(rules, ctx)
	if !ok {
		return nil
	}

	s.addResponseMessageAuthenticator(ctx.Response, ctx.NAS.Secret)
	s.SendAccept(ctx.Writer, ctx.Request, ctx.Response)

	s.logger.Info("radius 免认证放行",
		zap.String("username", ctx.Username),
		zap.String("mac", ctx.VendorRequest.MacAddr),
		zap.String("nasip", ctx.RemoteIP),
		zap.String("rule_type", rule.Type),
		zap.String("rule_value", rule.Value),
	)
	s.RecordAuthLog(&model.RadiusAuthLog{
		Username: ctx.Username,
		NasAddr:  nasAddrOf(ctx.NAS),
		NasPaddr: ctx.RemoteIP,
		MacAddr:  ctx.VendorRequest.MacAddr,
		AuthType: "bypass",
		Result:   model.RadiusAuthResultAccept,
		Reason:   truncateString("bypass:"+rule.Type+"/"+rule.Value, 255),
	})
	ctx.Stop()
	return nil
}

// matchBypassRule 返回首个命中当前请求的免认证规则。
// 候选 MAC 取厂商解析结果（已归一为 ':' 分隔）；候选 IP 取
// Framed-IP-Address 属性与可解析为 IP 的 Calling-Station-Id。
func matchBypassRule(rules []model.RadiusBypass, ctx *AuthPipelineContext) (model.RadiusBypass, bool) {
	mac := ctx.VendorRequest.MacAddr

	var ips []net.IP
	if ip := rfc2865.FramedIPAddress_Get(ctx.Request.Packet); ip != nil && !ip.IsUnspecified() {
		ips = append(ips, ip)
	}
	if ip := net.ParseIP(strings.TrimSpace(ctx.CallingStationID)); ip != nil {
		ips = append(ips, ip)
	}

	for _, rule := range rules {
		switch rule.Type {
		case model.RadiusBypassTypeMac:
			if mac != "" && model.MacListContains(rule.Value, mac) {
				return rule, true
			}
		case model.RadiusBypassTypeIP:
			if bypassIPMatch(rule.Value, ips) {
				return rule, true
			}
		}
	}
	return model.RadiusBypass{}, false
}

// bypassIPMatch 判断规则取值（单 IP 或 CIDR）是否覆盖任一候选 IP。
func bypassIPMatch(value string, ips []net.IP) bool {
	if len(ips) == 0 {
		return false
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, "/") {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return false
		}
		for _, ip := range ips {
			if ipNet.Contains(ip) {
				return true
			}
		}
		return false
	}
	ruleIP := net.ParseIP(value)
	if ruleIP == nil {
		return false
	}
	for _, ip := range ips {
		if ruleIP.Equal(ip) {
			return true
		}
	}
	return false
}

// stageLoadUser 加载并校验用户（状态/有效期，密码已解密，套餐已回填）。
func (s *AuthService) stageLoadUser(ctx *AuthPipelineContext) error {
	user, err := s.GetValidUser(ctx.Username, ctx.IsMacAuth)
	if err != nil {
		return err
	}
	ctx.User = user
	return nil
}

// stageEAPDispatch 将 EAP 请求交给 EAP 协调器进行多轮挑战；
// 成功后跳过密码校验直接跑策略检查并 Accept。
// 首选方法不在启用列表（radius.EapEnabledHandlers）时直接拒绝。
func (s *AuthService) stageEAPDispatch(ctx *AuthPipelineContext) error {
	if !ctx.IsEAP || s.eapHelper == nil {
		return nil
	}

	if !eapMethodAllowed(s.cfg().EAPEnabledHandlers, ctx.EAPMethod) {
		err := radiuserrors.NewAuthError(radiuserrors.MetricsRadiusRejectPasswdError,
			"eap method "+ctx.EAPMethod+" is not enabled")
		s.RecordAuthLog(newRejectLog(ctx, err))
		s.SendReject(ctx.Writer, ctx.Request, err)
		ctx.Stop()
		return nil
	}

	handled, success, eapErr := s.eapHelper.HandleEAPAuthentication(
		ctx.Writer,
		ctx.Request,
		ctx.User,
		ctx.NAS,
		ctx.VendorRequest,
		ctx.Response,
		ctx.EAPMethod,
	)

	if eapErr != nil {
		s.logger.Warn("radius eap 认证失败",
			zap.String("username", ctx.Username),
			zap.String("nasip", ctx.RemoteIP),
			zap.Error(eapErr),
		)
		_ = s.eapHelper.SendEAPFailure(ctx.Writer, ctx.Request, ctx.NAS.Secret, eapErr)
		s.eapHelper.CleanupState(ctx.Request)
		s.RecordAuthLog(newRejectLog(ctx, eapErr))
		ctx.Stop()
		return nil
	}

	if handled {
		if success {
			if err := s.authenticateUser(ctx.Context, ctx.Request, ctx.Response, ctx.User, ctx.NAS, ctx.VendorRequest, ctx.IsMacAuth, true); err != nil {
				_ = s.eapHelper.SendEAPFailure(ctx.Writer, ctx.Request, ctx.NAS.Secret, err)
				s.eapHelper.CleanupState(ctx.Request)
				s.RecordAuthLog(newRejectLog(ctx, err))
				ctx.Stop()
				return nil
			}
			s.sendAcceptResponse(ctx, true)
		}
		ctx.Stop()
	}
	return nil
}

// stagePluginAuth 非 EAP 路径：密码校验 + 策略检查 + Accept。
func (s *AuthService) stagePluginAuth(ctx *AuthPipelineContext) error {
	if ctx.IsStopped() {
		return nil
	}
	if err := s.authenticateUser(ctx.Context, ctx.Request, ctx.Response, ctx.User, ctx.NAS, ctx.VendorRequest, ctx.IsMacAuth, false); err != nil {
		return err
	}
	s.sendAcceptResponse(ctx, false)
	ctx.Stop()
	return nil
}

// newRejectLog 构造一条拒绝认证日志。
func newRejectLog(ctx *AuthPipelineContext, err error) *model.RadiusAuthLog {
	reason := "authentication failed"
	if err != nil {
		reason = truncateString(err.Error(), 255)
	}
	return &model.RadiusAuthLog{
		Username: ctx.Username,
		NasAddr:  nasAddrOf(ctx.NAS),
		NasPaddr: ctx.RemoteIP,
		MacAddr:  ctx.VendorRequest.MacAddr,
		AuthType: authTypeOf(ctx),
		Result:   model.RadiusAuthResultReject,
		Reason:   reason,
	}
}
