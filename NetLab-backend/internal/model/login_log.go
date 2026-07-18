package model

import "time"

// 登录方式。
const (
	// LoginTypePassword 用户名密码登录。
	LoginTypePassword = "password"
	// LoginTypeTwoFactor 两步验证（TOTP）登录。
	LoginTypeTwoFactor = "2fa"
	// LoginTypeRecovery 恢复码登录。
	LoginTypeRecovery = "recovery"
	// LoginTypePasskey 通行密钥登录。
	LoginTypePasskey = "passkey"
	// LoginTypeOAuth 第三方 OAuth 登录。
	LoginTypeOAuth = "oauth"
)

// 登录结果状态。
const (
	// LoginStatusSuccess 登录成功。
	LoginStatusSuccess = "success"
	// LoginStatusFailed 登录失败。
	LoginStatusFailed = "failed"
	// LoginStatusPending 密码校验通过、等待两步验证。
	LoginStatusPending = "pending"
)

// LoginLog 表示一条登录日志。
type LoginLog struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	// UserID 允许为 NULL：登录失败时可能无法定位到具体用户。
	UserID      *uint64 `gorm:"index" json:"userId"`
	Username    string  `gorm:"type:varchar(64);not null;default:'';index" json:"username"`
	LoginType   string  `gorm:"type:varchar(20);not null;default:''" json:"loginType"`
	Status      string  `gorm:"type:varchar(20);not null;default:'';index" json:"status"`
	IP          string  `gorm:"type:varchar(45);not null;default:''" json:"ip"`
	UserAgent   string  `gorm:"type:varchar(512);not null;default:''" json:"userAgent"`
	Fingerprint string  `gorm:"type:varchar(128);not null;default:''" json:"fingerprint"`
	// OS 与 Browser 由前端解析后通过请求头上报，服务端原样截断存储。
	OS      string `gorm:"type:varchar(64);not null;default:''" json:"os"`
	Browser string `gorm:"type:varchar(64);not null;default:''" json:"browser"`
	// Location 预留给 IP 归属地解析，暂未填充。
	Location  string    `gorm:"type:varchar(128);not null;default:''" json:"location"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();index" json:"createdAt"`
}

// TableName 指定 LoginLog 的数据库表名。
func (LoginLog) TableName() string { return "nb_login_logs" }
