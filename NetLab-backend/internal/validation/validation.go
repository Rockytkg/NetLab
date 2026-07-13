package validation

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"netlab-backend/internal/model"
	"netlab-backend/pkg/apperrors"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func Invalid(message string) *apperrors.AppError {
	return apperrors.New(apperrors.ErrCodeInvalidCode, message)
}

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

func ValidatePassword(password string) *apperrors.AppError {
	if len(password) < 8 || len(password) > 128 {
		return apperrors.ErrWeakPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return apperrors.New(apperrors.ErrCodeWeakPassword, "password must contain letters and numbers")
	}
	return nil
}

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

func NormalizeRole(role string, allowAdmin bool) (model.UserRole, *apperrors.AppError) {
	value := strings.TrimSpace(role)
	switch model.UserRole(value) {
	case model.RoleEditor, model.RoleViewer:
		return model.UserRole(value), nil
	case model.RoleAdmin:
		if !allowAdmin {
			return "", apperrors.New(apperrors.ErrCodeOperationDenied, "admin role is protected")
		}
		return model.RoleAdmin, nil
	case model.RoleSuperAdmin:
		return "", apperrors.New(apperrors.ErrCodeOperationDenied, "super admin role is reserved for the built-in admin account")
	default:
		return "", Invalid("invalid role: " + value)
	}
}

func NormalizeStatus(status string) (model.UserStatus, *apperrors.AppError) {
	value := strings.TrimSpace(status)
	switch model.UserStatus(value) {
	case model.StatusActive, model.StatusDisabled, model.StatusLocked:
		return model.UserStatus(value), nil
	default:
		return "", Invalid("invalid user status")
	}
}
