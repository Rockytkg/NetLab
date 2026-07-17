package rbac

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/service/rbac"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// Handler 处理 RBAC 管理端点。
type Handler struct {
	svc *rbac.Service
}

// NewHandler 创建 RBAC 管理 Handler。
func NewHandler(svc *rbac.Service) *Handler {
	return &Handler{svc: svc}
}

// ─── 角色管理 ───────────────────────────────────────────────────────────────

// ListRoles 处理 GET /api/rbac/roles
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list roles", err))
		return
	}
	views := make([]dtoresponse.RoleView, len(roles))
	for i := range roles {
		views[i] = dtoresponse.ToRoleView(&roles[i])
	}
	response.SuccessOK(c, views)
}

// GetRole 处理 GET /api/rbac/roles/:id
func (h *Handler) GetRole(c *gin.Context) {
	role, err := h.svc.GetRole(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to get role", err))
		return
	}
	if role == nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeUserNotFound, "role not found"))
		return
	}
	view := dtoresponse.ToRoleView(role)

	// 附加权限列表
	permIDs, err := h.svc.GetRolePermissionIDs(c.Request.Context(), strconv.FormatUint(role.ID, 10))
	if err == nil {
		perms, listErr := h.svc.ListPermissions(c.Request.Context())
		if listErr == nil {
			for _, p := range perms {
				for _, pid := range permIDs {
					if strconv.FormatUint(p.ID, 10) == pid {
						ref := dtoresponse.ToPermissionRef(&p)
						view.Permissions = append(view.Permissions, ref)
						break
					}
				}
			}
		}
	}

	response.SuccessOK(c, view)
}

// CreateRole 处理 POST /api/rbac/roles
func (h *Handler) CreateRole(c *gin.Context) {
	var params request.CreateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), params.Name, params.Description)
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to create role", err))
		return
	}
	response.SuccessCreated(c, dtoresponse.ToRoleView(role))
}

// UpdateRole 处理 PUT /api/rbac/roles/:id
func (h *Handler) UpdateRole(c *gin.Context) {
	var params request.UpdateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if err := h.svc.UpdateRole(c.Request.Context(), c.Param("id"), params.Name, params.Description); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, apperrors.New(apperrors.ErrCodeUserNotFound, "role not found"))
			return
		}
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update role", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "role updated"})
}

// DeleteRole 处理 DELETE /api/rbac/roles/:id
func (h *Handler) DeleteRole(c *gin.Context) {
	if err := h.svc.DeleteRole(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to delete role", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "role deleted"})
}

// ─── 权限管理 ───────────────────────────────────────────────────────────────

// ListPermissions 处理 GET /api/rbac/permissions
func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list permissions", err))
		return
	}
	views := make([]dtoresponse.PermissionView, len(perms))
	for i := range perms {
		views[i] = dtoresponse.ToPermissionView(&perms[i])
	}
	response.SuccessOK(c, views)
}

// GetRolePermissions 处理 GET /api/rbac/roles/:id/permissions
func (h *Handler) GetRolePermissions(c *gin.Context) {
	permIDs, err := h.svc.GetRolePermissionIDs(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to get role permissions", err))
		return
	}

	allPerms, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list permissions", err))
		return
	}

	permSet := make(map[string]bool, len(permIDs))
	for _, id := range permIDs {
		permSet[id] = true
	}

	views := make([]dtoresponse.PermissionRef, 0, len(permIDs))
	for _, p := range allPerms {
		if permSet[strconv.FormatUint(p.ID, 10)] {
			views = append(views, dtoresponse.ToPermissionRef(&p))
		}
	}
	response.SuccessOK(c, views)
}

// SetRolePermissions 处理 PUT /api/rbac/roles/:id/permissions
func (h *Handler) SetRolePermissions(c *gin.Context) {
	var params request.SetRolePermissionsParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if err := h.svc.SetRolePermissions(c.Request.Context(), c.Param("id"), params.PermissionIDs); err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to set role permissions", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "permissions updated"})
}

// ListAllPermissions 处理 GET /api/rbac/permissions 的别名。
func (h *Handler) ListAllPermissions(c *gin.Context) {
	h.ListPermissions(c)
}
