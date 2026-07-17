package request

// CreateRoleParams 是 POST /api/rbac/roles 的请求体。
type CreateRoleParams struct {
	Name        string `json:"name" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
}

// UpdateRoleParams 是 PUT /api/rbac/roles/:id 的请求体。
type UpdateRoleParams struct {
	Name        string `json:"name" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
}

// SetRolePermissionsParams 是 PUT /api/rbac/roles/:id/permissions 的请求体。
type SetRolePermissionsParams struct {
	PermissionIDs []string `json:"permissionIds" binding:"required"`
}
