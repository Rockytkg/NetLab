package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 保存应用程序的所有配置。
type Config struct {
	Server    ServerConfig
	DB        DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Auth      AuthConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
	Captcha   CaptchaConfig
	Log       LogConfig
	OAuth     OAuthConfig
	SMTP      SMTPConfig
}

// ServerConfig 保存服务器相关配置。
type ServerConfig struct {
	Port         string
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig 保存 PostgreSQL 连接配置。
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN 返回 PostgreSQL 连接字符串。
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig 保存 Redis 连接配置。
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// Addr 返回 Redis 地址。
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig 保存 JWT 相关配置。
type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	Issuer        string
}

// AuthConfig 保存用于保护公开认证端点的预共享签名 key/salt。
type AuthConfig struct {
	SignatureKey  string
	SignatureSalt string
}

// RateLimitConfig 保存限流配置。
type RateLimitConfig struct {
	Enabled bool
	Global  int
}

// CORSConfig 保存 CORS 配置。
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CaptchaConfig 保存验证码配置。
type CaptchaConfig struct {
	Width      int
	Height     int
	MaxRetries int
}

// LogConfig 保存日志配置。
type LogConfig struct {
	Level  string
	Format string
}

// SMTPConfig 保存 SMTP 邮件服务器配置。
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// IsConfigured 在已提供 SMTP 设置时返回 true。
func (s SMTPConfig) IsConfigured() bool {
	return s.Host != "" && s.Port > 0 && s.Username != "" && s.From != ""
}

// OAuthProviderConfig 保存单个 OAuth 提供商的配置。
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OAuthConfig 保存 OAuth 第三方登录配置。
type OAuthConfig struct {
	GitHub  OAuthProviderConfig
	Google  OAuthProviderConfig
	LinuxDO OAuthProviderConfig
	WeChat  OAuthProviderConfig
	QQ      OAuthProviderConfig
}

