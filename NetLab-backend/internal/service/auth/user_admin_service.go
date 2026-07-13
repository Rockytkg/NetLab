package auth

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

// UserAdminService 承载管理端的用户管理业务逻辑：分页查询、批量改角色、
// 批量重置密码与 CSV 批量导入。
type UserAdminService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

// NewUserAdminService 创建一个新的 UserAdminService。
func NewUserAdminService(userRepo *repository.UserRepository, logger *zap.Logger) *UserAdminService {
	return &UserAdminService{userRepo: userRepo, logger: logger}
}

// assignableRoles 是允许通过管理端批量/导入分配的角色集合。admin 仅允许在单用户创建/编辑表单中显式设置。
var assignableRoles = map[string]bool{
	string(model.RoleEditor): true,
	string(model.RoleViewer): true,
}

// AdminUserView 是返回给管理端的用户视图（不含密码哈希等敏感字段）。
type AdminUserView struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Avatar      string  `json:"avatar"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	IsAdmin     bool    `json:"isAdmin"`
	LastLoginAt *string `json:"lastLoginAt"`
	CreatedAt   string  `json:"createdAt"`
}

// UserListResult 是分页用户列表的返回结构。
type UserListResult struct {
	Items []AdminUserView `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// ImportSummary 汇总一次 CSV 导入的结果。
type ImportSummary struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// ListUsers 分页返回用户列表。内置 admin/super_admin 永远隐藏；普通管理员默认展示。
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

// CreateUser 创建单个用户。管理端单用户表单允许显式分配 admin。
func (s *UserAdminService) CreateUser(ctx context.Context, username, email string, role string, password string) (*AdminUserView, *apperrors.AppError) {
	normalizedUsername, appErr := validation.NormalizeUsername(username)
	if appErr != nil {
		return nil, appErr
	}
	if strings.EqualFold(normalizedUsername, "admin") {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "admin username is reserved for the built-in super administrator")
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
	user := &model.User{
		Username:          normalizedUsername,
		Email:             normalizedEmail,
		PasswordHash:      hash,
		Role:              normalizedRole,
		Status:            model.StatusActive,
		PasswordChangedAt: &now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to create user", err)
	}
	view := toAdminUserView(user)
	s.logger.Info("created user from admin panel", zap.String("userID", view.ID), zap.String("username", view.Username))
	return &view, nil
}

// UpdateUser 更新单个用户的邮箱、角色和状态。
func (s *UserAdminService) UpdateUser(ctx context.Context, id, email string, role string, status string) *apperrors.AppError {
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
	} else if existing != nil && existing.ID.String() != id {
		return apperrors.ErrEmailExists
	}
	if err := s.userRepo.UpdateManagedFields(ctx, id, normalizedEmail, normalizedRole, normalizedStatus); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update user", err)
	}
	return nil
}

// BatchUpdateRole 批量修改用户角色。
func (s *UserAdminService) BatchUpdateRole(ctx context.Context, ids []string, role string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidCredentials, "no users selected")
	}
	normalizedRole, appErr := validation.NormalizeRole(role, true)
	if appErr != nil {
		return 0, appErr
	}

	affected, err := s.userRepo.BatchUpdateRole(ctx, ids, normalizedRole)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update role", err)
	}
	s.logger.Info("batch updated user role",
		zap.Int("count", len(ids)),
		zap.String("role", string(normalizedRole)),
	)
	return affected, nil
}

// BatchDelete 删除选中的用户。调用方的管理权限由路由 RBAC 统一控制，
// 这里不再按目标用户角色做额外保护。
func (s *UserAdminService) BatchDelete(ctx context.Context, ids []string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidCredentials, "no users selected")
	}
	users, err := s.userRepo.FindByIDs(ctx, ids)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", err)
	}
	if len(users) != len(ids) {
		return 0, apperrors.ErrUserNotFound
	}
	affected, err := s.userRepo.BatchDelete(ctx, ids)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to delete users", err)
	}
	return affected, nil
}

