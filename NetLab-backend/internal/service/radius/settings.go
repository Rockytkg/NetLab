package radius

import (
	"context"
	"fmt"
	"strings"

	"netlab-backend/config"
	dtorequest "netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/model"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
)

// 合法的 EAP 方法名（与 radiusd 支持的 handler 一一对应）。
var validEapMethods = []string{"eap-md5", "eap-mschapv2", "eap-tls", "eap-peap", "eap-ttls"}

// EffectiveConfig 计算 RADIUS 运行时的生效配置：以 envCfg（环境变量）为底，
// 逐段用管理端保存在 DB 的配置 blob 覆盖（段存在才覆盖）。
// DB 读取失败时回落到 env 配置，保证调用方（含进程启动路径）不被阻断。
func (s *Service) EffectiveConfig(ctx context.Context) config.RadiusConfig {
	cfg := s.envCfg
	if sys, ok, err := s.cfgSvc.RadiusSystem(ctx); err == nil && ok {
		cfg.Enabled = sys.Enabled
		cfg.BindHost = sys.BindHost
		cfg.AuthPort = sys.AuthPort
		cfg.AcctPort = sys.AcctPort
		cfg.RadsecEnabled = sys.RadsecEnabled
		cfg.RadsecPort = sys.RadsecPort
		cfg.RadsecCertID = sys.RadsecCertID
		cfg.RadsecCACertID = sys.RadsecCACertID
	}
	if srv, ok, err := s.cfgSvc.RadiusServer(ctx); err == nil && ok {
		cfg.MessageAuthMode = srv.MessageAuthMode
		cfg.IgnorePassword = srv.IgnorePassword
		cfg.SessionTimeout = srv.SessionTimeout
		cfg.AcctInterimInterval = srv.AcctInterimInterval
		cfg.HistoryDays = srv.HistoryDays
		cfg.RejectDelayMaxRejects = srv.RejectDelayMaxRejects
		cfg.RejectDelayWindowSeconds = srv.RejectDelayWindowSeconds
	}
	if eap, ok, err := s.cfgSvc.RadiusEap(ctx); err == nil && ok {
		cfg.EAPEnabled = eap.Enabled
		cfg.EAPMethod = eap.Method
		cfg.EAPEnabledHandlers = eap.EnabledHandlers
		cfg.EAPServerCertID = eap.TLSServerCertID
		cfg.EAPClientCACertID = eap.TLSClientCAID
		cfg.EAPTLSMinVersion = eap.TLSMinVersion
	}
	return cfg
}

// GetSettings 返回三段 RADIUS 设置的生效值（env 与 DB 配置合并后的结果）。
func (s *Service) GetSettings(ctx context.Context) (*dtoresponse.RadiusSettingsResponse, error) {
	cfg := s.EffectiveConfig(ctx)
	return &dtoresponse.RadiusSettingsResponse{
		System: dtoresponse.RadiusSystemSettings{
			Enabled:        cfg.Enabled,
			BindHost:       cfg.BindHost,
			AuthPort:       cfg.AuthPort,
			AcctPort:       cfg.AcctPort,
			RadsecEnabled:  cfg.RadsecEnabled,
			RadsecPort:     cfg.RadsecPort,
			RadsecCertID:   cfg.RadsecCertID,
			RadsecCACertID: cfg.RadsecCACertID,
		},
		Server: dtoresponse.RadiusServerSettings{
			MessageAuthMode:          cfg.MessageAuthMode,
			IgnorePassword:           cfg.IgnorePassword,
			SessionTimeout:           cfg.SessionTimeout,
			AcctInterimInterval:      cfg.AcctInterimInterval,
			HistoryDays:              cfg.HistoryDays,
			RejectDelayMaxRejects:    cfg.RejectDelayMaxRejects,
			RejectDelayWindowSeconds: cfg.RejectDelayWindowSeconds,
		},
		Eap: dtoresponse.RadiusEapSettings{
			Enabled:         cfg.EAPEnabled,
			Method:          cfg.EAPMethod,
			EnabledHandlers: cfg.EAPEnabledHandlers,
			TLSServerCertID: cfg.EAPServerCertID,
			TLSClientCAID:   cfg.EAPClientCACertID,
			TLSMinVersion:   cfg.EAPTLSMinVersion,
		},
	}, nil
}

