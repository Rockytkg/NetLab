package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

// UserImportService 承载用户 JSON 批量导入业务。
type UserImportService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

// NewUserImportService 创建用户导入服务。
func NewUserImportService(userRepo *repository.UserRepository, logger *zap.Logger) *UserImportService {
	return &UserImportService{userRepo: userRepo, logger: logger}
}

// UserImportRecord 是前端解析表格后提交的一条用户记录。
type UserImportRecord struct {
	Username       string
	Nickname       string
	Phone          string
	Email          string
	RoleID         string
	RoleIdentifier string
	Password       string
}

// ImportUsers 从 JSON 记录批量导入用户。表格解析不在后端执行。
func (s *UserImportService) ImportUsers(ctx context.Context, records []UserImportRecord) (*ImportSummary, *apperrors.AppError) {
	summary := &ImportSummary{Errors: []string{}}
	for rowNumber, record := range records {
		line := rowNumber + 1
		username, nameErr := validation.NormalizeUsername(record.Username)
		nickname, nicknameErr := validation.NormalizeNickname(record.Nickname)
		phone, phoneErr := validation.NormalizePhone(record.Phone)
		email, emailErr := validation.NormalizeEmail(record.Email)
		if nameErr != nil || nicknameErr != nil || phoneErr != nil || emailErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: username, nickname, phone and email are required and valid", line))
			summary.Skipped++
			continue
		}
		if strings.EqualFold(username, "superadmin") {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: superadmin username is reserved", line))
			summary.Skipped++
			continue
		}

		role, roleErr := s.resolveRole(ctx, record.RoleID, record.RoleIdentifier)
		if roleErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: %s", line, roleErr.Message))
			summary.Skipped++
			continue
		}
		normalizedRole, roleErr := validation.NormalizeRole(role, false)
		if roleErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: %s", line, roleErr.Message))
			summary.Skipped++
			continue
		}

		password := strings.TrimSpace(record.Password)
		if password == "" {
			password = username
		}
		if passwordErr := validation.ValidatePassword(password); passwordErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: %s", line, passwordErr.Message))
			summary.Skipped++
			continue
		}

		exists, checkErr := s.userRepo.ExistsByUsername(ctx, username)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check username", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: username already exists: %s", line, username))
			summary.Skipped++
			continue
		}
		exists, checkErr = s.userRepo.ExistsByPhone(ctx, phone)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check phone", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: phone already exists: %s", line, phone))
			summary.Skipped++
			continue
		}
		exists, checkErr = s.userRepo.ExistsByEmail(ctx, email)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check email", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: email already exists: %s", line, email))
			summary.Skipped++
			continue
		}

		hash, hashErr := crypto.HashPassword(password)
		if hashErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: failed to hash password", line))
			summary.Skipped++
			continue
		}
		now := time.Now()
		user := &model.User{
			Username: username, Nickname: nickname, Phone: phone, Email: email, PasswordHash: hash,
			Role: normalizedRole, Status: model.StatusActive,
			ForcePasswordChange: true, ForceEmailChange: true, PasswordChangedAt: &now,
		}
		if createErr := s.userRepo.Create(ctx, user); createErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: failed to create user", line))
			summary.Skipped++
			continue
		}
		summary.Created++
	}

	s.logger.Info("json user import finished", zap.Int("created", summary.Created), zap.Int("skipped", summary.Skipped))
	return summary, nil
}

func (s *UserImportService) resolveRole(ctx context.Context, roleID, roleIdentifier string) (string, *apperrors.AppError) {
	roleID = strings.TrimSpace(roleID)
	roleIdentifier = strings.TrimSpace(roleIdentifier)
	if roleID == "" && roleIdentifier == "" {
		return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "roleId or roleIdentifier is required")
	}
	if roleID != "" {
		id, err := strconv.ParseUint(roleID, 10, 64)
		if err != nil || id == 0 {
			return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid roleId")
		}
		role, err := s.userRepo.FindRoleByID(ctx, id)
		if err != nil {
			return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "roleId not found")
		}
		return role.Role, nil
	}
	return roleIdentifier, nil
}
