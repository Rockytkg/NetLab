package auth

import (
	"context"
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

// UserAdminService 承载用户资源的管理业务，访问控制由路由层 RBAC 权限负责。
type UserAdminService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

func NewUserAdminService(userRepo *repository.UserRepository, logger *zap.Logger) *UserAdminService {
	return &UserAdminService{userRepo: userRepo, logger: logger}
}

// AdminUserView 是返回给用户资源接口的安全视图。
type AdminUserView struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Avatar      string  `json:"avatar"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	LastLoginAt *string `json:"lastLoginAt"`
	CreatedAt   string  `json:"createdAt"`
}

type UserListResult struct {
	Items []AdminUserView `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

type ImportSummary struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

func (s *UserAdminService) ListUsers(ctx context.Context, page, size int, keyword, status, role string) (*UserListResult, *apperrors.AppError) {
	status = strings.TrimSpace(status)
	if status != "" {
		if _, appErr := validation.NormalizeStatus(status); appErr != nil {
			return nil, appErr
		}
	}
	role = strings.TrimSpace(role)
	if role != "" {
		if _, appErr := validation.NormalizeRole(role, true); appErr != nil {
			return nil, appErr
		}
	}
	users, total, err := s.userRepo.List(ctx, page, size, strings.TrimSpace(keyword), status, role)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list users", err)
	}
	items := make([]AdminUserView, len(users))
	for i := range users {
		items[i] = toAdminUserView(&users[i])
	}
	if page < 1 {
		page = 1
	}
	return &UserListResult{Items: items, Total: total, Page: page, Size: size}, nil
}

func (s *UserAdminService) CreateUser(ctx context.Context, username, email, role, password string) (*AdminUserView, *apperrors.AppError) {
	normalizedUsername, appErr := validation.NormalizeUsername(username)
	if appErr != nil {
		return nil, appErr
	}
	if strings.EqualFold(normalizedUsername, "superadmin") {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "superadmin username is reserved for the built-in super administrator")
	}
	normalizedEmail, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return nil, appErr
	}
	normalizedRole, appErr := validation.NormalizeRole(role, true)
	if appErr != nil {
		return nil, appErr
	}
	if appErr := validation.ValidatePassword(password); appErr != nil {
		return nil, appErr
	}
	if exists, err := s.userRepo.ExistsByUsername(ctx, normalizedUsername); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if exists {
		return nil, apperrors.ErrUsernameExists
	}
	if exists, err := s.userRepo.ExistsByEmail(ctx, normalizedEmail); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if exists {
		return nil, apperrors.ErrEmailExists
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	now := time.Now()
	user := &model.User{Username: normalizedUsername, Email: normalizedEmail, PasswordHash: hash, Role: normalizedRole, Status: model.StatusActive, PasswordChangedAt: &now}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to create user", err)
	}
	view := toAdminUserView(user)
	s.logger.Info("created user", zap.String("userID", view.ID), zap.String("username", view.Username))
	return &view, nil
}

func (s *UserAdminService) UpdateUser(ctx context.Context, id, email, role, status string) *apperrors.AppError {
	users, err := s.userRepo.FindByIDs(ctx, []string{id})
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", err)
	}
	if len(users) == 0 {
		return apperrors.ErrUserNotFound
	}
	normalizedEmail, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	normalizedRole, appErr := validation.NormalizeRole(role, true)
	if appErr != nil {
		return appErr
	}
	normalizedStatus, appErr := validation.NormalizeStatus(status)
	if appErr != nil {
		return appErr
	}
	if existing, err := s.userRepo.FindByEmail(ctx, normalizedEmail); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if existing != nil && strconv.FormatUint(existing.ID, 10) != id {
		return apperrors.ErrEmailExists
	}
	if err := s.userRepo.UpdateManagedFields(ctx, id, normalizedEmail, normalizedRole, normalizedStatus); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update user", err)
	}
	return nil
}

func (s *UserAdminService) BatchUpdateRole(ctx context.Context, ids []string, role string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidRequest, "no users selected")
	}
	normalizedRole, appErr := validation.NormalizeRole(role, true)
	if appErr != nil {
		return 0, appErr
	}
	affected, err := s.userRepo.BatchUpdateRole(ctx, ids, normalizedRole)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update role", err)
	}
	return affected, nil
}

func (s *UserAdminService) BatchDelete(ctx context.Context, ids []string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidRequest, "no users selected")
	}
	affected, err := s.userRepo.BatchDelete(ctx, ids)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to delete users", err)
	}
	return affected, nil
}

func (s *UserAdminService) BatchResetPassword(ctx context.Context, ids []string, newPassword string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidRequest, "no users selected")
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return 0, appErr
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	affected, err := s.userRepo.BatchUpdatePassword(ctx, ids, hash)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to reset passwords", err)
	}
	return affected, nil
}

func toAdminUserView(u *model.User) AdminUserView {
	view := AdminUserView{ID: strconv.FormatUint(u.ID, 10), Username: u.Username, Email: u.Email, Avatar: u.Avatar, Role: string(u.Role), Status: string(u.Status), CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
	if u.LastLoginAt != nil {
		value := u.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		view.LastLoginAt = &value
	}
	return view
}
