package model

import "time"

// RoleType 表示角色类型：内置角色受系统保护，自定义角色可自由管理。
type RoleType string

const (
	// RoleTypeBuiltin 内置角色（superadmin、admin），不可删除。
	RoleTypeBuiltin RoleType = "builtin"
	// RoleTypeCustom 用户自定义角色，可增删改。
	RoleTypeCustom RoleType = "custom"
)

// Role 表示一个角色定义。
type Role struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Role            string    `gorm:"column:role;type:varchar(64);uniqueIndex;not null" json:"role"`
	RoleName        string    `gorm:"column:role_name;type:varchar(128);not null" json:"roleName"`
	Description     string    `gorm:"type:varchar(255)" json:"description"`
	RoleType        RoleType  `gorm:"column:role_type;type:varchar(16);not null;default:'custom'" json:"type"`
	ManagementLevel int       `gorm:"column:management_level;not null;default:0;index" json:"managementLevel"`
	Hidden          bool      `gorm:"column:is_hidden;not null;default:false" json:"hidden"`
	Version         uint64    `gorm:"not null;default:1" json:"version"`
	CreatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

// TableName 指定 Role 的数据库表名。
func (Role) TableName() string { return "nb_roles" }

// Permission 表示一个资源:操作的细粒度权限项。
type Permission struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Resource    string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_resource_action" json:"resource"`
	Action      string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_resource_action" json:"action"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	Builtin     bool      `gorm:"not null;default:true" json:"builtin"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
}

// TableName 指定 Permission 的数据库表名。
func (Permission) TableName() string { return "nb_permissions" }

// RolePermission 是角色与权限的多对多关联表。
type RolePermission struct {
	RoleID       uint64    `gorm:"primaryKey;not null" json:"roleId"`
	PermissionID uint64    `gorm:"primaryKey;not null" json:"permissionId"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
}

// TableName 指定 RolePermission 的数据库表名。
func (RolePermission) TableName() string { return "nb_role_permissions" }
