package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// RecoveryCodes is the JSONB representation of a user's hashed recovery codes.
// PostgreSQL drivers return JSONB values as []byte, so the type must explicitly
// implement sql.Scanner instead of relying on GORM's default slice mapping.
type RecoveryCodes []string

func (r *RecoveryCodes) Scan(value any) error {
	if value == nil {
		*r = RecoveryCodes{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan recovery codes from %T", value)
	}

	if len(data) == 0 {
		*r = RecoveryCodes{}
		return nil
	}
	var codes []string
	if err := json.Unmarshal(data, &codes); err != nil {
		return fmt.Errorf("decode recovery codes: %w", err)
	}
	if codes == nil {
		codes = []string{}
	}
	*r = RecoveryCodes(codes)
	return nil
}

func (r RecoveryCodes) Value() (driver.Value, error) {
	if r == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(r))
}

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
	RoleID              uint64     `gorm:"column:role_id;not null;index" json:"roleId"`
	Role                UserRole   `gorm:"-" json:"-"`
	Status              UserStatus `gorm:"type:varchar(16);not null;default:'active'" json:"status"`
	ForcePasswordChange bool       `gorm:"type:boolean;not null;default:false" json:"forcePasswordChange"`
	ForceEmailChange    bool       `gorm:"type:boolean;not null;default:false" json:"forceEmailChange"`
	TwoFactorEnabled    bool       `gorm:"type:boolean;not null;default:false" json:"twoFactorEnabled"`
	TwoFactorSecret     string     `gorm:"type:varchar(512)" json:"-"`
	PreferredAuthMethod string     `gorm:"type:varchar(16);not null;default:'totp'" json:"preferredAuthMethod"`

	// ── 恢复码 ──
	RecoveryCodes     RecoveryCodes `gorm:"type:jsonb;default:'[]'" json:"-"`
	PasswordChangedAt *time.Time    `gorm:"type:timestamptz" json:"passwordChangedAt"`
	CreatedAt         time.Time     `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt         time.Time     `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (User) TableName() string { return "nb_users" }

// BeforeCreate 在创建前设置默认值。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.RoleID == 0 {
		role := u.Role
		if role == "" {
			role = RoleViewer
		}
		var roleModel Role
		if err := tx.Where("role = ?", role).First(&roleModel).Error; err != nil {
			return err
		}
		u.RoleID = roleModel.ID
		u.Role = UserRole(roleModel.Role)
	}
	if u.Status == "" {
		u.Status = StatusActive
	}
	if u.RecoveryCodes == nil {
		u.RecoveryCodes = RecoveryCodes{}
	}
	return nil
}

func (u *User) AfterFind(tx *gorm.DB) error {
	if u.RoleID == 0 {
		return nil
	}
	var role Role
	if err := tx.Select("id, role").First(&role, u.RoleID).Error; err != nil {
		return err
	}
	u.Role = UserRole(role.Role)
	return nil
}

func (u *User) IsActive() bool { return u.Status == StatusActive }

// ── TokenUser interface ──────────────────────────────────────────────────

func (u *User) GetID() string       { return strconv.FormatUint(u.ID, 10) }
func (u *User) GetUsername() string { return u.Username }
func (u *User) GetRole() string     { return strconv.FormatUint(u.RoleID, 10) }

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
