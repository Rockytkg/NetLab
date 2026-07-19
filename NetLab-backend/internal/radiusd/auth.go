package radiusd

import (
	"context"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"netlab-backend/internal/model"
	radiuserrors "netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/registry"
	"netlab-backend/internal/radiusd/vendors"
	"netlab-backend/internal/radiusd/vendors/microsoft"
)

// AuthService 处理 RADIUS Access-Request（认证请求）。
type AuthService struct {
	*RadiusService
	eapHelper *EAPAuthHelper
	pipeline  *AuthPipeline
}

// NewAuthService 构造认证服务并注册默认 stage。
func NewAuthService(radiusService *RadiusService) *AuthService {
	s := &AuthService{RadiusService: radiusService}
	if radiusService.cfg().EAPEnabled {
		s.eapHelper = NewEAPAuthHelper(radiusService)
	}
	s.pipeline = NewAuthPipeline()
	s.registerDefaultStages()
	return s
}

// ServeRADIUS 实现 radius.Handler 接口，处理单个 Access-Request。
func (s *AuthService) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	// 仅兜底未预期 panic；正常错误走错误返回路径。
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Error("radius auth 处理 panic",
				zap.Any("recover", rec),
				zap.Stack("stacktrace"),
			)
			s.SendReject(w, r, radiuserrors.NewError("internal error"))
		}
	}()

	if r == nil {
		return
	}

	pipelineCtx := NewAuthPipelineContext(s, w, r)
	defer func() {
		if pipelineCtx.RateLimitChecked && pipelineCtx.Username != "" {
			s.ReleaseAuthRateLimit(pipelineCtx.Username)
		}
	}()

	if err := s.pipeline.Execute(pipelineCtx); err != nil {
		s.logAndReject(w, r, pipelineCtx, err)
	}
}

// SendAccept 写出 Access-Accept 并清理 EAP 状态。
func (s *AuthService) SendAccept(w radius.ResponseWriter, r *radius.Request, resp *radius.Packet) {
	if err := w.Write(resp); err != nil {
		s.logger.Error("radius 写出 accept 失败", zap.Error(err))
		return
	}
	if s.eapHelper != nil {
		s.eapHelper.CleanupState(r)
	}
}

// SendReject 写出 Access-Reject（带 Reply-Message 与 Message-Authenticator 签名）。
func (s *AuthService) SendReject(w radius.ResponseWriter, r *radius.Request, err error) {
	resp := r.Response(radius.CodeAccessReject)
	if err != nil {
		msg := err.Error()
		if len(msg) > 253 {
			msg = msg[:253]
		}
		_ = rfc2865.ReplyMessage_SetString(resp, msg)
	}
	s.addResponseMessageAuthenticator(resp, string(r.Secret))
	if writeErr := w.Write(resp); writeErr != nil {
		s.logger.Error("radius 写出 reject 失败", zap.Error(writeErr))
	}
	if s.eapHelper != nil {
		s.eapHelper.CleanupState(r)
	}
}

// logAndReject 记录拒绝日志并发送 reject；同时写认证日志。
// 拒绝错误先经认证守卫链处理（如拒绝延迟守卫在连续拒绝超阈值时替换错误），
// 随后固定 1 秒拒绝延迟用于减缓暴力破解与口令探测。
func (s *AuthService) logAndReject(w radius.ResponseWriter, r *radius.Request, ctx *AuthPipelineContext, err error) {
	if err != nil {
		err = s.runAuthGuards(ctx, err)
	}
	s.logger.Warn("radius 认证拒绝",
		zap.String("username", ctx.Username),
		zap.String("nasip", ctx.RemoteIP),
		zap.Error(err),
	)
	s.RecordAuthLog(&model.RadiusAuthLog{
		Username: ctx.Username,
		NasAddr:  nasAddrOf(ctx.NAS),
		NasPaddr: ctx.RemoteIP,
		MacAddr:  ctx.VendorRequest.MacAddr,
		AuthType: authTypeOf(ctx),
		Result:   model.RadiusAuthResultReject,
		Reason:   truncateString(err.Error(), 255),
	})
	time.Sleep(time.Second)
	s.SendReject(w, r, err)
}