// UpdateSystemSettings 校验并持久化 RADIUS 系统（监听器）设置，随后应用到运行时。
func (s *Service) UpdateSystemSettings(ctx context.Context, req *dtorequest.RadiusSystemSettingsRequest) *apperrors.AppError {
	if strings.TrimSpace(req.BindHost) == "" {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "bindHost 不能为空")
	}
	if !validRadiusPort(req.AuthPort) || !validRadiusPort(req.AcctPort) {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "端口必须在 1-65535 之间")
	}
	// RadSec 仅在启用时强制端口合法（未启用允许占位 0）。
	if req.RadsecEnabled && !validRadiusPort(req.RadsecPort) {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "radsecPort 必须在 1-65535 之间")
	}
	if appErr := s.checkCertRef(ctx, req.RadsecCertID, model.RadiusCertTypeServer, "radsecCertId"); appErr != nil {
		return appErr
	}
	if appErr := s.checkCertRef(ctx, req.RadsecCACertID, model.RadiusCertTypeCA, "radsecCaCertId"); appErr != nil {
		return appErr
	}
	// RadSec 启用时必须能解析到服务器证书：DB 引用或 env 文件路径二选一。
	if req.RadsecEnabled && req.RadsecCertID == 0 && s.envCfg.RadsecCertFile == "" {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "请选择 RadSec 服务器证书")
	}

	if err := s.cfgSvc.SetRadiusSystem(ctx, sysconfig.RadiusSystemSettings{
		Enabled:        req.Enabled,
		BindHost:       strings.TrimSpace(req.BindHost),
		AuthPort:       req.AuthPort,
		AcctPort:       req.AcctPort,
		RadsecEnabled:  req.RadsecEnabled,
		RadsecPort:     req.RadsecPort,
		RadsecCertID:   req.RadsecCertID,
		RadsecCACertID: req.RadsecCACertID,
	}); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to save radius system settings", err)
	}
	s.manager.Apply(s.EffectiveConfig(ctx))
	return nil
}

// ListenerSettings 返回系统设置中管理的 RADIUS 基础监听配置。
func (s *Service) ListenerSettings(ctx context.Context) dtoresponse.RadiusListenerSettings {
	cfg := s.EffectiveConfig(ctx)
	return dtoresponse.RadiusListenerSettings{
		Enabled:  cfg.Enabled,
		BindHost: cfg.BindHost,
		AuthPort: cfg.AuthPort,
		AcctPort: cfg.AcctPort,
	}
}

// UpdateListenerSettings 合并当前 RadSec 配置后更新监听器，保持同一配置表记录和热更新路径。
func (s *Service) UpdateListenerSettings(ctx context.Context, req *dtorequest.RadiusListenerSettingsRequest) *apperrors.AppError {
	cfg := s.EffectiveConfig(ctx)
	return s.UpdateSystemSettings(ctx, &dtorequest.RadiusSystemSettingsRequest{
		Enabled:        req.Enabled,
		BindHost:       req.BindHost,
		AuthPort:       req.AuthPort,
		AcctPort:       req.AcctPort,
		RadsecEnabled:  cfg.RadsecEnabled,
		RadsecPort:     cfg.RadsecPort,
		RadsecCertID:   cfg.RadsecCertID,
		RadsecCACertID: cfg.RadsecCACertID,
	})
}

