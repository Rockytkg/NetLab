// Package log 提供登录日志查询端点的 HTTP 处理。
package log

import (
	"strconv"

	"github.com/gin-gonic/gin"

	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/middleware"
	"netlab-backend/internal/model"
	logsvc "netlab-backend/internal/service/log"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// Handler 处理登录日志相关端点的 HTTP 请求。
type Handler struct {
	logService *logsvc.Service
}

// NewHandler 创建一个新的登录日志 Handler。
func NewHandler(logService *logsvc.Service) *Handler {
	return &Handler{logService: logService}
}

// ListLoginLogs 处理 GET /api/logs/logins
// @Summary      获取登录日志列表
// @Description  分页查询登录日志，支持按关键词（用户名/IP）、状态、登录方式筛选；仅返回不高于操作者管理级别的用户日志
// @Tags         Logs
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码"          default(1)
// @Param        size       query     int     false  "每页数量"       default(20)
// @Param        keyword    query     string  false  "搜索关键词（用户名/IP）"
// @Param        status     query     string  false  "登录状态（success/failed/pending）"
// @Param        loginType  query     string  false  "登录方式（password/2fa/recovery/passkey/oauth）"
// @Success      200        {object}  response.ApiResponse{data=dtoresponse.LoginLogListResult}
// @Failure      401        {object}  response.ApiResponse
// @Failure      403        {object}  response.ApiResponse
// @Router       /api/logs/logins [get]
func (h *Handler) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	keyword := c.Query("keyword")
	status := c.Query("status")
	loginType := c.Query("loginType")

	roleID := middleware.GetUserRole(c)
	if roleID == "" {
		response.Error(c, apperrors.ErrUnauthorized)
		return
	}

	logs, total, appErr := h.logService.List(c.Request.Context(), roleID, page, size, keyword, status, loginType)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	items := make([]dtoresponse.LoginLogItem, len(logs))
	for i, l := range logs {
		items[i] = loginLogToDTO(l)
	}
	response.SuccessOK(c, dtoresponse.LoginLogListResult{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// loginLogToDTO 将 LoginLog 模型转换为 API 响应的 DTO。
func loginLogToDTO(l model.LoginLog) dtoresponse.LoginLogItem {
	return dtoresponse.LoginLogItem{
		ID:          l.ID,
		Username:    l.Username,
		LoginType:   l.LoginType,
		Status:      l.Status,
		IP:          l.IP,
		UserAgent:   l.UserAgent,
		Fingerprint: l.Fingerprint,
		OS:          l.OS,
		Browser:     l.Browser,
		Location:    l.Location,
		CreatedAt:   l.CreatedAt,
	}
}

// DeleteLoginLogs 处理 DELETE /api/logs/logins
// @Summary      批量删除登录日志
// @Description  按 ID 批量删除登录日志；仅删除不高于操作者管理级别的用户日志，返回实际删除条数
// @Tags         Logs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "待删除的日志 ID 列表（最多 500 条）"
// @Success      200   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Failure      401   {object}  response.ApiResponse
// @Failure      403   {object}  response.ApiResponse
// @Router       /api/logs/logins [delete]
func (h *Handler) DeleteLoginLogs(c *gin.Context) {
	var body struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid request parameters: "+err.Error()))
		return
	}

	roleID := middleware.GetUserRole(c)
	if roleID == "" {
		response.Error(c, apperrors.ErrUnauthorized)
		return
	}

	deleted, appErr := h.logService.Delete(c.Request.Context(), roleID, body.IDs)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, gin.H{"deleted": deleted})
}
