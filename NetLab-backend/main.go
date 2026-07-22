package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"netlab-backend/config"
	"netlab-backend/internal/database"
	"netlab-backend/internal/handler/admin"
	"netlab-backend/internal/handler/auth"
	loghandler "netlab-backend/internal/handler/log"
	portalHandler "netlab-backend/internal/handler/portal"
	radiusHandler "netlab-backend/internal/handler/radius"
	rbacHandler "netlab-backend/internal/handler/rbac"
	"netlab-backend/internal/mailer"
	"netlab-backend/internal/middleware"
	"netlab-backend/internal/portal"
	"netlab-backend/internal/portal/protocol"
	cmccv1 "netlab-backend/internal/portal/protocol/cmcc/v1"
	cmccv2 "netlab-backend/internal/portal/protocol/cmcc/v2"
	huaweiv1 "netlab-backend/internal/portal/protocol/huawei/v1"
	huaweiv2 "netlab-backend/internal/portal/protocol/huawei/v2"
	"netlab-backend/internal/radiusd"
	"netlab-backend/internal/repository"
	"netlab-backend/internal/router"
	authsvc "netlab-backend/internal/service/auth"
	sysconfig "netlab-backend/internal/service/config"
	logsvc "netlab-backend/internal/service/log"
	portalsvc "netlab-backend/internal/service/portal"
	radiussvc "netlab-backend/internal/service/radius"
	"netlab-backend/internal/service/rbac"
	"netlab-backend/pkg/captcha"
	"netlab-backend/pkg/crypto"
	"netlab-backend/pkg/i18n"
)

// @title           NetLab API
// @version         1.0
// @description     NetLab Backend API — Enterprise-grade authentication and lab management platform.
// @contact.name    NetLab Team
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Type "Bearer" followed by a space and JWT token.

