package request

// CreateRoleParams 是 POST /api/rbac/roles 的请求体。
type CreateRoleParams struct {
	Role        string   `json:"role" binding:"required,min=2,max=64"`
	RoleName    string   `json:"roleName" binding:"required,min=1,max=128"`
	Description string   `json:"description" binding:"max=255"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleParams 是 PUT /api/rbac/roles/:id 的请求体。
type UpdateRoleParams struct {
	RoleName    string `json:"roleName" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"max=255"`
}

// SetRolePermissionsParams 是 PUT /api/rbac/roles/:id/permissions 的请求体。
type SetRolePermissionsParams struct {
	Permissions []string `json:"permissions" binding:"required"`
}
