package response

import (
	"strconv"

	"netlab-backend/internal/model"
)

// RoleView 是角色的 API 视图。
type RoleView struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Permissions []PermissionRef `json:"permissions,omitempty"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

// PermissionRef 是权限的简洁引用。
type PermissionRef struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// PermissionView 是权限的详细视图。
type PermissionView struct {
	ID          string `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// ─── 转换 ──────────────────────────────────────────────────────────────────

func ToRoleView(r *model.Role) RoleView {
	return RoleView{
		ID:          strconv.FormatUint(r.ID, 10),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToPermissionRef(p *model.Permission) PermissionRef {
	return PermissionRef{
		ID:       strconv.FormatUint(p.ID, 10),
		Resource: p.Resource,
		Action:   p.Action,
	}
}

func ToPermissionView(p *model.Permission) PermissionView {
	return PermissionView{
		ID:          strconv.FormatUint(p.ID, 10),
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
