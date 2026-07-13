package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/pkg/crypto"
)

// NewPostgresDB 创建一个新的 GORM 数据库连接。
func NewPostgresDB(cfg config.DatabaseConfig, mode string) (*gorm.DB, error) {
	logLevel := logger.Warn
	if mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                 logger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true, // 性能更好；使用显式事务
		PrepareStmt:            true, // 缓存预编译语句
		TranslateError:         true, // 将方言错误翻译为 gorm.ErrDuplicatedKey 等，便于业务层精确判定
	})
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// 连接池配置
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	log.Println("[DB] PostgreSQL connected successfully")

	return db, nil
}

// AutoMigrate 为所有模型运行 GORM 自动迁移。
// 验证码仅存储在 Redis 中（临时、基于 TTL）；
// 无需 PostgreSQL 表。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.AuthBinding{},
		&model.SystemConfig{},
		&model.RecoveryCode{},
	)
}

// SeedDefaultConfigs 在默认系统配置不存在时插入它们。
// SMTP、OAuth 等第三方集成配置由「系统设置」在运行时管理并存入数据库，
// 此处仅初始化安全策略开关与备案信息的默认值（幂等，已存在则跳过）。
func SeedDefaultConfigs(db *gorm.DB) error {
	defaults := []model.SystemConfig{
		{Key: "registration_enabled", Value: `true`, Description: "Allow new user registration"},
		{Key: "captcha_enabled", Value: `false`, Description: "Require image captcha on login and registration"},
		{Key: "passkey_enabled", Value: `true`, Description: "Enable WebAuthn/Passkey authentication"},
		{Key: "password_reset_enabled", Value: `true`, Description: "Enable password reset via email"},
		{Key: "password_max_age_days", Value: `0`, Description: "Password max age in days; 0 means never expires"},
		{Key: "two_factor_required", Value: `false`, Description: "Require two-factor authentication for backend access"},
		{Key: "icp_beian", Value: `""`, Description: "ICP filing number shown on the login page (e.g. 京ICP备12345678号-1)"},
		{Key: "police_beian", Value: `""`, Description: "Public-security (公安) filing number shown on the login page"},
	}
	for _, c := range defaults {
		if err := createIfAbsent(db, c); err != nil {
			return err
		}
	}

	log.Println("[DB] Default configurations seeded")
	return nil
}

// createIfAbsent 仅在 key 不存在时创建配置项。
func createIfAbsent(db *gorm.DB, c model.SystemConfig) error {
	var existing model.SystemConfig
	if err := db.Where("key = ?", c.Key).First(&existing).Error; err == gorm.ErrRecordNotFound {
		if err := db.Create(&c).Error; err != nil {
			return fmt.Errorf("seed config %s: %w", c.Key, err)
		}
	}
	return nil
}

// SeedDefaultAdmin 在数据库中不存在任何用户时创建一个默认管理员用户。
// 凭据：admin / admin（邮箱：admin@admin.com，角色：admin）
// 默认管理员必须在首次登录后修改邮箱和密码，避免继续使用初始化凭据。
func SeedDefaultAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := crypto.HashPassword("admin")
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin := &model.User{
		Username:            "admin",
		Email:               "admin@admin.com",
		PasswordHash:        hash,
		Role:                model.RoleSuperAdmin,
		Status:              model.StatusActive,
		ForcePasswordChange: true,
		ForceEmailChange:    true,
	}

	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Println("[DB] Default admin user seeded (admin / admin)")
	return nil
}

// Transaction 在数据库事务中执行 fn。
// 如果 fn 返回错误，则回滚事务。
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
