package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"netlab-backend/internal/contextkeys"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/i18n"
)

// ApiResponse 是标准的响应封装。
// code === 0 或 code === 200 表示成功；否则为业务错误。
type ApiResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}

// Success 以本地化的成功消息进行响应。
func Success(c *gin.Context, status int, data interface{}) {
	locale := contextkeys.GetLocale(c)
	c.JSON(status, ApiResponse{
		Code:    0,
		Data:    data,
		Message: i18n.MustT(locale, "success"),
	})
}

// SuccessOK 是 HTTP 200 的 Success 简写。
func SuccessOK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data)
}

// SuccessCreated 是 HTTP 201 的 Success 简写。
func SuccessCreated(c *gin.Context, data interface{}) {
	Success(c, http.StatusCreated, data)
}

// SuccessNoContent 以 HTTP 204 且无响应体进行响应。
func SuccessNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error 以本地化的业务错误进行响应。
// 消息根据错误码并使用请求的 locale 解析得到。
func Error(c *gin.Context, appErr *apperrors.AppError) {
	locale := contextkeys.GetLocale(c)
	status := appErr.Code.HTTPStatus()
	c.AbortWithStatusJSON(status, ApiResponse{
		Code:    int(appErr.Code),
		Message: localizeError(locale, appErr),
	})
}

// ErrorWithData 以本地化的业务错误并附带额外数据进行响应。
func ErrorWithData(c *gin.Context, appErr *apperrors.AppError, data interface{}) {
	locale := contextkeys.GetLocale(c)
	status := appErr.Code.HTTPStatus()
	c.AbortWithStatusJSON(status, ApiResponse{
		Code:    int(appErr.Code),
		Data:    data,
		Message: localizeError(locale, appErr),
	})
}

// AbortWithError 是一个便捷函数，供 handler 在出错后中断
// 中间件链时使用。
func AbortWithError(c *gin.Context, appErr *apperrors.AppError) {
	Error(c, appErr)
}

// ValidationError 以 422 响应校验错误。
func ValidationError(c *gin.Context, errors []apperrors.ValidationError) {
	locale := contextkeys.GetLocale(c)
	count := len(errors)
	msg := i18n.T(locale, "validation_failed_with_count", map[string]any{"Count": count})
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, ApiResponse{
		Code:    http.StatusUnprocessableEntity,
		Data:    errors,
		Message: msg,
	})
}

// InternalError 使用本地化消息以通用的 500 进行响应。
// 若本地化不可用，则回退到提供的 message。
func InternalError(c *gin.Context, message string) {
	locale := contextkeys.GetLocale(c)
	localized := i18n.MustT(locale, "error.internal")
	if localized == "error.internal" {
		// i18n bundle 未初始化或键缺失 —— 使用原始 message
		localized = message
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, ApiResponse{
		Code:    http.StatusInternalServerError,
		Message: localized,
	})
}

// localizeError 为一个 AppError 解析本地化消息。
// 它使用 i18n 键 "error.{code}"，若本地化失败则回退到
// AppError 内置的英文消息。
func localizeError(locale string, appErr *apperrors.AppError) string {
	key := fmt.Sprintf("error.%d", appErr.Code)
	msg := i18n.MustT(locale, key)
	if msg == key {
		// 回退到内嵌的英文消息
		return appErr.Message
	}
	return msg
}
