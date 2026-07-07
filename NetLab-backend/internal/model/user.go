package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole 表示用户在系统中的角色。
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
	RoleViewer UserRole = "viewer"
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
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username            string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email               string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash        string     `gorm:"type:varchar(255);not null" json:"-"`
	Avatar              string     `gorm:"type:varchar(512)" json:"avatar"`
	Roles               []string   `gorm:"type:jsonb;serializer:json;not null;default:'[\"viewer\"]'" json:"roles"`
	Status              UserStatus `gorm:"type:varchar(16);not null;default:'active'" json:"status"`
	FailedLoginAttempts int        `gorm:"type:int;default:0" json:"-"`
	LockedUntil         *time.Time `gorm:"type:timestamptz" json:"-"`
	LastLoginAt         *time.Time `gorm:"type:timestamptz" json:"last_login_at"`
	CreatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}

// BeforeCreate 在 UUID 未设置时进行初始化。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Roles == nil {
		u.Roles = []string{string(RoleViewer)}
	}
	if u.Status == "" {
		u.Status = StatusActive
	}
	return nil
}

// IsActive 在账户状态正常时返回 true。
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// IsLocked 在账户当前处于锁定状态时返回 true。
func (u *User) IsLocked() bool {
	if u.Status == StatusLocked && u.LockedUntil != nil {
		return time.Now().Before(*u.LockedUntil)
	}
	return false
}
