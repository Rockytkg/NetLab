package database

import (
	"encoding/json"
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
		&model.PasskeyCredential{},
		&model.SystemConfig{},
	)
}

// SeedDefaultConfigs 在默认系统配置不存在时插入它们。
// OAuth 提供商在启动时始终从配置同步。
func SeedDefaultConfigs(db *gorm.DB, oauthCfg config.OAuthConfig) error {
	defaults := []model.SystemConfig{
		{Key: "registration_enabled", Value: `true`, Description: "Allow new user registration"},
		{Key: "captcha_enabled", Value: `false`, Description: "Require image captcha on login and registration"},
		{Key: "passkey_enabled", Value: `true`, Description: "Enable WebAuthn/Passkey authentication"},
		{Key: "icp_beian", Value: `""`, Description: "ICP filing number shown on the login page (e.g. 京ICP备12345678号-1)"},
		{Key: "police_beian", Value: `""`, Description: "Public-security (公安) filing number shown on the login page"},
	}

	for _, cfg := range defaults {
		var existing model.SystemConfig
		result := db.Where("key = ?", cfg.Key).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&cfg).Error; err != nil {
				return fmt.Errorf("seed config %s: %w", cfg.Key, err)
			}
		}
	}

	// 迁移：如果存在旧的 login_captcha_enabled，则重命名为 captcha_enabled
	var oldKey model.SystemConfig
	if err := db.Where("key = ?", "login_captcha_enabled").First(&oldKey).Error; err == nil {
		// 如果新 key 不存在，则将值复制到新 key
		var newKey model.SystemConfig
		if err := db.Where("key = ?", "captcha_enabled").First(&newKey).Error; err != nil {
			db.Create(&model.SystemConfig{
				Key:         "captcha_enabled",
				Value:       oldKey.Value,
				Description: "Require image captcha on login and registration",
			})
		}
		db.Delete(&oldKey)
	}

	// OAuth 提供商：始终从环境变量同步，以便配置更改在重启后生效。
	oauthProviders := buildOAuthProviders(oauthCfg)
	oauthJSON, err := json.Marshal(oauthProviders)
	if err != nil {
		return fmt.Errorf("marshal oauth providers: %w", err)
	}

	var existing model.SystemConfig
	result := db.Where("key = ?", "oauth_providers").First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&model.SystemConfig{
			Key:         "oauth_providers",
			Value:       string(oauthJSON),
			Description: "Configured OAuth providers",
		}).Error; err != nil {
			return fmt.Errorf("seed oauth_providers: %w", err)
		}
	} else if result.Error == nil {
		existing.Value = string(oauthJSON)
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("sync oauth_providers: %w", err)
		}
	}

	log.Println("[DB] Default configurations seeded")
	return nil
}

// SeedDefaultAdmin 在数据库中不存在任何用户时创建一个默认管理员用户。
// 凭据：admin / admin（邮箱：admin@admin.com，角色：admin）
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
		Username:     "admin",
		Email:        "admin@admin.com",
		PasswordHash: hash,
		Roles:        []string{string(model.RoleAdmin)},
		Status:       model.StatusActive,
	}

	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Println("[DB] Default admin user seeded (admin / admin)")
	return nil
}

// oauthProviderEntry 是为每个提供商存储在 system_configs 中的 JSON 结构。
type oauthProviderEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
	AuthURL string `json:"auth_url"`
}

func buildOAuthProviders(cfg config.OAuthConfig) []oauthProviderEntry {
	var providers []oauthProviderEntry
	for _, p := range cfg.EnabledProviders() {
		authURL := buildAuthURL(p)
		if authURL != "" {
			providers = append(providers, oauthProviderEntry{
				ID:      p.ID,
				Name:    p.Name,
				Icon:    p.Icon,
				Color:   p.Color,
				AuthURL: authURL,
			})
		}
	}
	return providers
}

func buildAuthURL(p config.OAuthProviderEntry) string {
	base := ""
	qs := ""
	switch p.ID {
	case "github":
		base = "https://github.com/login/oauth/authorize"
		qs = "client_id=" + p.ClientID + "&scope=read:user+user:email"
	case "google":
		base = "https://accounts.google.com/o/oauth2/v2/auth"
		qs = "client_id=" + p.ClientID + "&response_type=code&scope=openid+profile+email"
	case "linuxdo":
		base = "https://connect.linux.do/oauth2/authorize"
		qs = "client_id=" + p.ClientID + "&response_type=code&scope=openid+profile+email"
	case "wechat":
		base = "https://open.weixin.qq.com/connect/qrconnect"
		qs = "appid=" + p.ClientID + "&response_type=code&scope=snsapi_login"
		if p.RedirectURL != "" {
			qs += "&redirect_uri=" + p.RedirectURL
		}
		return base + "?" + qs + "#wechat_redirect"
	case "qq":
		base = "https://graph.qq.com/oauth2.0/authorize"
		qs = "client_id=" + p.ClientID + "&response_type=code&scope=get_user_info"
	}
	if p.RedirectURL != "" {
		qs += "&redirect_uri=" + p.RedirectURL
	}
	return base + "?" + qs
}

// Transaction 在数据库事务中执行 fn。
// 如果 fn 返回错误，则回滚事务。
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
