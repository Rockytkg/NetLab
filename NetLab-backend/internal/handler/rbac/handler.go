package rbac

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/middleware"
	"netlab-backend/internal/service/rbac"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// Handler 处理 RBAC 角色与权限管理端点的 HTTP 请求。
type Handler struct{ svc *rbac.Service }

// NewHandler 创建一个新的 RBAC Handler。
func NewHandler(svc *rbac.Service) *Handler { return &Handler{svc: svc} }

// ListRoles 处理 GET /api/rbac/roles
// @Summary      获取角色列表
// @Description  返回系统中所有角色（含内置角色和自定义角色）
// @Tags         RBAC
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=[]dtoresponse.RoleView}
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/rbac/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context(), middleware.GetUserRole(c))
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
// @Summary      获取角色详情
// @Description  返回指定角色的详细信息及其权限列表
// @Tags         RBAC
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "角色 ID"
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.RoleView}
// @Failure      404  {object}  response.ApiResponse
// @Router       /api/rbac/roles/{id} [get]
func (h *Handler) GetRole(c *gin.Context) {
	if !h.svc.CanManageRole(c.Request.Context(), middleware.GetUserRole(c), c.Param("id")) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	role, err := h.svc.GetRole(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to get role", err))
		return
	}
	if role == nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeUserNotFound, "role not found"))
		return
	}
	perms, err := h.svc.GetRolePermissions(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to get role permissions", err))
		return
	}
	view := dtoresponse.ToRoleView(role)
	view.Permissions = make([]dtoresponse.PermissionRef, len(perms))
	for i := range perms {
		view.Permissions[i] = dtoresponse.ToPermissionRef(&perms[i])
	}
	response.SuccessOK(c, view)
}

// CreateRole 处理 POST /api/rbac/roles
// @Summary      创建角色
// @Description  创建自定义角色并可选分配权限
// @Tags         RBAC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.CreateRoleParams  true  "角色创建参数"
// @Success      201   {object}  response.ApiResponse{data=dtoresponse.RoleView}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/rbac/roles [post]
func (h *Handler) CreateRole(c *gin.Context) {
	var params request.CreateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, err.Error()))
		return
	}
	role, err := h.svc.CreateRole(c.Request.Context(), params.Role, params.RoleName, params.Description, params.Permissions)
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to create role", err))
		return
	}
	response.SuccessCreated(c, dtoresponse.ToRoleView(role))
}

// UpdateRole 处理 PUT /api/rbac/roles/:id
// @Summary      更新角色
// @Description  更新角色的名称和描述（内置角色受管理级别保护）
// @Tags         RBAC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                   true  "角色 ID"
// @Param        body  body      request.UpdateRoleParams true  "角色更新参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      403   {object}  response.ApiResponse
// @Failure      404   {object}  response.ApiResponse
// @Router       /api/rbac/roles/{id} [put]
func (h *Handler) UpdateRole(c *gin.Context) {
	var params request.UpdateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, err.Error()))
		return
	}
	if !h.svc.CanManageRole(c.Request.Context(), middleware.GetUserRole(c), c.Param("id")) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	if err := h.svc.UpdateRole(c.Request.Context(), c.Param("id"), params.RoleName, params.Description); err != nil {
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
// @Summary      删除角色
// @Description  删除自定义角色（内置角色不可删除，仍被用户引用的角色不可删除）
// @Tags         RBAC
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "角色 ID"
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/rbac/roles/{id} [delete]
func (h *Handler) DeleteRole(c *gin.Context) {
	if !h.svc.CanManageRole(c.Request.Context(), middleware.GetUserRole(c), c.Param("id")) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	if err := h.svc.DeleteRole(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to delete role", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "role deleted"})
}

// ListPermissions 处理 GET /api/rbac/permissions
// @Summary      获取权限列表
// @Description  返回系统权限目录中的所有权限项
// @Tags         RBAC
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=[]dtoresponse.PermissionView}
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/rbac/permissions [get]
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
// @Summary      获取角色权限
// @Description  返回指定角色当前拥有的权限列表
// @Tags         RBAC
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "角色 ID"
// @Success      200  {object}  response.ApiResponse{data=[]dtoresponse.PermissionRef}
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/rbac/roles/{id}/permissions [get]
func (h *Handler) GetRolePermissions(c *gin.Context) {
	if !h.svc.CanManageRole(c.Request.Context(), middleware.GetUserRole(c), c.Param("id")) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	perms, err := h.svc.GetRolePermissions(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to get role permissions", err))
		return
	}
	views := make([]dtoresponse.PermissionRef, len(perms))
	for i := range perms {
		views[i] = dtoresponse.ToPermissionRef(&perms[i])
	}
	response.SuccessOK(c, views)
}

// SetRolePermissions 处理 PUT /api/rbac/roles/:id/permissions
// @Summary      设置角色权限
// @Description  全量替换指定角色的权限列表（内置角色受管理级别保护）
// @Tags         RBAC
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                           true  "角色 ID"
// @Param        body  body      request.SetRolePermissionsParams true  "权限键列表"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/rbac/roles/{id}/permissions [put]
func (h *Handler) SetRolePermissions(c *gin.Context) {
	var params request.SetRolePermissionsParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, err.Error()))
		return
	}
	if !h.svc.CanManageRole(c.Request.Context(), middleware.GetUserRole(c), c.Param("id")) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	if err := h.svc.SetRolePermissions(c.Request.Context(), c.Param("id"), params.Permissions); err != nil {
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to set role permissions", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "permissions updated"})
}

// ListAllPermissions 是 ListPermissions 的别名，保留用于路由兼容。
func (h *Handler) ListAllPermissions(c *gin.Context) { h.ListPermissions(c) }