// —— 密码校验与策略检查（插件化，移植自 auth_plugin_runner.go）——

// runAuthGuards 依次执行认证错误守卫（移植自 toughradius handleAuthError）：
// 守卫可观察/替换拒绝错误（目前为拒绝延迟守卫，连续拒绝超阈值时返回
// 限流错误）；返回最终用于日志与应答的错误。
func (s *AuthService) runAuthGuards(ctx *AuthPipelineContext, err error) error {
	authCtx := &auth.AuthContext{
		Request:       ctx.Request,
		User:          ctx.User,
		Nas:           ctx.NAS,
		VendorRequest: ctx.VendorRequest,
		IsMacAuth:     ctx.IsMacAuth,
		Metadata: map[string]interface{}{
			"username": ctx.Username,
		},
	}
	current := err
	for _, guard := range registry.GetAuthGuards() {
		result := guard.OnAuthError(context.Background(), authCtx, "reject", current)
		if result == nil {
			continue
		}
		switch result.Action {
		case auth.GuardActionStop:
			if result.Err != nil {
				return result.Err
			}
			return current
		default:
			if result.Err != nil {
				current = result.Err
			}
		}
	}
	return current
}

// authenticateUser 执行密码校验（EAP 成功路径或 IgnorePassword 开启时跳过）与策略检查。
func (s *AuthService) authenticateUser(
	ctx context.Context,
	r *radius.Request,
	response *radius.Packet,
	user *model.RadiusUser,
	nas *model.RadiusNas,
	vendorReq *vendors.VendorRequest,
	isMacAuth bool,
	skipPasswordValidation bool,
) error {
	authCtx := &auth.AuthContext{
		Request:       r,
		Response:      response,
		User:          user,
		Nas:           nas,
		VendorRequest: vendorReq,
		IsMacAuth:     isMacAuth,
		Metadata: map[string]interface{}{
			"acct_interim_interval": s.cfg().AcctInterimInterval,
			"session_timeout_cap":   s.cfg().SessionTimeout,
		},
	}

	if !isMacAuth && !skipPasswordValidation && !s.cfg().IgnorePassword {
		if err := s.validatePassword(ctx, authCtx, user, isMacAuth); err != nil {
			return err
		}
	}

	if !isMacAuth {
		for _, checker := range registry.GetPolicyCheckers() {
			if err := checker.Check(ctx, authCtx); err != nil {
				return err
			}
		}
	}
	return nil
}

// validatePassword 取本地明文密码（用户加载时已解密）并交给首个可处理的
// 密码校验器（PAP/CHAP/MSCHAP）。
func (s *AuthService) validatePassword(ctx context.Context, authCtx *auth.AuthContext, user *model.RadiusUser, isMacAuth bool) error {
	password := user.Password
	if isMacAuth {
		password = user.MacAddr
	}
	for _, validator := range registry.GetPasswordValidators() {
		if validator.CanHandle(authCtx) {
			return validator.Validate(ctx, authCtx, password)
		}
	}
	return radiuserrors.NewAuthError(radiuserrors.MetricsRadiusRejectPasswdError, "no suitable password validator found")
}

// applyAcceptEnhancers 在 Accept 前执行全部响应增强器（限速/地址池/静态 IP 等）。
func (s *AuthService) applyAcceptEnhancers(user *model.RadiusUser, nas *model.RadiusNas, vendorReq *vendors.VendorRequest, resp *radius.Packet) {
	authCtx := &auth.AuthContext{
		User:          user,
		Nas:           nas,
		VendorRequest: vendorReq,
		Response:      resp,
		Metadata: map[string]interface{}{
			"acct_interim_interval": s.cfg().AcctInterimInterval,
			"session_timeout_cap":   s.cfg().SessionTimeout,
		},
	}
	for _, enhancer := range registry.GetResponseEnhancers() {
		if err := enhancer.Enhance(context.Background(), authCtx); err != nil {
			s.logger.Warn("radius 响应增强器执行失败",
				zap.String("enhancer", enhancer.Name()),
				zap.Error(err),
			)
		}
	}
}

