package router

import (
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"netlab-backend/config"
	_ "netlab-backend/docs" // Swagger 文档（匿名导入以触发 init() 注册）
	"netlab-backend/internal/handler/auth"
	"netlab-backend/internal/middleware"
	authsvc "netlab-backend/internal/service/auth"
	pkgjwt "netlab-backend/pkg/jwt"
)

// RouterConfig 持有路由配置所需的全部依赖。
type RouterConfig struct {
	Config        *config.Config
	Logger        *zap.Logger
	AuthHandler   *auth.AuthHandler
	AuthService   *authsvc.AuthService
	TokenService  *authsvc.TokenService
	CryptoService *authsvc.CryptoService
	JWTManager    *pkgjwt.Manager
	Blacklist     pkgjwt.BlacklistManager
	RateLimiter   *middleware.RateLimiter
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
		publicAuth.Use(middleware.OptionalAuth(cfg.JWTManager, cfg.Blacklist))
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

			// 系统配置——中等限流（登录页加载时调用）
			publicAuth.GET("/config",
				cfg.limitModerate("auth-config"),
				cfg.AuthHandler.GetSystemConfig,
			)
		}

		// ── 已认证路由（需要 JWT + 会话签名）──
		authenticated := api.Group("")
		authenticated.Use(middleware.RequireAuth(cfg.JWTManager, cfg.Blacklist))
		// 会话级 HMAC 签名，用于保证请求完整性。
		// 签名密钥由 TokenService 从 Redis 中按用户解析获得。
		// 未携带签名的请求会放行以保持向后兼容；
		// 携带无效签名的请求则会被拒绝。
		authenticated.Use(middleware.SessionSignature(cfg.TokenService.GetSessionSigningKey))
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
