package apperrors

import (
	"fmt"
	"net/http"
)

// ErrorCode 表示业务错误码。
type ErrorCode int

// 业务错误码 —— 必须与前端 BUSINESS_ERROR_I18N_MAP 保持一致。
// 每个错误码关联的 message 作为英文兜底文案；
// 服务端 i18n 通过 go-i18n 使用键 "error.{code}" 解析本地化文案。
const (
	ErrCodeInvalidCredentials     ErrorCode = 1001
	ErrCodeAccountLocked          ErrorCode = 1002
	ErrCodeAccountDisabled        ErrorCode = 1003
	ErrCodeTokenExpired           ErrorCode = 1004
	ErrCodeInvalidRefreshToken    ErrorCode = 1005
	ErrCodeUserNotFound           ErrorCode = 1006
	ErrCodeEmailExists            ErrorCode = 1007
	ErrCodeUsernameExists         ErrorCode = 1008
	ErrCodeInvalidCode            ErrorCode = 1009
	ErrCodeWeakPassword           ErrorCode = 1010
	ErrCodeRateLimited            ErrorCode = 1011
	ErrCodeSessionExpired         ErrorCode = 1012
	ErrCodeDuplicateEntry         ErrorCode = 1013
	ErrCodeOperationDenied        ErrorCode = 1014
	ErrCodeResourceInUse          ErrorCode = 1015
	ErrCodeEmailNotConfigured     ErrorCode = 1016
	ErrCodeEmailSendFailed        ErrorCode = 1017
	ErrCodePasswordResetClosed    ErrorCode = 1018
	ErrCodeInvalidTwoFactorCode   ErrorCode = 1020
	ErrCodeTwoFactorNotConfigured ErrorCode = 1021
)

// HTTPStatus 将错误码映射为对应的 HTTP 状态码。
func (c ErrorCode) HTTPStatus() int {
	switch c {
	case ErrCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrCodeAccountLocked, ErrCodeAccountDisabled:
		return http.StatusForbidden
	case ErrCodeTokenExpired, ErrCodeInvalidRefreshToken, ErrCodeSessionExpired:
		return http.StatusUnauthorized
	case ErrCodeUserNotFound:
		return http.StatusNotFound
	case ErrCodeEmailExists, ErrCodeUsernameExists, ErrCodeDuplicateEntry:
		return http.StatusConflict
	case ErrCodeInvalidCode, ErrCodeWeakPassword:
		return http.StatusBadRequest
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeOperationDenied, ErrCodePasswordResetClosed:
		return http.StatusForbidden
	case ErrCodeInvalidTwoFactorCode, ErrCodeTwoFactorNotConfigured:
		return http.StatusBadRequest
	case ErrCodeResourceInUse:
		return http.StatusConflict
	case ErrCodeEmailNotConfigured, ErrCodeEmailSendFailed:
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadRequest
	}
}

// AppError 是包含错误码和消息的应用级错误。
// Message 字段保存英文兜底文案；调用方应优先使用
// response.Error()，它会通过 go-i18n 解析本地化消息。
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error // 包装的内部错误（不暴露给客户端）
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New 使用给定的错误码和英文兜底消息创建一个新的 AppError。
func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap 创建一个包装内部错误的 AppError。
func Wrap(code ErrorCode, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// 常见场景的预定义错误。
// 它们携带英文兜底消息；response.go 在写入响应时对其进行本地化。
var (
	ErrInvalidCredentials     = New(ErrCodeInvalidCredentials, "invalid credentials")
	ErrAccountLocked          = New(ErrCodeAccountLocked, "account locked")
	ErrAccountDisabled        = New(ErrCodeAccountDisabled, "account disabled")
	ErrTokenExpired           = New(ErrCodeTokenExpired, "token expired")
	ErrInvalidRefreshToken    = New(ErrCodeInvalidRefreshToken, "invalid refresh token")
	ErrUserNotFound           = New(ErrCodeUserNotFound, "user not found")
	ErrEmailExists            = New(ErrCodeEmailExists, "email already exists")
	ErrUsernameExists         = New(ErrCodeUsernameExists, "username already exists")
	ErrInvalidCode            = New(ErrCodeInvalidCode, "invalid verification code")
	ErrWeakPassword           = New(ErrCodeWeakPassword, "password does not meet strength requirements")
	ErrRateLimited            = New(ErrCodeRateLimited, "too many requests, please try again later")
	ErrSessionExpired         = New(ErrCodeSessionExpired, "session expired")
	ErrDuplicateEntry         = New(ErrCodeDuplicateEntry, "duplicate entry")
	ErrOperationDenied        = New(ErrCodeOperationDenied, "operation denied")
	ErrResourceInUse          = New(ErrCodeResourceInUse, "resource in use")
	ErrEmailNotConfigured     = New(ErrCodeEmailNotConfigured, "email service is not configured")
	ErrEmailSendFailed        = New(ErrCodeEmailSendFailed, "failed to send verification email")
	ErrPasswordResetClosed    = New(ErrCodePasswordResetClosed, "password reset is disabled")
	ErrInvalidTwoFactorCode   = New(ErrCodeInvalidTwoFactorCode, "invalid two-factor authentication code")
	ErrTwoFactorNotConfigured = New(ErrCodeTwoFactorNotConfigured, "two-factor authentication is not configured")
)

// ValidationError 用于请求校验失败。
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors 聚合多个字段错误。
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	return fmt.Sprintf("validation errors: %d fields", len(ve))
}
