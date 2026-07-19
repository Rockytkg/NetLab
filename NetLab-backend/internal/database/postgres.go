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
// 验证码仅存储在 Redis 中（临时、基于 TTL），无需 PostgreSQL 表。
func AutoMigrate(db *gorm.DB) error {
	models := []any{
		&model.User{},
		&model.Passkey{},
		&model.SystemConfig{},
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.LoginLog{},
		&model.RadiusProfile{},
		&model.RadiusUser{},
		&model.RadiusNas{},
		&model.RadiusOnline{},
		&model.RadiusAccounting{},
		&model.RadiusAuthLog{},
		&model.RadiusCert{},
		&model.RadiusBypass{},
	}

	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	// nb_radius_users.mac_addr 需要容纳逗号分隔的多 MAC 列表；AutoMigrate
	// 不会扩大既有列，幂等 ALTER 兼容旧库（新库列已是 varchar(1024)）。
	if err := db.Exec("ALTER TABLE nb_radius_users ALTER COLUMN mac_addr TYPE varchar(1024)").Error; err != nil {
		return err
	}
	// MAC 认证按 lower(mac_addr) 等值/列表匹配，函数索引支撑热路径查询。
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_radius_users_mac_addr_lower ON nb_radius_users (lower(mac_addr))").Error; err != nil {
		return err
	}
	// Casbin 持久化已被有意移除；应用现在仅以规范化的
	// 角色/权限表作为唯一授权数据源。
	return db.Exec("DROP TABLE IF EXISTS nb_policies").Error
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
		{Key: "smtp", Value: `{}`, Description: "SMTP email delivery settings"},
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

// SeedDefaultAdmin 在数据库中不存在任何用户时创建默认管理员用户。
// 超级管理员凭据：superadmin / superadmin（邮箱：superadmin@netlab.local）
// 管理员凭据：admin / admin（邮箱：admin@admin.com）
// 所有默认管理员必须在首次登录后修改邮箱和密码。
func SeedDefaultAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	// 创建隐藏运维用户 superadmin（role: superadmin）
	saHash, err := crypto.HashPassword("superadmin")
	if err != nil {
		return fmt.Errorf("hash superadmin password: %w", err)
	}
	superAdmin := &model.User{
		Username:            "superadmin",
		Nickname:            "超级管理员",
		Phone:               "13800000000",
		Email:               "superadmin@netlab.local",
		PasswordHash:        saHash,
		Role:                model.RoleSuperAdmin,
		Status:              model.StatusActive,
		ForcePasswordChange: true,
		ForceEmailChange:    true,
	}
	if err := db.Create(superAdmin).Error; err != nil {
		return fmt.Errorf("create superadmin user: %w", err)
	}

	// 创建管理员 admin（role: admin）
	adminHash, err := crypto.HashPassword("admin")
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	admin := &model.User{
		Username:            "admin",
		Nickname:            "管理员",
		Phone:               "13800000099",
		Email:               "admin@admin.com",
		PasswordHash:        adminHash,
		Role:                model.RoleAdmin,
		Status:              model.StatusActive,
		ForcePasswordChange: true,
		ForceEmailChange:    true,
	}
	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Println("[DB] Default users seeded: superadmin / superadmin, admin / admin")
	return nil
}

// Transaction 在数据库事务中执行 fn。
// 如果 fn 返回错误，则回滚事务。
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