// UpdateServerSettings 校验并持久化 RADIUS 服务器策略设置，随后应用到运行时。
func (s *Service) UpdateServerSettings(ctx context.Context, req *dtorequest.RadiusServerSettingsRequest) *apperrors.AppError {
	switch req.MessageAuthMode {
	case "disabled", "warn", "enforce":
	default:
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "messageAuthMode 必须是 disabled|warn|enforce 之一")
	}
	if req.AcctInterimInterval < 30 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "acctInterimInterval 不能小于 30 秒")
	}
	if req.SessionTimeout < 0 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "sessionTimeout 不能为负数")
	}
	if req.HistoryDays < 0 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "historyDays 不能为负数")
	}
	if req.RejectDelayMaxRejects < 1 || req.RejectDelayMaxRejects > 1000 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "rejectDelayMaxRejects 必须在 1-1000 之间")
	}
	if req.RejectDelayWindowSeconds < 1 || req.RejectDelayWindowSeconds > 3600 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "rejectDelayWindowSeconds 必须在 1-3600 之间")
	}

	if err := s.cfgSvc.SetRadiusServer(ctx, sysconfig.RadiusServerSettings{
		MessageAuthMode:          req.MessageAuthMode,
		IgnorePassword:           req.IgnorePassword,
		SessionTimeout:           req.SessionTimeout,
		AcctInterimInterval:      req.AcctInterimInterval,
		HistoryDays:              req.HistoryDays,
		RejectDelayMaxRejects:    req.RejectDelayMaxRejects,
		RejectDelayWindowSeconds: req.RejectDelayWindowSeconds,
	}); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to save radius server settings", err)
	}
	s.manager.Apply(s.EffectiveConfig(ctx))
	return nil
}

// UpdateEapSettings 校验并持久化 RADIUS EAP（802.1X）设置，随后应用到运行时。
func (s *Service) UpdateEapSettings(ctx context.Context, req *dtorequest.RadiusEapSettingsRequest) *apperrors.AppError {
	if !isValidEapMethod(req.Method) {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "method 必须是 eap-md5|eap-mschapv2|eap-tls|eap-peap|eap-ttls 之一")
	}
	if appErr := validateEnabledHandlers(req.EnabledHandlers); appErr != nil {
		return appErr
	}
	switch req.TLSMinVersion {
	case "1.2", "1.3":
	default:
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "tlsMinVersion 必须是 1.2|1.3 之一")
	}
	if appErr := s.checkCertRef(ctx, req.TLSServerCertID, model.RadiusCertTypeServer, "tlsServerCertId"); appErr != nil {
		return appErr
	}
	if appErr := s.checkCertRef(ctx, req.TLSClientCAID, model.RadiusCertTypeCA, "tlsClientCaId"); appErr != nil {
		return appErr
	}

	if err := s.cfgSvc.SetRadiusEap(ctx, sysconfig.RadiusEapSettings{
		Enabled:         req.Enabled,
		Method:          req.Method,
		EnabledHandlers: req.EnabledHandlers,
		TLSServerCertID: req.TLSServerCertID,
		TLSClientCAID:   req.TLSClientCAID,
		TLSMinVersion:   req.TLSMinVersion,
	}); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to save radius eap settings", err)
	}
	s.manager.Apply(s.EffectiveConfig(ctx))
	return nil
}

// checkCertRef 校验证书引用：非 0 时证书必须存在且类型与槽位匹配
// （服务器槽位须 server 类型，CA 槽位须 ca 类型）。
func (s *Service) checkCertRef(ctx context.Context, id uint64, wantType, slot string) *apperrors.AppError {
	if id == 0 {
		return nil
	}
	cert, err := s.certs.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius cert", err)
	}
	if cert == nil {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, fmt.Sprintf("%s 引用的证书不存在", slot))
	}
	if cert.CertType != wantType {
		if wantType == model.RadiusCertTypeServer {
			return apperrors.New(apperrors.ErrCodeInvalidRequest, fmt.Sprintf("%s 必须使用服务器（server）类型的证书", slot))
		}
		return apperrors.New(apperrors.ErrCodeInvalidRequest, fmt.Sprintf("%s 必须使用 CA（ca）类型的证书", slot))
	}
	return nil
}

// validRadiusPort 判定端口是否在合法范围内。
func validRadiusPort(port int) bool {
	return port >= 1 && port <= 65535
}

// isValidEapMethod 判定是否为合法的 EAP 方法名。
func isValidEapMethod(method string) bool {
	for _, m := range validEapMethods {
		if method == m {
			return true
		}
	}
	return false
}

// validateEnabledHandlers 校验 EAP 允许方法列表："*" 或合法方法名的逗号列表（逐名校验）。
func validateEnabledHandlers(raw string) *apperrors.AppError {
	if raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if !isValidEapMethod(name) {
			return apperrors.New(apperrors.ErrCodeInvalidRequest,
				fmt.Sprintf("enabledHandlers 必须是 \"*\" 或合法 EAP 方法的逗号列表（%q 不合法）", name))
		}
	}
	return nil
}
