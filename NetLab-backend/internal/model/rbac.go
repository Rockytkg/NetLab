package model

import (
	"time"
)

// Role 表示一个角色定义。
type Role struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (Role) TableName() string { return "nb_roles" }

// Permission 表示一个资源:操作的细粒度权限项。
type Permission struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Resource    string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_resource_action" json:"resource"`
	Action      string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_perm_resource_action" json:"action"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
}

func (Permission) TableName() string { return "nb_permissions" }

// RolePermission 是角色与权限的多对多关联表。
type RolePermission struct {
	RoleID       uint64    `gorm:"not null;uniqueIndex:idx_rp_role_perm" json:"roleId"`
	PermissionID uint64    `gorm:"not null;uniqueIndex:idx_rp_role_perm" json:"permissionId"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
}

func (RolePermission) TableName() string { return "nb_role_permissions" }
