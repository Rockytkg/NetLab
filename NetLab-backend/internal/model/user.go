package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// RecoveryCodes 是用户恢复码哈希列表的 JSONB 表示。
// PostgreSQL 驱动以 []byte 返回 JSONB 值，因此该类型必须显式实现
// sql.Scanner，而不能依赖 GORM 默认的切片映射。
type RecoveryCodes []string

// Scan 实现 sql.Scanner，将数据库中的 JSONB 值解码为恢复码列表。
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

// Value 实现 driver.Valuer，将恢复码列表编码为 JSONB 存储值。
func (r RecoveryCodes) Value() (driver.Value, error) {
	if r == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(r))
}

// UserRole 表示用户在系统中的角色。
type UserRole string

const (
	// RoleSuperAdmin 超级管理员角色标识。
	RoleSuperAdmin UserRole = "superadmin"
	// RoleAdmin 管理员角色标识。
	RoleAdmin UserRole = "admin"
)

// UserStatus 表示账户状态。
type UserStatus string

const (
	// StatusActive 账户正常可用。
	StatusActive UserStatus = "active"
	// StatusDisabled 账户已被管理员禁用。
	StatusDisabled UserStatus = "disabled"
	// StatusLocked 账户已被锁定（如多次登录失败）。
	StatusLocked UserStatus = "locked"
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
	RoleName            string     `gorm:"-" json:"-"`
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

// TableName 指定 User 的数据库表名。
func (User) TableName() string { return "nb_users" }

// BeforeCreate 在创建前设置默认值。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.RoleID == 0 {
		role := u.Role
		if role == "" {
			role = UserRole("viewer")
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

// AfterFind 在查询后按 RoleID 回填角色标识（Role 字段不落库）。
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

// IsActive 报告账户是否处于正常可用状态。
func (u *User) IsActive() bool { return u.Status == StatusActive }

// ── TokenUser 接口实现 ───────────────────────────────────────────────────

// GetID 返回用户 ID 的字符串形式（用于 JWT 载荷）。
func (u *User) GetID() string { return strconv.FormatUint(u.ID, 10) }

// GetUsername 返回用户名（用于 JWT 载荷）。
func (u *User) GetUsername() string { return u.Username }

// GetRole 返回角色 ID 的字符串形式（用于 JWT 载荷与 RBAC 鉴权）。
func (u *User) GetRole() string { return strconv.FormatUint(u.RoleID, 10) }

// ─── 恢复码 ──────────────────────────────────────────────────────────

// ConsumeRecoveryCode 消费一个恢复码：若哈希匹配则从列表中移除并返回 true。
// 调用方需在返回 true 后持久化用户以完成一次性消费。
func (u *User) ConsumeRecoveryCode(codeHash string) bool {
	for i, h := range u.RecoveryCodes {
		if h == codeHash {
			u.RecoveryCodes = append(u.RecoveryCodes[:i], u.RecoveryCodes[i+1:]...)
			return true
		}
	}
	return false
}
