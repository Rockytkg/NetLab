package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// AdminHandler 处理系统设置与用户资源端点，访问由 RBAC 权限控制。
type AdminHandler struct {
	adminService        *authsvc.AdminService
	userAdminService    *authsvc.UserAdminService
	importExportService *authsvc.UserImportExportService
	mailer              *mailer.Provider
	logger              *zap.Logger
}

// NewAdminHandler 创建一个新的 AdminHandler。
func NewAdminHandler(adminService *authsvc.AdminService, userAdminService *authsvc.UserAdminService, importExportService *authsvc.UserImportExportService, mailerProvider *mailer.Provider, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		adminService:        adminService,
		userAdminService:    userAdminService,
		importExportService: importExportService,
		mailer:              mailerProvider,
		logger:              logger,
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

// UpdateUser 处理 PUT /api/users/:id
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

// BatchUpdateRole 处理 PUT /api/users/role
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

// BatchDeleteUsers 处理 DELETE /api/users
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

// ImportUsers 处理 POST /api/users/import，仅接受前端序列化后的 JSON。
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
			Email:    user.Email,
			Role:     user.Role,
			Password: user.Password,
		}
	}
	summary, appErr := h.importExportService.ImportUsers(c.Request.Context(), records)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, summary)
}

// xlsxContentType 是 .xlsx 文件的标准 MIME 类型。
const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ExportUsers 处理 POST /api/users/export。
// 仅导出请求体中勾选的用户；Excel 构建成功后才写响应头，避免出错时
// 向已声明为 xlsx 的响应写入 JSON 错误导致下载损坏文件。
func (h *AdminHandler) ExportUsers(c *gin.Context) {
	var params request.ExportUsersParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	data, appErr := h.importExportService.ExportUsersExcel(c.Request.Context(), params.UserIDs, contextkeys.GetLocale(c))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=users-%s.xlsx", time.Now().Format("20060102-150405")))
	c.Data(http.StatusOK, xlsxContentType, data)
}

// DownloadImportTemplate 处理 GET /api/users/import-template。
// 返回按请求 locale 本地化表头的 xlsx 导入模板。
func (h *AdminHandler) DownloadImportTemplate(c *gin.Context) {
	data, appErr := h.importExportService.BuildImportTemplate(contextkeys.GetLocale(c))
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=netlab-users-template.xlsx")
	c.Data(http.StatusOK, xlsxContentType, data)
}
