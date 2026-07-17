package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

// UserRole 表示用户在系统中的角色。
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleEditor     UserRole = "editor"
	RoleViewer     UserRole = "viewer"
)

// UserStatus 表示账户状态。
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusDisabled UserStatus = "disabled"
	StatusLocked   UserStatus = "locked"
)

// User 表示一个用户账户。
type User struct {
	ID                  uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username            string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Nickname            string     `gorm:"type:varchar(64);not null" json:"nickname"`
	Phone               string     `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
	Email               string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash        string     `gorm:"type:varchar(255);not null" json:"-"`
	Avatar              string     `gorm:"type:varchar(512)" json:"avatar"`
	Role                UserRole   `gorm:"type:varchar(32);not null;default:'viewer'" json:"role"`
	Status              UserStatus `gorm:"type:varchar(16);not null;default:'active'" json:"status"`
	ForcePasswordChange bool       `gorm:"type:boolean;not null;default:false" json:"forcePasswordChange"`
	ForceEmailChange    bool       `gorm:"type:boolean;not null;default:false" json:"forceEmailChange"`
	TwoFactorEnabled    bool       `gorm:"type:boolean;not null;default:false" json:"twoFactorEnabled"`
	TwoFactorSecret     string     `gorm:"type:varchar(512)" json:"-"`
	PreferredAuthMethod string     `gorm:"type:varchar(16);not null;default:'totp'" json:"preferredAuthMethod"`

	// ── 恢复码 ──
	RecoveryCodes     []string   `gorm:"type:jsonb;default:'[]'" json:"-"`
	PasswordChangedAt *time.Time `gorm:"type:timestamptz" json:"passwordChangedAt"`
	CreatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (User) TableName() string { return "nb_users" }

// BeforeCreate 在创建前设置默认值。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = RoleViewer
	}
	if u.Status == "" {
		u.Status = StatusActive
	}
	if u.RecoveryCodes == nil {
		u.RecoveryCodes = []string{}
	}
	return nil
}

func (u *User) IsActive() bool     { return u.Status == StatusActive }
func (u *User) IsLocked() bool     { return u.Status == StatusLocked }
func (u *User) IsPrivileged() bool { return u.Role == RoleSuperAdmin || u.Role == RoleAdmin }
func (u *User) IsSuperAdmin() bool { return u.Role == RoleSuperAdmin }

// ── TokenUser interface ──────────────────────────────────────────────────

func (u *User) GetID() string       { return strconv.FormatUint(u.ID, 10) }
func (u *User) GetUsername() string { return u.Username }
func (u *User) GetRole() string     { return string(u.Role) }

// ─── 恢复码 ──────────────────────────────────────────────────────────

func (u *User) ConsumeRecoveryCode(codeHash string) bool {
	for i, h := range u.RecoveryCodes {
		if h == codeHash {
			u.RecoveryCodes = append(u.RecoveryCodes[:i], u.RecoveryCodes[i+1:]...)
			return true
		}
	}
	return false
}