// BatchResetPassword 为选中用户设置统一的新密码，并清除锁定状态。
func (s *UserAdminService) BatchResetPassword(ctx context.Context, ids []string, newPassword string) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidCredentials, "no users selected")
	}
	if len(newPassword) < 8 {
		return 0, apperrors.ErrWeakPassword
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return 0, appErr
	}
	users, findErr := s.userRepo.FindByIDs(ctx, ids)
	if findErr != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", findErr)
	}
	if len(users) != len(ids) {
		return 0, apperrors.ErrUserNotFound
	}

	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	affected, err := s.userRepo.BatchUpdatePassword(ctx, ids, hash)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to reset passwords", err)
	}
	s.logger.Info("batch reset user passwords", zap.Int("count", len(ids)))
	return affected, nil
}

// ImportUsersCSV 从 CSV 批量导入用户。
//
// 期望的表头（大小写不敏感，列顺序不限）：username,email,role,password。
// role 可留空（默认 viewer），且不允许为 admin；password 可留空（默认与
// 用户名相同，仅用于快速初始化，建议导入后重置）。逐行校验并跳过重复项，
// 返回创建/跳过数量与错误明细。
func (s *UserAdminService) ImportUsersCSV(ctx context.Context, r io.Reader) (*ImportSummary, *apperrors.AppError) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // 允许行长不一致，由业务逐行校验
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeInvalidCredentials, "empty or invalid csv")
	}
	idx := indexHeader(header)
	if idx["username"] < 0 || idx["email"] < 0 {
		return nil, apperrors.New(apperrors.ErrCodeInvalidCredentials, "csv must contain 'username' and 'email' columns")
	}

	summary := &ImportSummary{Errors: []string{}}
	line := 1
	for {
		line++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: %v", line, err))
			continue
		}

		username, nameErr := validation.NormalizeUsername(field(record, idx["username"]))
		email, emailErr := validation.NormalizeEmail(field(record, idx["email"]))
		role := field(record, idx["role"])
		password := field(record, idx["password"])

		if nameErr != nil || emailErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: username and email are required", line))
			summary.Skipped++
			continue
		}
		if role == "" {
			role = string(model.RoleViewer)
		}
		if !assignableRoles[role] {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: role not assignable: %s", line, role))
			summary.Skipped++
			continue
		}
		if password == "" {
			password = username
		}
		if appErr := validation.ValidatePassword(password); appErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: password must be 8-128 characters and contain letters and numbers", line))
			summary.Skipped++
			continue
		}

		if exists, _ := s.userRepo.ExistsByUsername(ctx, username); exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: username already exists: %s", line, username))
			summary.Skipped++
			continue
		}
		if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: email already exists: %s", line, email))
			summary.Skipped++
			continue
		}

		hash, hErr := crypto.HashPassword(password)
		if hErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: failed to hash password", line))
			summary.Skipped++
			continue
		}

		now := time.Now()
		user := &model.User{
			Username:            username,
			Email:               email,
			PasswordHash:        hash,
			Role:                model.UserRole(role),
			Status:              model.StatusActive,
			ForcePasswordChange: true,
			ForceEmailChange:    true,
			PasswordChangedAt:   &now,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: failed to create user: %v", line, err))
			summary.Skipped++
			continue
		}
		summary.Created++
	}

	s.logger.Info("csv user import finished",
		zap.Int("created", summary.Created),
		zap.Int("skipped", summary.Skipped),
	)
	return summary, nil
}

// ─── 辅助 ────────────────────────────────────────────────────────────

func isAdmin(u *model.User) bool {
	return u.Role == model.RoleAdmin || u.Role == model.RoleSuperAdmin
}

func toAdminUserView(u *model.User) AdminUserView {
	v := AdminUserView{
		ID:        u.ID.String(),
		Username:  u.Username,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Role:      string(u.Role),
		Status:    string(u.Status),
		IsAdmin:   isAdmin(u),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if u.LastLoginAt != nil {
		s := u.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		v.LastLoginAt = &s
	}
	return v
}

// indexHeader 建立表头列名到索引的映射（小写、去空格）。
func indexHeader(header []string) map[string]int {
	idx := map[string]int{"username": -1, "email": -1, "role": -1, "password": -1}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if pos := strings.IndexAny(key, " ("); pos > 0 {
			key = key[:pos]
		}
		if _, ok := idx[key]; ok {
			idx[key] = i
		}
	}
	return idx
}

// field 安全地按索引取值并去除首尾空白；索引越界或为 -1 时返回空串。
func field(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}
