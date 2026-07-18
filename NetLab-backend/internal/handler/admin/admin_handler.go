package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"netlab-backend/internal/contextkeys"
	"netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/mailer"
	"netlab-backend/internal/middleware"
	authsvc "netlab-backend/internal/service/auth"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// AdminHandler 处理系统设置与用户资源端点，访问由 RBAC 权限控制。
type AdminHandler struct {
	adminService     *authsvc.AdminService
	userAdminService *authsvc.UserAdminService
	importService    *authsvc.UserImportService
	mailer           *mailer.Provider
	logger           *zap.Logger
}

// NewAdminHandler 创建一个新的 AdminHandler。
func NewAdminHandler(adminService *authsvc.AdminService, userAdminService *authsvc.UserAdminService, importService *authsvc.UserImportService, mailerProvider *mailer.Provider, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		adminService:     adminService,
		userAdminService: userAdminService,
		importService:    importService,
		mailer:           mailerProvider,
		logger:           logger,
	}
}

// GetSettings 处理 GET /api/settings
// @Summary      Get system settings
// @Description  Return all system settings (secrets masked). Admin only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/settings [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.adminService.GetSettings(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, settings)
}

// UpdateSecurity 处理 PUT /api/settings/security
// @Summary      更新安全策略
// @Description  更新系统安全配置（注册开关、验证码、密码策略等）
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateSecurityParams  true  "安全策略参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/settings/security [put]
func (h *AdminHandler) UpdateSecurity(c *gin.Context) {
	var params request.UpdateSecurityParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}

	if err := h.adminService.UpdateSecurity(c.Request.Context(), sysconfig.SecuritySettings{
		RegistrationEnabled:  params.RegistrationEnabled,
		CaptchaEnabled:       params.CaptchaEnabled,
		PasskeyEnabled:       params.PasskeyEnabled,
		PasswordResetEnabled: params.PasswordResetEnabled,
		PasswordMaxAgeDays:   params.PasswordMaxAgeDays,
	}); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "security settings updated"})
}

// UpdateBeian 处理 PUT /api/settings/beian
// @Summary      更新备案信息
// @Description  更新 ICP 备案号和公安备案号
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateBeianParams  true  "备案信息参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/settings/beian [put]
func (h *AdminHandler) UpdateBeian(c *gin.Context) {
	var params request.UpdateBeianParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}

	if err := h.adminService.UpdateBeian(c.Request.Context(), sysconfig.BeianSettings{
		ICPBeian:    params.ICPBeian,
		PoliceBeian: params.PoliceBeian,
	}); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "filing information updated"})
}

// UpdateSMTP 处理 PUT /api/settings/smtp
// @Summary      更新 SMTP 邮件配置
// @Description  更新 SMTP 邮件服务器配置（密码将使用 AES-256-GCM 加密存储）
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateSMTPParams  true  "SMTP 配置参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/settings/smtp [put]
func (h *AdminHandler) UpdateSMTP(c *gin.Context) {
	var params request.UpdateSMTPParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}

	if err := h.adminService.UpdateSMTP(c.Request.Context(), sysconfig.SMTPSettings{
		Enabled:  params.Enabled,
		Host:     params.Host,
		Port:     params.Port,
		Username: params.Username,
		Password: params.Password,
		From:     params.From,
		UseTLS:   params.UseTLS,
	}); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "smtp settings updated"})
}

// TestSMTP 处理 POST /api/settings/smtp/test
// @Summary      测试 SMTP 连接
// @Description  向指定邮箱发送测试邮件以验证 SMTP 配置是否有效
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.TestSMTPParams  true  "测试邮件参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      503   {object}  response.ApiResponse
// @Router       /api/settings/smtp/test [post]
func (h *AdminHandler) TestSMTP(c *gin.Context) {
	var params request.TestSMTPParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}

	locale := contextkeys.GetLocale(c)
	if err := h.mailer.SendTestFromStored(c.Request.Context(), params.To, locale); err != nil {
		h.logger.Warn("smtp test failed", zap.Error(err))
		response.Error(c, apperrors.Wrap(apperrors.ErrCodeEmailSendFailed, "failed to send test email", err))
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "test email sent"})
}

// UpdateOAuthProvider 处理 PUT /api/settings/oauth/:provider
// @Summary      更新 OAuth 提供商配置
// @Description  更新指定 OAuth 提供商的客户端 ID、密钥和回调地址（密钥使用 AES-256-GCM 加密存储）
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        provider  path      string                    true  "OAuth 提供商 ID（如 github、google）"
// @Param        body      body      request.UpdateOAuthParams true  "OAuth 配置参数"
// @Success      200       {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400       {object}  response.ApiResponse
// @Failure      403       {object}  response.ApiResponse
// @Router       /api/settings/oauth/{provider} [put]
func (h *AdminHandler) UpdateOAuthProvider(c *gin.Context) {
	provider := c.Param("provider")
	var params request.UpdateOAuthParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}

	if err := h.adminService.UpdateOAuthProvider(c.Request.Context(), provider, sysconfig.ProviderSettings{
		Enabled:      params.Enabled,
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		RedirectURL:  params.RedirectURL,
	}); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "oauth provider updated"})
}

// ─── 用户管理 ────────────────────────────────────────────────────────