// OAuthProviderEntry 是已配置 OAuth 提供商的公开元数据。
type OAuthProviderEntry struct {
	ID           string
	Name         string
	Icon         string
	Color        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// HasAnyEnabled 在至少配置了一个 OAuth 提供商时返回 true。
func (o OAuthConfig) HasAnyEnabled() bool {
	return o.GitHub.ClientID != "" ||
		o.Google.ClientID != "" ||
		o.LinuxDO.ClientID != "" ||
		o.WeChat.ClientID != "" ||
		o.QQ.ClientID != ""
}

// EnabledProviders 返回已配置 OAuth 提供商及其元数据的列表。
func (o OAuthConfig) EnabledProviders() []OAuthProviderEntry {
	var providers []OAuthProviderEntry

	add := func(id, name, icon, color string, cfg OAuthProviderConfig) {
		if cfg.ClientID != "" && cfg.ClientSecret != "" {
			providers = append(providers, OAuthProviderEntry{
				ID:           id,
				Name:         name,
				Icon:         icon,
				Color:        color,
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
				RedirectURL:  cfg.RedirectURL,
			})
		}
	}

	add("github", "GitHub", "github", "#24292f", o.GitHub)
	add("google", "Google", "google", "#4285f4", o.Google)
	add("linuxdo", "LinuxDO", "linuxdo", "#4a90d9", o.LinuxDO)
	add("wechat", "WeChat", "wechat", "#07c160", o.WeChat)
	add("qq", "QQ", "qq", "#12b7f5", o.QQ)

	return providers
}

// Load 从 .env 文件和环境变量中读取配置。
func Load() (*Config, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("SERVER_MODE", "debug")
	v.SetDefault("SERVER_READ_TIMEOUT", "30s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "30s")

	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "netlab")
	v.SetDefault("DB_PASSWORD", "changeme")
	v.SetDefault("DB_NAME", "netlab")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 10)
	v.SetDefault("DB_CONN_MAX_LIFETIME", "5m")

	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_POOL_SIZE", 10)

	v.SetDefault("JWT_ACCESS_SECRET", "")
	v.SetDefault("JWT_REFRESH_SECRET", "")
	v.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	v.SetDefault("JWT_ISSUER", "netlab")

	v.SetDefault("AUTH_SIGNATURE_KEY", "")
	v.SetDefault("AUTH_SIGNATURE_SALT", "")

	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("RATE_LIMIT_GLOBAL", 100)

	v.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	v.SetDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	v.SetDefault("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,X-Request-Id,X-Signature,X-Timestamp,Accept-Language,X-User-Language")

	// 验证码渲染尺寸。前端在登录框内以 32px 高度展示该图片，
	// 因此源图按 2 倍（64px）渲染，可在高分屏上获得清晰的整数倍缩放；
	// 宽度按 3:1 比例取 192px，保证算式有充足的水平空间。
	v.SetDefault("CAPTCHA_WIDTH", 192)
	v.SetDefault("CAPTCHA_HEIGHT", 64)
	v.SetDefault("CAPTCHA_MAX_RETRIES", 5)

	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	// OAuth 默认值
	v.SetDefault("OAUTH_GITHUB_CLIENT_ID", "")
	v.SetDefault("OAUTH_GITHUB_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_GITHUB_REDIRECT_URL", "")
	v.SetDefault("OAUTH_GOOGLE_CLIENT_ID", "")
	v.SetDefault("OAUTH_GOOGLE_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_GOOGLE_REDIRECT_URL", "")
	v.SetDefault("OAUTH_LINUXDO_CLIENT_ID", "")
	v.SetDefault("OAUTH_LINUXDO_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_LINUXDO_REDIRECT_URL", "")
	v.SetDefault("OAUTH_WECHAT_CLIENT_ID", "")
	v.SetDefault("OAUTH_WECHAT_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_WECHAT_REDIRECT_URL", "")
	v.SetDefault("OAUTH_QQ_CLIENT_ID", "")
	v.SetDefault("OAUTH_QQ_CLIENT_SECRET", "")
	v.SetDefault("OAUTH_QQ_REDIRECT_URL", "")

	// SMTP 默认值
	v.SetDefault("SMTP_HOST", "")
	v.SetDefault("SMTP_PORT", 587)
	v.SetDefault("SMTP_USERNAME", "")
	v.SetDefault("SMTP_PASSWORD", "")
	v.SetDefault("SMTP_FROM", "")
	v.SetDefault("SMTP_USE_TLS", true)

	// 从 .env 读取
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	_ = v.ReadInConfig() // 如果 .env 不存在则忽略错误

	// 解析时长
	readTimeout, _ := time.ParseDuration(v.GetString("SERVER_READ_TIMEOUT"))
	writeTimeout, _ := time.ParseDuration(v.GetString("SERVER_WRITE_TIMEOUT"))
	accessExpiry, _ := time.ParseDuration(v.GetString("JWT_ACCESS_EXPIRY"))
	refreshExpiry, _ := time.ParseDuration(v.GetString("JWT_REFRESH_EXPIRY"))
	connMaxLifetime, _ := time.ParseDuration(v.GetString("DB_CONN_MAX_LIFETIME"))

	cfg := &Config{
		Server: ServerConfig{
			Port:         v.GetString("SERVER_PORT"),
			Mode:         v.GetString("SERVER_MODE"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		DB: DatabaseConfig{
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetInt("DB_PORT"),
			User:            v.GetString("DB_USER"),
			Password:        v.GetString("DB_PASSWORD"),
			Name:            v.GetString("DB_NAME"),
			SSLMode:         v.GetString("DB_SSLMODE"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: connMaxLifetime,
		},
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
			PoolSize: v.GetInt("REDIS_POOL_SIZE"),
		},
		JWT: JWTConfig{
			AccessSecret:  v.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: v.GetString("JWT_REFRESH_SECRET"),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
			Issuer:        v.GetString("JWT_ISSUER"),
		},
		Auth: AuthConfig{
			SignatureKey:  v.GetString("AUTH_SIGNATURE_KEY"),
			SignatureSalt: v.GetString("AUTH_SIGNATURE_SALT"),
		},
		RateLimit: RateLimitConfig{
			Enabled: v.GetBool("RATE_LIMIT_ENABLED"),
			Global:  v.GetInt("RATE_LIMIT_GLOBAL"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitAndTrim(v.GetString("CORS_ALLOWED_ORIGINS")),
			AllowedMethods: splitAndTrim(v.GetString("CORS_ALLOWED_METHODS")),
			AllowedHeaders: splitAndTrim(v.GetString("CORS_ALLOWED_HEADERS")),
		},
		Captcha: CaptchaConfig{
			Width:      v.GetInt("CAPTCHA_WIDTH"),
			Height:     v.GetInt("CAPTCHA_HEIGHT"),
			MaxRetries: v.GetInt("CAPTCHA_MAX_RETRIES"),
		},
		Log: LogConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
		OAuth: OAuthConfig{
			GitHub: OAuthProviderConfig{
				ClientID:     v.GetString("OAUTH_GITHUB_CLIENT_ID"),
				ClientSecret: v.GetString("OAUTH_GITHUB_CLIENT_SECRET"),
				RedirectURL:  v.GetString("OAUTH_GITHUB_REDIRECT_URL"),
			},
			Google: OAuthProviderConfig{
				ClientID:     v.GetString("OAUTH_GOOGLE_CLIENT_ID"),
				ClientSecret: v.GetString("OAUTH_GOOGLE_CLIENT_SECRET"),
				RedirectURL:  v.GetString("OAUTH_GOOGLE_REDIRECT_URL"),
			},
			LinuxDO: OAuthProviderConfig{
				ClientID:     v.GetString("OAUTH_LINUXDO_CLIENT_ID"),
				ClientSecret: v.GetString("OAUTH_LINUXDO_CLIENT_SECRET"),
				RedirectURL:  v.GetString("OAUTH_LINUXDO_REDIRECT_URL"),
			},
			WeChat: OAuthProviderConfig{
				ClientID:     v.GetString("OAUTH_WECHAT_CLIENT_ID"),
				ClientSecret: v.GetString("OAUTH_WECHAT_CLIENT_SECRET"),
				RedirectURL:  v.GetString("OAUTH_WECHAT_REDIRECT_URL"),
			},
			QQ: OAuthProviderConfig{
				ClientID:     v.GetString("OAUTH_QQ_CLIENT_ID"),
				ClientSecret: v.GetString("OAUTH_QQ_CLIENT_SECRET"),
				RedirectURL:  v.GetString("OAUTH_QQ_REDIRECT_URL"),
			},
		},
		SMTP: SMTPConfig{
			Host:     v.GetString("SMTP_HOST"),
			Port:     v.GetInt("SMTP_PORT"),
			Username: v.GetString("SMTP_USERNAME"),
			Password: v.GetString("SMTP_PASSWORD"),
			From:     v.GetString("SMTP_FROM"),
			UseTLS:   v.GetBool("SMTP_USE_TLS"),
		},
	}

	return cfg, nil
}

// splitAndTrim 分割以逗号分隔的字符串并去除每个元素的空白字符。
// 空元素会被跳过。之所以需要这样做，是因为 viper.GetStringSlice 不会
// 分割以逗号分隔的环境变量/默认值——它会将其视为单个元素。
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
