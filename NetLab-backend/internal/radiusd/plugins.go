package radiusd

import (
	"time"

	accthandlers "netlab-backend/internal/radiusd/plugins/accounting/handlers"
	"netlab-backend/internal/radiusd/plugins/auth/checkers"
	"netlab-backend/internal/radiusd/plugins/auth/enhancers"
	"netlab-backend/internal/radiusd/plugins/auth/guards"
	"netlab-backend/internal/radiusd/plugins/auth/validators"
	eaphandlers "netlab-backend/internal/radiusd/plugins/eap/handlers"
	// 厂商解析器通过 init() 自注册到 vendors 注册表
	_ "netlab-backend/internal/radiusd/plugins/vendorparsers/parsers"
	"netlab-backend/internal/radiusd/registry"
)

// registerPlugins 注册全部认证/记账插件与 EAP 方法（进程内调用一次）。
// 移植自 toughradius plugins/init.go，改为从本服务注入依赖。
// 所有注册均按名幂等，可在进程内重建 Server 时安全重复调用。
func registerPlugins(s *RadiusService) {
	// 密码校验器（按 CanHandle 取首个可处理的）
	registry.RegisterPasswordValidator(&validators.PAPValidator{})
	registry.RegisterPasswordValidator(&validators.CHAPValidator{})
	registry.RegisterPasswordValidator(&validators.MSCHAPValidator{})

	// 策略检查器（按 Order 升序执行）
	registry.RegisterPolicyChecker(&checkers.StatusChecker{})
	registry.RegisterPolicyChecker(&checkers.ExpireChecker{})
	registry.RegisterPolicyChecker(&checkers.MacBindChecker{})
	registry.RegisterPolicyChecker(&checkers.VlanBindChecker{})
	registry.RegisterPolicyChecker(checkers.NewOnlineCountChecker(s.SessionRepo))

	// 认证错误守卫（拒绝延迟：阈值/窗口每次错误处理时读取实时配置）
	registry.RegisterAuthGuard(guards.NewRejectDelayGuard(
		func() int64 { return int64(s.cfg().RejectDelayMaxRejects) },
		func() time.Duration { return time.Duration(s.cfg().RejectDelayWindowSeconds) * time.Second },
	))

	// 响应增强器（限速/地址池/静态 IP 等按 NAS 厂商匹配）
	registry.RegisterResponseEnhancer(enhancers.NewDefaultAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewHuaweiAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewH3CAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewZTEAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewMikrotikAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewIkuaiAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewArubaAcceptEnhancer())
	registry.RegisterResponseEnhancer(enhancers.NewCiscoAcceptEnhancer())

	// 记账处理器
	registry.RegisterAccountingHandler(accthandlers.NewStartHandler(s.SessionRepo, s.AccountingRepo))
	registry.RegisterAccountingHandler(accthandlers.NewUpdateHandler(s.SessionRepo))
	registry.RegisterAccountingHandler(accthandlers.NewStopHandler(s.SessionRepo, s.AccountingRepo))
	registry.RegisterAccountingHandler(accthandlers.NewNasStateHandler(s.SessionRepo))

	// EAP 方法（仅在启用 802.1X 时注册；先清空避免重建时残留旧方法）
	registry.ResetEAPHandlers()
	if s.cfg().EAPEnabled {
		registry.RegisterEAPHandler(eaphandlers.NewMD5Handler())
		registry.RegisterEAPHandler(eaphandlers.NewMSCHAPv2Handler())

		settings, resolver := newEAPTLSProviders(s)
		registry.RegisterEAPHandler(eaphandlers.NewTLSHandlerWithConfig(
			eaphandlers.NewSettingsTLSConfigProvider(settings, resolver)))
		registry.RegisterEAPHandler(eaphandlers.NewPEAPHandlerWithConfig(
			eaphandlers.NewSettingsPEAPConfigProvider(settings, resolver)))
		registry.RegisterEAPHandler(eaphandlers.NewTTLSHandlerWithConfig(
			eaphandlers.NewSettingsTTLSConfigProvider(settings, resolver)))
	}
}