// ListUsers 处理 GET /api/users
// @Summary      获取用户列表
// @Description  分页查询用户，支持按关键词、状态、角色筛选
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        page     query     int     false  "页码"           default(1)
// @Param        size     query     int     false  "每页数量"        default(20)
// @Param        keyword  query     string  false  "搜索关键词（用户名/邮箱/手机号）"
// @Param        status   query     string  false  "用户状态（active/disabled/locked）"
// @Param        role     query     string  false  "角色标识"
// @Success      200      {object}  response.ApiResponse
// @Failure      403      {object}  response.ApiResponse
// @Router       /api/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	keyword := c.Query("keyword")
	status := c.Query("status")
	role := c.Query("role")

	result, err := h.userAdminService.ListUsers(c.Request.Context(), page, size, keyword, status, role)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, result)
}

// CreateUser 处理 POST /api/users
// @Summary      创建用户
// @Description  管理员创建新用户账号
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.CreateUserParams  true  "用户创建参数"
// @Success      201   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Failure      409   {object}  response.ApiResponse
// @Router       /api/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var params request.CreateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if !h.userAdminService.CanAssignRole(c.Request.Context(), middleware.GetUserRole(c), params.Role) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	user, err := h.userAdminService.CreateUser(c.Request.Context(), params.Username, params.Nickname, params.Phone, params.Email, params.Role, params.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, user)
}

// UpdateUser 处理 PUT /api/users/:id
// @Summary      更新用户信息
// @Description  更新指定用户的昵称、手机号、邮箱、角色和状态
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "用户 ID"
// @Param        body  body      request.UpdateUserParams true  "用户更新参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	var params request.UpdateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if !h.userAdminService.CanUpdateUser(c.Request.Context(), middleware.GetUserID(c), middleware.GetUserRole(c), c.Param("id"), params.Role, params.Status, params.DisableTwoFactor) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	if err := h.userAdminService.UpdateUser(c.Request.Context(), c.Param("id"), params.Nickname, params.Phone, params.Email, params.Role, params.Status, params.DisableTwoFactor); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "user updated"})
}

// BatchUpdateRole 处理 PUT /api/users/role
// @Summary      批量更新用户角色
// @Description  批量为多个用户分配同一角色
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchUpdateRoleParams  true  "批量角色更新参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users/role [put]
func (h *AdminHandler) BatchUpdateRole(c *gin.Context) {
	var params request.BatchUpdateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	for _, id := range params.UserIDs {
		if !h.userAdminService.CanManageUser(c.Request.Context(), middleware.GetUserRole(c), id) {
			response.Error(c, apperrors.ErrOperationDenied)
			return
		}
	}
	if !h.userAdminService.CanAssignRole(c.Request.Context(), middleware.GetUserRole(c), params.Role) {
		response.Error(c, apperrors.ErrOperationDenied)
		return
	}
	if _, err := h.userAdminService.BatchUpdateRole(c.Request.Context(), params.UserIDs, params.Role); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "role updated"})
}

// BatchDeleteUsers 处理 DELETE /api/users
// @Summary      批量删除用户
// @Description  批量删除指定用户（软删除，不可恢复超级管理员）
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchDeleteUsersParams  true  "批量删除参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users [delete]
func (h *AdminHandler) BatchDeleteUsers(c *gin.Context) {
	var params request.BatchDeleteUsersParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if _, err := h.userAdminService.BatchDelete(c.Request.Context(), params.UserIDs); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "users deleted"})
}

// BatchResetPassword 处理 PUT /api/users/reset-password
// @Summary      批量重置密码
// @Description  批量为多个用户重置密码
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchResetPasswordParams  true  "批量重置密码参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users/reset-password [put]
func (h *AdminHandler) BatchResetPassword(c *gin.Context) {
	var params request.BatchResetPasswordParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if _, err := h.userAdminService.BatchResetPassword(c.Request.Context(), params.UserIDs, params.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "passwords reset"})
}

// ImportUsers 处理 POST /api/users/import
// @Summary      导入用户
// @Description  批量导入用户数据（前端 Excel 解析后提交 JSON）
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ImportUsersParams  true  "导入用户数据"
// @Success      200   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users/import [post]
func (h *AdminHandler) ImportUsers(c *gin.Context) {
	var params request.ImportUsersParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid import data: "+err.Error()))
		return
	}
	records := make([]authsvc.UserImportRecord, len(params.Users))
	for i, user := range params.Users {
		records[i] = authsvc.UserImportRecord{
			Username: user.Username,
			Nickname: user.Nickname,
			Phone:    user.Phone,
			Email:    user.Email,
			RoleID:   user.RoleID,
			Role:     user.Role,
			Password: user.Password,
		}
	}
	summary, appErr := h.importService.ImportUsers(c.Request.Context(), records)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, summary)
}

// ExportUsers 处理 POST /api/users/export
// @Summary      导出用户数据
// @Description  导出指定用户的 JSON 数据，供前端生成 Excel 文件
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ExportUsersParams  true  "导出参数（用户 ID 列表）"
// @Success      200   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/users/export [post]
func (h *AdminHandler) ExportUsers(c *gin.Context) {
	var params request.ExportUsersParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	data, appErr := h.userAdminService.ExportUsersData(c.Request.Context(), params.UserIDs)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, data)
}
