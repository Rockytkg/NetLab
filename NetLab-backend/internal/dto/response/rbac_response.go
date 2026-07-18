package response

import (
	"strconv"

	"netlab-backend/internal/model"
)

// RoleView 是角色的 API 视图。
type RoleView struct {
	ID              string          `json:"id"`
	Role            string          `json:"role"`
	RoleName        string          `json:"roleName"`
	Description     string          `json:"description,omitempty"`
	Type            string          `json:"type"`
	ManagementLevel int             `json:"managementLevel"`
	Hidden          bool            `json:"hidden"`
	Permissions     []PermissionRef `json:"permissions,omitempty"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// PermissionRef 是权限的简洁引用。
type PermissionRef struct {
	Code     string `json:"code"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// PermissionView 是权限的详细视图。
type PermissionView struct {
	Code        string `json:"code"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// ─── 转换 ──────────────────────────────────────────────────────────────────

// ToRoleView 将 Role 模型转换为 API 视图（不含权限列表，由调用方按需填充）。
func ToRoleView(r *model.Role) RoleView {
	return RoleView{
		ID:              strconv.FormatUint(r.ID, 10),
		Role:            r.Role,
		RoleName:        r.RoleName,
		Description:     r.Description,
		Type:            string(r.RoleType),
		ManagementLevel: r.ManagementLevel,
		Hidden:          r.Hidden,
		CreatedAt:       r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToPermissionRef 将 Permission 模型转换为简洁引用（code 为 resource.action 格式）。
func ToPermissionRef(p *model.Permission) PermissionRef {
	return PermissionRef{
		Code:     p.Resource + "." + p.Action,
		Resource: p.Resource,
		Action:   p.Action,
	}
}

// ToPermissionView 将 Permission 模型转换为详细视图。
func ToPermissionView(p *model.Permission) PermissionView {
	return PermissionView{
		Code:        p.Resource + "." + p.Action,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
