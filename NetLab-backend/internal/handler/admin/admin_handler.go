package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"netlab-backend/internal/contextkeys"
	"netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/mailer"
	authsvc "netlab-backend/internal/service/auth"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// AdminHandler 处理系统设置管理端点（需 admin 角色）。
type AdminHandler struct {
	adminService     *authsvc.AdminService
	userAdminService *authsvc.UserAdminService
	mailer           *mailer.Provider
	logger           *zap.Logger
}

// NewAdminHandler 创建一个新的 AdminHandler。
func NewAdminHandler(adminService *authsvc.AdminService, userAdminService *authsvc.UserAdminService, mailerProvider *mailer.Provider, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		adminService:     adminService,
		userAdminService: userAdminService,
		mailer:           mailerProvider,
		logger:           logger,
	}
}

// GetSettings 处理 GET /api/admin/settings
// @Summary      Get system settings
// @Description  Return all system settings (secrets masked). Admin only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse
// @Failure      403  {object}  response.ApiResponse
// @Router       /api/admin/settings [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	settings, err := h.adminService.GetSettings(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, settings)
}

// UpdateSecurity 处理 PUT /api/admin/settings/security
// @Summary      Update security settings
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateSecurityParams  true  "Security settings"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/settings/security [put]
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

// UpdateBeian 处理 PUT /api/admin/settings/beian
// @Summary      Update filing (备案) information
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateBeianParams  true  "Beian settings"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/settings/beian [put]
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

// UpdateSMTP 处理 PUT /api/admin/settings/smtp
// @Summary      Update SMTP settings
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.UpdateSMTPParams  true  "SMTP settings"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/settings/smtp [put]
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

// TestSMTP 处理 POST /api/admin/settings/smtp/test
// @Summary      Send a test email using the current SMTP settings
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.TestSMTPParams  true  "Recipient"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/settings/smtp/test [post]
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

// UpdateOAuthProvider 处理 PUT /api/admin/settings/oauth/:provider
// @Summary      Update an OAuth provider configuration
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        provider  path  string  true  "Provider ID"
// @Param        body  body      request.UpdateOAuthParams  true  "OAuth provider settings"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/settings/oauth/{provider} [put]
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

// ListUsers 处理 GET /api/admin/users
// @Summary      List users
// @Description  Paginated user list with optional keyword filter. Admin only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        page     query     int     false  "Page number (1-based)"
// @Param        size     query     int     false  "Page size"
// @Param        keyword  query     string  false  "Filter by username or email"
// @Param        status   query     string  false  "Filter by status"
// @Param        role     query     string  false  "Filter by role"
// @Success      200      {object}  response.ApiResponse
// @Router       /api/admin/users [get]
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

// CreateUser 处理 POST /api/admin/users
// @Summary      Create one user
// @Description  Create a user from the admin panel. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.CreateUserParams  true  "New user fields"
// @Success      200   {object}  response.ApiResponse
// @Router       /api/admin/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var params request.CreateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	user, err := h.userAdminService.CreateUser(c.Request.Context(), params.Username, params.Email, params.Role, params.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, user)
}

// UpdateUser 处理 PUT /api/admin/users/:id
// @Summary      Update one user
// @Description  Update a user's email, role, and status. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "User ID"
// @Param        body  body      request.UpdateUserParams  true  "Managed user fields"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	var params request.UpdateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if err := h.userAdminService.UpdateUser(c.Request.Context(), c.Param("id"), params.Email, params.Role, params.Status); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "user updated"})
}

// BatchUpdateRole 处理 PUT /api/admin/users/role
// @Summary      Batch update user role
// @Description  Update role for multiple users. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchUpdateRoleParams  true  "User IDs and role"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/users/role [put]
func (h *AdminHandler) BatchUpdateRole(c *gin.Context) {
	var params request.BatchUpdateRoleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if _, err := h.userAdminService.BatchUpdateRole(c.Request.Context(), params.UserIDs, params.Role); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "role updated"})
}

// BatchDeleteUsers 处理 DELETE /api/admin/users
// @Summary      Batch delete users
// @Description  Delete selected users after confirmation. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchDeleteUsersParams  true  "User IDs"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/users [delete]
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

// BatchResetPassword 处理 PUT /api/admin/users/reset-password
// @Summary      Batch reset user passwords
// @Description  Set a common new password for multiple users. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.BatchResetPasswordParams  true  "User IDs and new password"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/admin/users/reset-password [put]
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

// ImportUsers 处理 POST /api/admin/users/import
// @Summary      Import users from CSV
// @Description  Bulk-create users from an uploaded CSV file (columns: username,email,role,password). Admin only.
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file  formData  file  true  "CSV file"
// @Success      200   {object}  response.ApiResponse
// @Router       /api/admin/users/import [post]
func (h *AdminHandler) ImportUsers(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "csv file is required"))
		return
	}
	// 限制上传大小（2 MB），避免超大文件占用内存。
	if fileHeader.Size > 2<<20 {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "csv file too large (max 2MB)"))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "failed to open csv file"))
		return
	}
	defer f.Close()

	summary, appErr := h.userAdminService.ImportUsersCSV(c.Request.Context(), f)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, summary)
}