// updateBind 将首次见到的 MAC/VLAN 学习写入用户记录（绑定开启时后续认证校验）。
// MAC 学习语义：开启 MAC 绑定时仅在 mac_addr 为空（首次认证）时学习一次，
// 之后绝不覆盖既有列表；未开启绑定时持续更新为最近看到的 MAC。
func (s *AuthService) updateBind(user *model.RadiusUser, vendorReq *vendors.VendorRequest) {
	if vendorReq.MacAddr != "" {
		learn := user.MacAddr == "" || !user.GetBindMac()
		if learn && user.MacAddr != vendorReq.MacAddr {
			s.UpdateUserMac(user.Username, vendorReq.MacAddr)
		}
	}
	vid1, vid2 := int(vendorReq.Vlanid1), int(vendorReq.Vlanid2)
	if vid1 > 0 && (user.Vlanid1 != vid1 || user.Vlanid2 != vid2) {
		_ = s.UserRepo.UpdateVlanID(context.Background(), user.Username, vid1, vid2)
	}
}

// sendAcceptResponse 是 Accept 的唯一出口：增强响应、写出、更新绑定与认证日志。
func (s *AuthService) sendAcceptResponse(ctx *AuthPipelineContext, isEapFlow bool) {
	if ctx.NAS == nil || ctx.User == nil {
		s.logger.Warn("跳过 accept 响应：上下文不完整", zap.Bool("is_eap", isEapFlow))
		return
	}
	vendorReq := ctx.VendorRequest
	if vendorReq == nil {
		vendorReq = &vendors.VendorRequest{}
	}

	s.applyAcceptEnhancers(ctx.User, ctx.NAS, vendorReq, ctx.Response)

	if isEapFlow && s.eapHelper != nil {
		if err := s.eapHelper.SendEAPSuccess(ctx.Writer, ctx.Request, ctx.Response, ctx.NAS.Secret); err != nil {
			s.logger.Error("发送 EAP-Success 失败", zap.Error(err))
		}
		s.eapHelper.CleanupState(ctx.Request)
	} else {
		s.addResponseMessageAuthenticator(ctx.Response, ctx.NAS.Secret)
		s.SendAccept(ctx.Writer, ctx.Request, ctx.Response)
	}

	s.updateBind(ctx.User, vendorReq)
	s.UpdateUserLastOnline(ctx.User.Username)

	s.logger.Info("radius 认证通过",
		zap.String("username", ctx.Username),
		zap.String("nasip", ctx.RemoteIP),
		zap.Bool("is_eap", isEapFlow),
	)
	s.RecordAuthLog(&model.RadiusAuthLog{
		Username: ctx.Username,
		NasAddr:  ctx.NAS.Ipaddr,
		NasPaddr: ctx.RemoteIP,
		MacAddr:  vendorReq.MacAddr,
		AuthType: authTypeOf(ctx),
		Result:   model.RadiusAuthResultAccept,
		Reason:   "",
	})
}

// nasAddrOf 安全取 NAS IP。
func nasAddrOf(nas *model.RadiusNas) string {
	if nas == nil {
		return ""
	}
	return nas.Ipaddr
}

// authTypeOf 判定认证方式用于日志记录。
func authTypeOf(ctx *AuthPipelineContext) string {
	switch {
	case ctx.IsMacAuth:
		return "mac"
	case ctx.IsEAP:
		return "eap"
	default:
		return authTypeFromPacket(ctx.Request.Packet)
	}
}

// authTypeFromPacket 从请求属性判定密码协议类型。
func authTypeFromPacket(p *radius.Packet) string {
	if p == nil {
		return "pap"
	}
	if len(rfc2865.CHAPPassword_Get(p)) > 0 {
		return "chap"
	}
	if len(microsoft.MSCHAP2Response_Get(p)) > 0 || len(microsoft.MSCHAPResponse_Get(p)) > 0 {
		return "mschap"
	}
	return "pap"
}

// truncateString 截断字符串到 maxLen 字节。
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
