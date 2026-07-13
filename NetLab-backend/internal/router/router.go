package router

import (
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"netlab-backend/config"
	_ "netlab-backend/docs" // Swagger 文档（匿名导入以触发 init() 注册）
	"netlab-backend/internal/handler/admin"
	"netlab-backend/internal/handler/auth"
	"netlab-backend/internal/middleware"
	authsvc "netlab-backend/internal/service/auth"
	pkgjwt "netlab-backend/pkg/jwt"

	"github.com/casbin/casbin/v3"
)

// RouterConfig 持有路由配置所需的全部依赖。
type RouterConfig struct {
	Config        *config.Config
	Logger        *zap.Logger
	AuthHandler   *auth.AuthHandler
	AdminHandler  *admin.AdminHandler
	AuthService   *authsvc.AuthService
	TokenService  *authsvc.TokenService
	CryptoService *authsvc.CryptoService
	JWTManager    *pkgjwt.Manager
	SessionStore  pkgjwt.SessionValidator
	RateLimiter   *middleware.RateLimiter
	Enforcer      *casbin.Enforcer
}

// Setup 配置所有路由和中间件，并返回 Gin 引擎。
func Setup(cfg RouterConfig) *gin.Engine {
	r := gin.New()

	// ── 全局中间件链 ─────────────────────────────────────
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(cfg.Config.CORS))
	r.Use(middleware.Recovery(cfg.Logger))
	r.Use(middleware.I18N())

	// 全局限流：每个 IP 每分钟 100 次请求
	if cfg.RateLimiter != nil && cfg.Config.RateLimit.Enabled {
		r.Use(cfg.RateLimiter.Global(cfg.Config.RateLimit.Global, time.Minute))
	}

	// ── 健康检查 ────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── Swagger 文档 ────────────────────────────────────────
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ── API 路由 ──────────────────────────────────────────
	api := r.Group("/api")
	{
		publicAuth := api.Group("/auth")
		publicAuth.Use(middleware.OptionalAuth(cfg.JWTManager, cfg.SessionStore))
		{
			// ── 预共享密钥保护的端点（仅签名）──
			// 请求体以明文发送（机密性由 HTTPS 保证）。
			// Signature 中间件校验请求体的 HMAC，并强制
			// ±5 分钟的时间戳窗口以拒绝重放攻击。
			securePublic := publicAuth.Group("")
			securePublic.Use(middleware.Signature(middleware.SignatureConfig{
				Required:      true,
				SignatureKey:  cfg.CryptoService.SignatureKey(),
				SignatureSalt: cfg.CryptoService.SignatureSalt(),
			}))
			{
				// 登录：严格限流——每个 IP 每分钟 5 次请求
				securePublic.POST("/login",
					cfg.limitStrict("auth-login"),
					cfg.AuthHandler.Login,
				)

				// 注册：中等限流——每个 IP 每分钟 10 次请求
				securePublic.POST("/register",
					cfg.limitModerate("auth-register"),
					cfg.AuthHandler.Register,
				)

				// 重置密码：严格限流——每个 IP 每分钟 5 次请求
				securePublic.POST("/reset-password",
					cfg.limitStrict("auth-reset-pw"),
					cfg.AuthHandler.ResetPassword,
				)
			}

			// ── 公开端点（无需加密）────────────
			// 刷新 token——严格限流以防止滥用
			publicAuth.POST("/refresh",
				cfg.limitStrict("auth-refresh"),
				cfg.AuthHandler.RefreshToken,
			)

			// 图形验证码——中等限流
			publicAuth.GET("/captcha",
				cfg.limitModerate("auth-captcha"),
				cfg.AuthHandler.GetCaptcha,
			)

			// 发送验证码——非常严格：每个 IP 每分钟 3 次请求
			publicAuth.POST("/send-code",
				cfg.limitVeryStrict("auth-sendcode"),
				cfg.AuthHandler.SendCode,
			)

			// 校验验证码——中等限流
			publicAuth.POST("/verify-code",
				cfg.limitModerate("auth-verifycode"),
				cfg.AuthHandler.VerifyCode,
			)

			// 忘记密码——中等限流
			publicAuth.POST("/forgot-password",
				cfg.limitModerate("auth-forgot"),
				cfg.AuthHandler.ForgotPassword,
			)

			// Passkey 认证选项——中等限流
			publicAuth.GET("/passkey/auth-options",
				cfg.limitModerate("passkey-auth-opt"),
				cfg.AuthHandler.GetPasskeyAuthOptions,
			)

			// Passkey 校验——严格限流
			publicAuth.POST("/passkey/verify",
				cfg.limitStrict("passkey-verify"),
				cfg.AuthHandler.VerifyPasskeyAuth,
			)

			// 两步验证登录校验 (严格限流): 交换挑战令牌为访问令牌
			publicAuth.POST("/login/2fa",
				cfg.limitStrict("auth-login-2fa"),
				cfg.AuthHandler.VerifyTwoFactorLogin,
			)

			// 两步验证恢复码登录 (严格限流)
			publicAuth.POST("/login/recovery",
				cfg.limitStrict("auth-login-recovery"),
				cfg.AuthHandler.VerifyRecoveryLogin,
			)

			// OAuth 授权——中等限流（用于发起 OAuth 流程）
			publicAuth.GET("/oauth/authorize",
				cfg.limitModerate("oauth-authorize"),
				cfg.AuthHandler.OAuthAuthorize,
			)

			// OAuth 回调——中等限流
			publicAuth.POST("/oauth/callback",
				cfg.limitModerate("oauth-callback"),
				cfg.AuthHandler.OAuthCallback,
			)
			publicAuth.POST("/oauth/bind-existing",
				cfg.limitStrict("oauth-bind-existing"),
				cfg.AuthHandler.OAuthBindExisting,
			)
			publicAuth.POST("/oauth/create-account",
				cfg.limitStrict("oauth-create-account"),
				cfg.AuthHandler.OAuthCreateAccount,
			)

			// 系统配置——中等限流（登录页加载时调用）
			publicAuth.GET("/config",
				cfg.limitModerate("auth-config"),
				cfg.AuthHandler.GetSystemConfig,
			)
		}

		// ── 已认证路由（需要 JWT）──
		authenticated := api.Group("")
		authenticated.Use(middleware.RequireAuth(cfg.JWTManager, cfg.SessionStore))
		authenticated.Use(middleware.Authorize(cfg.Enforcer))
		{
			authUser := authenticated.Group("/auth")
			{
				// 用户信息——标准限流（频繁轮询）
				authUser.GET("/userinfo",
					cfg.limitStandard("auth-userinfo"),
					cfg.AuthHandler.GetUserInfo,
				)

				// 登出——标准限流
				authUser.POST("/logout",
					cfg.limitStandard("auth-logout"),
					cfg.AuthHandler.Logout,
				)

				// Passkey 注册选项——标准限流
				authUser.GET("/passkey/register-options",
					cfg.limitStandard("passkey-reg-opt"),
					cfg.AuthHandler.GetPasskeyRegisterOptions,
				)

				// Passkey 注册——中等限流
				authUser.POST("/passkey/register",
					cfg.limitModerate("passkey-register"),
					cfg.AuthHandler.VerifyPasskeyRegistration,
				)

				// Passkey 列表——标准限流
				authUser.GET("/passkey/list",
					cfg.limitStandard("passkey-list"),
					cfg.AuthHandler.ListPasskeys,
				)

				// Passkey 删除——标准限流
				authUser.DELETE("/passkey/:id",
					cfg.limitStandard("passkey-delete"),
					cfg.AuthHandler.DeletePasskey,
				)

				// ── 账户自助操作 ──
				// 修改密码——严格限流
				authUser.POST("/account/change-password",
					cfg.limitStrict("account-change-pw"),
					cfg.AuthHandler.ChangePassword,
				)
				authUser.POST("/account/security-update",
					cfg.limitStrict("account-security-update"),
					cfg.AuthHandler.CompleteSecurityUpdate,
				)

				// 两步验证 (TOTP) 绑定与管理 - 已登录用户
				authUser.POST("/2fa/setup",
					cfg.limitModerate("account-2fa-setup"),
					cfg.AuthHandler.BeginTwoFactorSetup,
				)
				authUser.POST("/2fa/enable",
					cfg.limitStrict("account-2fa-enable"),
					cfg.AuthHandler.ConfirmTwoFactorSetup,
				)
				authUser.POST("/2fa/disable",
					cfg.limitStrict("account-2fa-disable"),
					cfg.AuthHandler.DisableTwoFactor,
				)
				authUser.PUT("/account/preferred-auth-method",
					cfg.limitModerate("account-preferred-auth-method"),
					cfg.AuthHandler.SetPreferredAuthMethod,
				)

				// 账户内取验证码（发往本人邮箱）——非常严格限流
				authUser.POST("/account/email-code",
					cfg.limitVeryStrict("account-email-code"),
					cfg.AuthHandler.SendAccountEmailCode,
				)
				authUser.POST("/account/email-change-code",
					cfg.limitVeryStrict("account-email-change-code"),
					cfg.AuthHandler.SendChangeEmailCode,
				)
				authUser.PUT("/account/email",
					cfg.limitStrict("account-change-email"),
					cfg.AuthHandler.ChangeEmail,
				)

				// ── 第三方账号绑定管理 ──
				// 绑定列表——标准限流
				authUser.GET("/oauth/bindings",
					cfg.limitStandard("oauth-bindings-list"),
					cfg.AuthHandler.ListOAuthBindings,
				)
				// 获取绑定授权 URL——中等限流
				authUser.GET("/oauth/bind-url",
					cfg.limitModerate("oauth-bind-url"),
					cfg.AuthHandler.GetOAuthBindURL,
				)
				// 完成绑定——中等限流
				authUser.POST("/oauth/bind",
					cfg.limitModerate("oauth-bind"),
					cfg.AuthHandler.BindOAuth,
				)
				// 解绑——标准限流
				authUser.DELETE("/oauth/bindings/:provider",
					cfg.limitStandard("oauth-unbind"),
					cfg.AuthHandler.UnbindOAuth,
				)
			}

			// ── 管理端路由（需要 admin 角色）──
			// 系统设置的读写仅对 admin 开放。
			if cfg.AdminHandler != nil {
				adminGroup := authenticated.Group("/admin")
				{
					adminGroup.GET("/settings",
						cfg.limitStandard("admin-settings-get"),
						cfg.AdminHandler.GetSettings,
					)
					adminGroup.PUT("/settings/security",
						cfg.limitModerate("admin-security"),
						cfg.AdminHandler.UpdateSecurity,
					)
					adminGroup.PUT("/settings/beian",
						cfg.limitModerate("admin-beian"),
						cfg.AdminHandler.UpdateBeian,
					)
					adminGroup.PUT("/settings/smtp",
						cfg.limitModerate("admin-smtp"),
						cfg.AdminHandler.UpdateSMTP,
					)
					adminGroup.POST("/settings/smtp/test",
						cfg.limitStrict("admin-smtp-test"),
						cfg.AdminHandler.TestSMTP,
					)
					adminGroup.PUT("/settings/oauth/:provider",
						cfg.limitModerate("admin-oauth"),
						cfg.AdminHandler.UpdateOAuthProvider,
					)

					// ── 用户管理 ──
					adminGroup.GET("/users",
						cfg.limitStandard("admin-users-list"),
						cfg.AdminHandler.ListUsers,
					)
					adminGroup.POST("/users",
						cfg.limitModerate("admin-users-create"),
						cfg.AdminHandler.CreateUser,
					)
					adminGroup.DELETE("/users",
						cfg.limitModerate("admin-users-delete"),
						cfg.AdminHandler.BatchDeleteUsers,
					)
					adminGroup.PUT("/users/role",
						cfg.limitModerate("admin-users-role"),
						cfg.AdminHandler.BatchUpdateRole,
					)
					adminGroup.PUT("/users/reset-password",
						cfg.limitModerate("admin-users-reset-pw"),
						cfg.AdminHandler.BatchResetPassword,
					)
					adminGroup.POST("/users/import",
						cfg.limitStrict("admin-users-import"),
						cfg.AdminHandler.ImportUsers,
					)
					adminGroup.PUT("/users/:id",
						cfg.limitModerate("admin-users-update"),
						cfg.AdminHandler.UpdateUser,
					)
				}
			}
		}
	}

	return r
}

// ─── 限流辅助方法 ────────────────────────────────────
// 若限流被禁用，每个方法都返回 no-op。

func (cfg *RouterConfig) limitVeryStrict(prefix string) gin.HandlerFunc {
	if cfg.RateLimiter == nil {
		return nil
	}
	return cfg.RateLimiter.ByRule(middleware.RateLimitByIP(3, time.Minute, prefix))
}

func (cfg *RouterConfig) limitStrict(prefix string) gin.HandlerFunc {
	if cfg.RateLimiter == nil {
		return nil
	}
	return cfg.RateLimiter.ByRule(middleware.RateLimitByIP(5, time.Minute, prefix))
}

func (cfg *RouterConfig) limitModerate(prefix string) gin.HandlerFunc {
	if cfg.RateLimiter == nil {
		return nil
	}
	return cfg.RateLimiter.ByRule(middleware.RateLimitByIP(15, time.Minute, prefix))
}

func (cfg *RouterConfig) limitStandard(prefix string) gin.HandlerFunc {
	if cfg.RateLimiter == nil {
		return nil
	}
	return cfg.RateLimiter.ByRule(middleware.RateLimitByIP(60, time.Minute, prefix))
}