func main() {
	// ── 加载配置 ────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// ── 初始化日志器 ──────────────────────────────────────────────────
	logger := newLogger(cfg.Log)
	defer logger.Sync()

	logger.Info("Starting NetLab Backend",
		zap.String("mode", cfg.Server.Mode),
		zap.String("port", cfg.Server.Port),
	)

	// ── 初始化 i18n Bundle ───────────────────────────────────────────
	if err := i18n.Init("messages/zh-CN.json", "messages/en-US.json"); err != nil {
		logger.Warn("Failed to load i18n message files — falling back to English",
			zap.Error(err),
		)
	} else {
		logger.Info("i18n message files loaded",
			zap.String("languages", "zh-CN, en-US"),
		)
	}

	// ── 设置 Gin 模式 ────────────────────────────────────────────────
	gin.SetMode(cfg.Server.Mode)

	// ── 连接 PostgreSQL ─────────────────────────────────────────────
	db, err := database.NewPostgresDB(cfg.DB, cfg.Server.Mode)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}

	// ── 首次启动时创建数据库表 ────────────────────────────────────────
	// 仅创建不存在的表；已有表结构由显式、人工审核的数据库迁移管理。
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database schema", zap.Error(err))
	}

	// ── 初始化配置加密器 ─────────────────────────────────────────────
	// 敏感配置（SMTP 密码、OAuth 密钥）以 AES-256-GCM 加密后存储于数据库。
	encKey := cfg.Auth.EffectiveEncryptionKey()
	if encKey == "" {
		logger.Warn("no CONFIG_ENCRYPTION_KEY or AUTH_SIGNATURE_KEY set — using an insecure default key for config encryption; set one in production")
		encKey = "netlab-insecure-default-config-key"
	}
	configCipher, err := crypto.NewAESCipher(encKey)
	if err != nil {
		logger.Fatal("Failed to initialize config cipher", zap.Error(err))
	}

	// ── 填充默认数据（幂等 —— 若已存在则跳过） ────
	// 仅初始化安全策略与备案信息默认值；SMTP / OAuth 由「系统设置」管理。
	if err := database.SeedDefaultConfigs(db); err != nil {
		logger.Warn("Seed configs warning", zap.Error(err))
	}

	// ── 连接 Redis ─────────────────────────────────────────────────
	rdb, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// ── 初始化仓储层 ─────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(rdb)
	passkeyRepo := repository.NewPasskeyRepository(db)
	bindingRepo := repository.NewOAuthBindingRepository(db)
	configRepo := repository.NewConfigRepository(db)
	loginLogRepo := repository.NewLoginLogRepository(db)
	radiusUserRepo := repository.NewRadiusUserRepository(db)
	radiusNasRepo := repository.NewRadiusNasRepository(db)
	radiusSessionRepo := repository.NewRadiusSessionRepository(db)
	radiusAccountingRepo := repository.NewRadiusAccountingRepository(db)
	radiusAuthLogRepo := repository.NewRadiusAuthLogRepository(db)
	radiusCertRepo := repository.NewRadiusCertRepository(db)
	radiusBypassRepo := repository.NewRadiusBypassRepository(db)
	portalNasRepo := repository.NewPortalNasRepository(db)
	portalSessionRepo := repository.NewPortalSessionRepository(db)

	// ── 初始化运行时配置服务 ─────────────────────────────────────────
	configService := sysconfig.NewService(configRepo, configCipher, rdb)

	// ── 初始化服务层 ─────────────────────────────────────────────────
	cryptoService, err := authsvc.NewCryptoService(cfg.Auth)
	if err != nil {
		logger.Warn("Crypto service initialization warning (auth signature key may be missing)",
			zap.Error(err),
		)
	}

	tokenService := authsvc.NewTokenService(cfg.JWT, tokenRepo, userRepo)

	// 基于 Redis 存储的验证码管理器
	captchaStore := captcha.NewRedisStore(rdb)
	captchaMgr := captcha.NewManager(captchaStore, cfg.Captcha.Width, cfg.Captcha.Height, cfg.Captcha.MaxRetries)

	// 邮件发送器 —— 从运行时配置服务热加载 SMTP 设置
	emailSender := mailer.NewProvider(configService)

	verificationService := authsvc.NewVerificationService(userRepo, tokenRepo, configService, emailSender, captchaMgr, logger)
	authService := authsvc.NewAuthService(
		userRepo, tokenRepo, configService,
		tokenService, emailSender, logger, verificationService,
	)
	passwordService := authsvc.NewPasswordService(userRepo, tokenRepo, configService, tokenService, verificationService, logger)

	passkeyService := authsvc.NewPasskeyService(
		passkeyRepo, userRepo, tokenRepo, tokenService, configService, rdb, logger,
		cfg.Server.Mode,
	)

	oauthService := authsvc.NewOAuthService(
		configService, userRepo, bindingRepo, tokenRepo, tokenService, logger,
	)

	// ── 初始化 RBAC ──────────────────────────────────────────────────
	rbacService, err := rbac.NewService(db, logger)
	if err != nil {
		logger.Fatal("Failed to initialize RBAC service", zap.Error(err))
	}
	if err := database.SeedDefaultAdmin(db); err != nil {
		logger.Warn("Seed admin user warning", zap.Error(err))
	}

	loginLogService := logsvc.NewService(loginLogRepo, userRepo, rbacService, logger)

	userAdminService := authsvc.NewUserAdminService(userRepo, logger, rbacService)
	importService := authsvc.NewUserImportService(userRepo, logger)

	// ── 初始化 RADIUS 认证计费服务 ─────────────────────────────────────
	// RADIUS UDP 服务（1812/1813，可选 RadSec 2083）与 HTTP 管理 API 共用
	// 同一套仓储；Manager 持有运行时生命周期，管理端的配置变更经它热生效，
	// CoA 服务供管理端踢下线/动态授权使用。
	radiusManager := radiusd.NewManager(
		logger, configCipher,
		radiusUserRepo, radiusNasRepo, radiusSessionRepo, radiusAccountingRepo, radiusAuthLogRepo, radiusBypassRepo, radiusCertRepo,
	)
	radiusMgmtService := radiussvc.NewService(
		radiusUserRepo, radiusNasRepo, radiusSessionRepo, radiusAccountingRepo, radiusAuthLogRepo, radiusCertRepo, radiusBypassRepo,
		configCipher, radiusManager, configService, cfg.Radius,
	)
	// Portal 协议实现必须在组合根显式注册，避免 init 顺序影响运行时行为。
	portalRegistry := protocol.NewRegistry()
	for _, handler := range []protocol.ProtocolHandler{
		cmccv1.NewHandler(nil), cmccv2.NewHandler(nil),
		huaweiv1.NewHandler(nil), huaweiv2.NewHandler(nil),
	} {
		if err := portalRegistry.Register(handler); err != nil {
			logger.Fatal("注册Portal协议失败", zap.String("profile", string(handler.Profile())), zap.Error(err))
		}
	}
	portalMgmtService := portalsvc.NewService(
		portalNasRepo, portalSessionRepo, configCipher,
		portal.NewProtocolClient(portalRegistry, logger), portalsvc.NewSessionStore(rdb), configService, cfg.Portal, logger,
	)
	if err := portalMgmtService.Start(context.Background()); err != nil {
		logger.Fatal("初始化Portal运行时失败", zap.Error(err))
	}
	portalManager := portal.NewManager(logger, portalMgmtService.HandleNotification)
	portalMgmtService.SetManager(portalManager)
	if err := portalManager.Apply(portalMgmtService.EffectiveConfig(context.Background())); err != nil {
		logger.Fatal("初始化Portal通知监听器失败", zap.Error(err))
	}
	adminService := authsvc.NewAdminService(configService, passkeyService)

	// ── 初始化处理器 ─────────────────────────────────────────────────
	twoFactorService := authsvc.NewTwoFactorService(userRepo, tokenRepo, tokenService, configService, logger)

	authHandler := auth.NewAuthHandler(
		authService, verificationService, passwordService, passkeyService, tokenService, oauthService, twoFactorService, captchaMgr, rbacService, loginLogService, logger,
	)
	adminHandler := admin.NewAdminHandler(adminService, userAdminService, importService, emailSender, radiusMgmtService, portalMgmtService, logger)
	rHandler := rbacHandler.NewHandler(rbacService)
	logHandler := loghandler.NewHandler(loginLogService)
	radiusHandler := radiusHandler.NewHandler(radiusMgmtService)
	portalHandler := portalHandler.NewHandler(portalMgmtService)

	// ── 初始化限流器 ─────────────────────────────────────────────────
	var rateLimiter *middleware.RateLimiter
	if cfg.RateLimit.Enabled {
		rateLimiter = middleware.NewRateLimiter(rdb)
	}

	// ── 设置路由 ────────────────────────────────────────────────────
	r := router.Setup(router.RouterConfig{
		Config:        cfg,
		Logger:        logger,
		AuthHandler:   authHandler,
		AdminHandler:  adminHandler,
		RBACHandler:   rHandler,
		LogHandler:    logHandler,
		RadiusHandler: radiusHandler,
		PortalHandler: portalHandler,
		AuthService:   authService,
		TokenService:  tokenService,
		CryptoService: cryptoService,
		JWTManager:    tokenService.JWTManager(),
		SessionStore:  tokenRepo,
		RateLimiter:   rateLimiter,
		Authorizer:    rbacService,
	})

	// ── 启动服务器并支持优雅关闭 ─────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// ── 启动 RADIUS 认证计费服务（UDP 1812/1813，可选 RadSec 2083）─────
	// 生效配置 = 环境变量底座 + 管理端 DB 配置覆盖；DB 读取失败时回落 env
	// 配置（见 EffectiveConfig），不阻断进程启动。
	effectiveRadiusCfg := radiusMgmtService.EffectiveConfig(context.Background())
	radiusManager.Apply(effectiveRadiusCfg)
	if radiusManager.Running() {
		logger.Info("RADIUS service started",
			zap.String("authAddr", fmt.Sprintf("%s:%d", effectiveRadiusCfg.BindHost, effectiveRadiusCfg.AuthPort)),
			zap.String("acctAddr", fmt.Sprintf("%s:%d", effectiveRadiusCfg.BindHost, effectiveRadiusCfg.AcctPort)),
			zap.Bool("eap", effectiveRadiusCfg.EAPEnabled),
			zap.Bool("radsec", effectiveRadiusCfg.RadsecEnabled),
		)
	}
	// 在 goroutine 中启动服务器
	go func() {
		logger.Info("Server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// ── 优雅关闭 ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	if radiusManager.Running() {
		radiusCtx, radiusCancel := context.WithTimeout(context.Background(), radiusd.ShutdownTimeout)
		defer radiusCancel()
		radiusManager.Shutdown(radiusCtx)
		logger.Info("RADIUS service stopped")
	}
	if err := portalManager.Shutdown(); err != nil {
		logger.Warn("Portal通知监听器停止失败", zap.Error(err))
	}
	if err := portalMgmtService.Shutdown(ctx); err != nil {
		logger.Warn("Portal运行时停止超时", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// newLogger 根据配置创建一个 zap.Logger。
func newLogger(cfg config.LogConfig) *zap.Logger {
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
