package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 保存应用程序的所有配置。
//
// SMTP 与 OAuth 等第三方集成配置不再来自环境变量，而是持久化在数据库中
// 并通过「系统设置」管理（见 internal/service/config）。此处仅保留进程
// 启动所必需的基础设施配置。
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
	Radius    RadiusConfig
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
	AccessSecret          string
	RefreshSecret         string
	SigningMode           string
	PrivateKeyPath        string
	PublicKeyPath         string
	AccessExpiry          time.Duration
	RefreshExpiry         time.Duration
	SessionAbsoluteExpiry time.Duration
	Issuer                string
}

// AuthConfig 保存用于保护公开认证端点的预共享签名 key/salt，
// 以及用于加密数据库中敏感配置（SMTP 密码、OAuth 密钥）的主密钥。
type AuthConfig struct {
	SignatureKey  string
	SignatureSalt string
	// EncryptionKey 是 AES-256-GCM 主密钥，用于加密存储在
	// system_configs 表中的敏感字段。未显式配置时回退到 SignatureKey。
	EncryptionKey string
}

// EffectiveEncryptionKey 返回用于加密敏感配置的主密钥。
// 优先使用显式配置的 CONFIG_ENCRYPTION_KEY；未设置时回退到
// 签名 key，以保证在未额外配置的部署中加密功能仍可用。
func (a AuthConfig) EffectiveEncryptionKey() string {
	if a.EncryptionKey != "" {
		return a.EncryptionKey
	}
	return a.SignatureKey
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

// RadiusConfig 保存 RADIUS 认证计费服务配置。
// RADIUS 服务基于 UDP（RadSec 为 TCP/TLS），独立于 HTTP 服务运行。
type RadiusConfig struct {
	// Enabled 是 RADIUS 服务总开关。
	Enabled bool
	// BindHost 是 UDP 监听地址。
	BindHost string
	// AuthPort 是认证端口（RFC 2865，默认 1812）。
	AuthPort int
	// AcctPort 是记账端口（RFC 2866，默认 1813）。
	AcctPort int
	// AcctInterimInterval 是下发给 NAS 的记账间隔秒数，
	// 同时用于僵尸在线会话判定（超过 3 倍间隔未更新即清理）。
	AcctInterimInterval int
	// HistoryDays 是记账记录与认证日志的保留天数，0 表示永久保留。
	HistoryDays int
	// MessageAuthMode 是 Message-Authenticator 校验模式（BlastRADIUS 加固）：
	// disabled=不校验，warn=缺失仅告警，enforce=缺失或错误直接丢弃。
	MessageAuthMode string
	// IgnorePassword 跳过密码校验（调试用；MAC 认证场景之外的开放放行）。
	IgnorePassword bool
	// SessionTimeout 是下发给 NAS 的 Session-Timeout 上限秒数；
	// 0 表示不限制（按用户过期倒计时下发，与 toughradius 一致）。
	SessionTimeout int
	// RejectDelayMaxRejects 是连续拒绝触发临时封禁的阈值。
	RejectDelayMaxRejects int
	// RejectDelayWindowSeconds 是拒绝计数的观察窗口秒数。
	RejectDelayWindowSeconds int
	// EAP 相关：启用 802.1X（EAP-MD5/MSCHAPv2/TLS、PEAP、TTLS）。
	EAPEnabled  bool
	EAPMethod   string
	EAPCertFile string
	EAPKeyFile  string
	EAPCAFile   string
	// EAPEnabledHandlers 是允许使用的 EAP 方法列表（逗号分隔，"*" 为全部）。
	EAPEnabledHandlers string
	// EAPTLSMinVersion 是 EAP 隧道最低 TLS 版本（"1.2" 或 "1.3"）。
	EAPTLSMinVersion string
	// EAPServerCertID / EAPClientCACertID 引用 nb_radius_certs 中的证书；
	// 为 0 时回退到 EAPCertFile/EAPCAFile 文件路径。
	EAPServerCertID   uint64
	EAPClientCACertID uint64
	// RadSec 相关：RADIUS over TLS（RFC 6614，默认 2083，双向 TLS）。
	RadsecEnabled  bool
	RadsecPort     int
	RadsecCertFile string
	RadsecKeyFile  string
	RadsecCAFile   string
	// RadsecCertID / RadsecCACertID 引用 nb_radius_certs 中的证书；
	// 为 0 时回退到文件路径。
	RadsecCertID   uint64
	RadsecCACertID uint64
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
	v.SetDefault("JWT_SIGNING_MODE", "HS256")
	v.SetDefault("JWT_PRIVATE_KEY_PATH", "")
	v.SetDefault("JWT_PUBLIC_KEY_PATH", "")
	v.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	v.SetDefault("JWT_SESSION_ABSOLUTE_EXPIRY", "720h")
	v.SetDefault("JWT_ISSUER", "netlab")

	v.SetDefault("AUTH_SIGNATURE_KEY", "")
	v.SetDefault("AUTH_SIGNATURE_SALT", "")
	v.SetDefault("CONFIG_ENCRYPTION_KEY", "")

	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("RATE_LIMIT_GLOBAL", 100)

	v.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	v.SetDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	v.SetDefault("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,X-Request-Id,X-Signature,X-Timestamp,Accept-Language,X-User-Language,X-Browser-Fingerprint,X-Client-OS,X-Client-Browser")

	// 验证码渲染尺寸。前端在登录框内以 32px 高度展示该图片，
	// 因此源图按 2 倍（64px）渲染，可在高分屏上获得清晰的整数倍缩放；
	// 宽度按 3:1 比例取 192px，保证算式有充足的水平空间。
	v.SetDefault("CAPTCHA_WIDTH", 192)
	v.SetDefault("CAPTCHA_HEIGHT", 64)
	v.SetDefault("CAPTCHA_MAX_RETRIES", 5)

	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	v.SetDefault("RADIUS_ENABLED", false)
	v.SetDefault("RADIUS_BIND_HOST", "0.0.0.0")
	v.SetDefault("RADIUS_AUTH_PORT", 1812)
	v.SetDefault("RADIUS_ACCT_PORT", 1813)
	v.SetDefault("RADIUS_ACCT_INTERIM_INTERVAL", 120)
	v.SetDefault("RADIUS_HISTORY_DAYS", 90)
	v.SetDefault("RADIUS_MESSAGE_AUTH_MODE", "warn")
	v.SetDefault("RADIUS_IGNORE_PASSWORD", false)
	v.SetDefault("RADIUS_SESSION_TIMEOUT", 0)
	v.SetDefault("RADIUS_REJECT_DELAY_MAX_REJECTS", 7)
	v.SetDefault("RADIUS_REJECT_DELAY_WINDOW_SECONDS", 10)
	v.SetDefault("RADIUS_EAP_ENABLED", false)
	v.SetDefault("RADIUS_EAP_METHOD", "eap-md5")
	v.SetDefault("RADIUS_EAP_CERT_FILE", "")
	v.SetDefault("RADIUS_EAP_KEY_FILE", "")
	v.SetDefault("RADIUS_EAP_CA_FILE", "")
	v.SetDefault("RADIUS_EAP_ENABLED_HANDLERS", "*")
	v.SetDefault("RADIUS_EAP_TLS_MIN_VERSION", "1.2")
	v.SetDefault("RADIUS_RADSEC_ENABLED", false)
	v.SetDefault("RADIUS_RADSEC_PORT", 2083)
	v.SetDefault("RADIUS_RADSEC_CERT_FILE", "")
	v.SetDefault("RADIUS_RADSEC_KEY_FILE", "")
	v.SetDefault("RADIUS_RADSEC_CA_FILE", "")

	// 从 .env 读取
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	_ = v.ReadInConfig() // 如果 .env 不存在则忽略错误

	// 解析时长
	readTimeout := durationOrDefault(v.GetString("SERVER_READ_TIMEOUT"), 30*time.Second)
	writeTimeout := durationOrDefault(v.GetString("SERVER_WRITE_TIMEOUT"), 30*time.Second)
	accessExpiry := durationOrDefault(v.GetString("JWT_ACCESS_EXPIRY"), 15*time.Minute)
	refreshExpiry := durationOrDefault(v.GetString("JWT_REFRESH_EXPIRY"), 7*24*time.Hour)
	sessionAbsoluteExpiry := durationOrDefault(v.GetString("JWT_SESSION_ABSOLUTE_EXPIRY"), 30*24*time.Hour)
	connMaxLifetime := durationOrDefault(v.GetString("DB_CONN_MAX_LIFETIME"), 5*time.Minute)

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
			AccessSecret:          v.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret:         v.GetString("JWT_REFRESH_SECRET"),
			SigningMode:           strings.ToUpper(strings.TrimSpace(v.GetString("JWT_SIGNING_MODE"))),
			PrivateKeyPath:        v.GetString("JWT_PRIVATE_KEY_PATH"),
			PublicKeyPath:         v.GetString("JWT_PUBLIC_KEY_PATH"),
			AccessExpiry:          accessExpiry,
			RefreshExpiry:         refreshExpiry,
			SessionAbsoluteExpiry: sessionAbsoluteExpiry,
			Issuer:                v.GetString("JWT_ISSUER"),
		},
		Auth: AuthConfig{
			SignatureKey:  v.GetString("AUTH_SIGNATURE_KEY"),
			SignatureSalt: v.GetString("AUTH_SIGNATURE_SALT"),
			EncryptionKey: v.GetString("CONFIG_ENCRYPTION_KEY"),
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
		Radius: RadiusConfig{
			Enabled:                  v.GetBool("RADIUS_ENABLED"),
			BindHost:                 v.GetString("RADIUS_BIND_HOST"),
			AuthPort:                 v.GetInt("RADIUS_AUTH_PORT"),
			AcctPort:                 v.GetInt("RADIUS_ACCT_PORT"),
			AcctInterimInterval:      v.GetInt("RADIUS_ACCT_INTERIM_INTERVAL"),
			HistoryDays:              v.GetInt("RADIUS_HISTORY_DAYS"),
			MessageAuthMode:          strings.ToLower(strings.TrimSpace(v.GetString("RADIUS_MESSAGE_AUTH_MODE"))),
			IgnorePassword:           v.GetBool("RADIUS_IGNORE_PASSWORD"),
			SessionTimeout:           v.GetInt("RADIUS_SESSION_TIMEOUT"),
			RejectDelayMaxRejects:    v.GetInt("RADIUS_REJECT_DELAY_MAX_REJECTS"),
			RejectDelayWindowSeconds: v.GetInt("RADIUS_REJECT_DELAY_WINDOW_SECONDS"),
			EAPEnabled:               v.GetBool("RADIUS_EAP_ENABLED"),
			EAPMethod:                v.GetString("RADIUS_EAP_METHOD"),
			EAPCertFile:              v.GetString("RADIUS_EAP_CERT_FILE"),
			EAPKeyFile:               v.GetString("RADIUS_EAP_KEY_FILE"),
			EAPCAFile:                v.GetString("RADIUS_EAP_CA_FILE"),
			EAPEnabledHandlers:       v.GetString("RADIUS_EAP_ENABLED_HANDLERS"),
			EAPTLSMinVersion:         v.GetString("RADIUS_EAP_TLS_MIN_VERSION"),
			RadsecEnabled:            v.GetBool("RADIUS_RADSEC_ENABLED"),
			RadsecPort:               v.GetInt("RADIUS_RADSEC_PORT"),
			RadsecCertFile:           v.GetString("RADIUS_RADSEC_CERT_FILE"),
			RadsecKeyFile:            v.GetString("RADIUS_RADSEC_KEY_FILE"),
			RadsecCAFile:             v.GetString("RADIUS_RADSEC_CA_FILE"),
		},
	}

	if cfg.JWT.AccessSecret == "" || cfg.JWT.RefreshSecret == "" {
		return nil, fmt.Errorf("JWT access and refresh secrets must be non-empty")
	}
	if cfg.JWT.AccessSecret == cfg.JWT.RefreshSecret {
		return nil, fmt.Errorf("JWT access and refresh secrets must be different")
	}
	if cfg.JWT.SigningMode != "HS256" && cfg.JWT.SigningMode != "RS256" {
		return nil, fmt.Errorf("unsupported JWT signing mode: %s", cfg.JWT.SigningMode)
	}
	if cfg.JWT.SigningMode == "RS256" && (cfg.JWT.PrivateKeyPath == "" || cfg.JWT.PublicKeyPath == "") {
		return nil, fmt.Errorf("JWT RS256 signing requires private and public key paths")
	}

	if cfg.Radius.Enabled {
		switch cfg.Radius.MessageAuthMode {
		case "disabled", "warn", "enforce":
		default:
			return nil, fmt.Errorf("RADIUS_MESSAGE_AUTH_MODE must be one of: disabled | warn | enforce")
		}
		if cfg.Radius.AcctInterimInterval < 30 {
			return nil, fmt.Errorf("RADIUS_ACCT_INTERIM_INTERVAL must be >= 30 seconds")
		}
		if cfg.Radius.SessionTimeout < 0 {
			return nil, fmt.Errorf("RADIUS_SESSION_TIMEOUT must be >= 0 (0 means no cap)")
		}
		// EAP/RadSec 的证书可来自环境变量文件路径或管理端维护的 DB 证书
		// （nb_radius_certs，运行时选择），因此此处不强制证书文件存在；
		// 缺证书时对应的 TLS 方法/监听器会在运行期安全降级或报错。
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

func durationOrDefault(value string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
