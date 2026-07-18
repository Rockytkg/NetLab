package validation

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"netlab-backend/internal/model"
	"netlab-backend/pkg/apperrors"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

var (
	passwordHasLowercase = regexp.MustCompile(`[a-z]`)
	passwordHasUppercase = regexp.MustCompile(`[A-Z]`)
	passwordHasDigit     = regexp.MustCompile(`[0-9]`)
	passwordHasSpecial   = regexp.MustCompile(`[^A-Za-z0-9]`)
)

const (
	// PasswordMinLength 密码最小长度（按字符/rune 计）。
	PasswordMinLength = 8
	// PasswordMaxBytes 密码最大字节数（对齐 bcrypt 72 字节硬上限）。
	PasswordMaxBytes = 72
)

// Invalid 构造一个「参数无效」的应用错误（错误码为 ErrCodeInvalidCode）。
func Invalid(message string) *apperrors.AppError {
	return apperrors.New(apperrors.ErrCodeInvalidCode, message)
}

// NormalizeEmail 规范化并校验邮箱地址（去空白、转小写、格式校验）。
func NormalizeEmail(email string) (string, *apperrors.AppError) {
	value := strings.ToLower(strings.TrimSpace(email))
	if value == "" {
		return "", Invalid("email is required")
	}
	if len(value) > 255 {
		return "", Invalid("email must be at most 255 characters")
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || !strings.Contains(value, ".") {
		return "", Invalid("email format is invalid")
	}
	return value, nil
}

// NormalizeUsername 规范化并校验用户名（3-64 位，仅限字母、数字、下划线和连字符）。
func NormalizeUsername(username string) (string, *apperrors.AppError) {
	value := strings.TrimSpace(username)
	if len(value) < 3 || len(value) > 64 {
		return "", Invalid("username must be 3 to 64 characters")
	}
	if !usernamePattern.MatchString(value) {
		return "", Invalid("username may contain only letters, numbers, underscores, and hyphens")
	}
	return value, nil
}

// NormalizeNickname 规范化并校验昵称（非空且不超过 64 个字符）。
func NormalizeNickname(nickname string) (string, *apperrors.AppError) {
	value := strings.TrimSpace(nickname)
	if value == "" || utf8.RuneCountInString(value) > 64 {
		return "", Invalid("nickname is required and must be at most 64 characters")
	}
	return value, nil
}

// NormalizePhone 规范化并校验手机号（中国大陆 11 位手机号格式）。
func NormalizePhone(phone string) (string, *apperrors.AppError) {
	value := strings.TrimSpace(phone)
	if !phonePattern.MatchString(value) {
		return "", Invalid("phone format is invalid")
	}
	return value, nil
}

// ValidatePassword 校验密码是否满足基本复杂度要求。
func ValidatePassword(password string) *apperrors.AppError {
	// 1. 检查 UTF-8 合法性
	if !utf8.ValidString(password) {
		return apperrors.ErrWeakPassword
	}

	// 2. 最小字符数（使用 rune 而非 byte）
	if utf8.RuneCountInString(password) < PasswordMinLength {
		return apperrors.ErrWeakPassword
	}

	// 3. 最大字节数（bcrypt 硬上限）
	if len(password) > PasswordMaxBytes {
		return apperrors.ErrWeakPassword
	}

	if !passwordHasLowercase.MatchString(password) ||
		!passwordHasUppercase.MatchString(password) ||
		!passwordHasDigit.MatchString(password) ||
		!passwordHasSpecial.MatchString(password) {
		return apperrors.ErrWeakPassword
	}

	return nil
}

// NormalizeVerifyCode 规范化并校验邮箱验证码格式（6 位数字）。
func NormalizeVerifyCode(code string) (string, *apperrors.AppError) {
	value := strings.TrimSpace(code)
	if len(value) != 6 {
		return "", apperrors.ErrInvalidCode
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", apperrors.ErrInvalidCode
		}
	}
	return value, nil
}

// NormalizeRole 规范化并校验角色标识：superadmin 始终保留不可用，admin 仅在
// allowAdmin 为 true 时放行，其余按自定义角色格式（2-64 位字母、数字、下划线或连字符）校验。
func NormalizeRole(role string, allowAdmin bool) (model.UserRole, *apperrors.AppError) {
	value := strings.TrimSpace(role)
	if value == string(model.RoleSuperAdmin) || value == "superadmin" {
		return "", apperrors.New(apperrors.ErrCodeOperationDenied, "superadmin role is reserved")
	}
	if value == string(model.RoleAdmin) && !allowAdmin {
		return "", apperrors.New(apperrors.ErrCodeOperationDenied, "admin role is protected")
	}
	if len(value) < 2 || len(value) > 64 {
		return "", Invalid("invalid role: " + value)
	}
	for _, r := range value {
		if !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return "", Invalid("invalid role: " + value)
		}
	}
	return model.UserRole(value), nil
}

// NormalizeStatus 规范化并校验用户状态（active/disabled/locked）。
func NormalizeStatus(status string) (model.UserStatus, *apperrors.AppError) {
	value := strings.TrimSpace(status)
	switch model.UserStatus(value) {
	case model.StatusActive, model.StatusDisabled, model.StatusLocked:
		return model.UserStatus(value), nil
	default:
		return "", Invalid("invalid user status")
	}
}
