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
	userRepo     *repository.UserRepository
	logger       *zap.Logger
	roleResolver interface {
		RoleNameForIdentifier(string) string
	}
	roleManager interface {
		CanManageUser(context.Context, string, string) bool
		CanAssignRole(context.Context, string, string) bool
	}
}

// NewUserAdminService 创建用户管理服务。可选传入角色解析器（提供
// 角色显示名解析），若其同时实现角色管理接口，则一并用于管理级别判定。
func NewUserAdminService(userRepo *repository.UserRepository, logger *zap.Logger, resolvers ...interface{ RoleNameForIdentifier(string) string }) *UserAdminService {
	service := &UserAdminService{userRepo: userRepo, logger: logger}
	if len(resolvers) > 0 {
		service.roleResolver = resolvers[0]
		if manager, ok := resolvers[0].(interface {
			CanManageUser(context.Context, string, string) bool
			CanAssignRole(context.Context, string, string) bool
		}); ok {
			service.roleManager = manager
		}
	}
	return service
}

// CanManageUser 判定操作者能否管理目标用户（依据角色管理级别）。
func (s *UserAdminService) CanManageUser(ctx context.Context, actorRoleID, userID string) bool {
	return s.roleManager != nil && s.roleManager.CanManageUser(ctx, actorRoleID, userID)
}

// CanAssignRole 判定操作者能否将指定角色分配给他人。
func (s *UserAdminService) CanAssignRole(ctx context.Context, actorRoleID, roleCode string) bool {
	return s.roleManager != nil && s.roleManager.CanAssignRole(ctx, actorRoleID, roleCode)
}

// CanUpdateUser 区分「自助资料修改」与「管理性修改」：用户可修改自己的
// 资料字段，但不得通过管理端点更改自己的角色、状态或二步验证状态。
func (s *UserAdminService) CanUpdateUser(ctx context.Context, actorUserID, actorRoleID, userID, requestedRole, requestedStatus string, disableTwoFactor bool) bool {
	users, err := s.userRepo.FindByIDs(ctx, []string{userID})
	if err != nil || len(users) != 1 {
		return false
	}
	if strings.TrimSpace(actorUserID) == strings.TrimSpace(userID) {
		currentRole := string(users[0].Role)
		if currentRole == "" && users[0].RoleID != 0 {
			if role, roleErr := s.userRepo.FindRoleByID(ctx, users[0].RoleID); roleErr == nil && role != nil {
				currentRole = role.Role
			}
		}
		return strings.EqualFold(strings.TrimSpace(currentRole), strings.TrimSpace(requestedRole)) &&
			strings.EqualFold(strings.TrimSpace(string(users[0].Status)), strings.TrimSpace(requestedStatus)) &&
			!disableTwoFactor
	}
	return s.CanManageUser(ctx, actorRoleID, userID) && s.CanAssignRole(ctx, actorRoleID, requestedRole)
}

// AdminUserView 是返回给用户资源接口的安全视图。
type AdminUserView struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	Nickname         string `json:"nickname"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	Role             string `json:"role"`
	RoleName         string `json:"roleName"`
	Status           string `json:"status"`
	TwoFactorEnabled bool   `json:"twoFactorEnabled"`
	CreatedAt        string `json:"createdAt"`
}

// AdminUserExportView 是导出接口专用视图，由前端生成表格文件。
type AdminUserExportView struct {
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	RoleID    string `json:"roleId"`
	Role      string `json:"role"`
	RoleName  string `json:"roleName"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// UserListResult 是用户列表接口的分页返回结构。
type UserListResult struct {
	Items []AdminUserView `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// ImportSummary 汇总一次用户导入的结果：创建数、跳过数及逐条错误信息。
type ImportSummary struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// ListUsers 分页查询用户列表，支持按关键词、状态、角色过滤。
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
		items[i] = s.toAdminUserView(&users[i])
	}
	if page < 1 {
		page = 1
	}
	return &UserListResult{Items: items, Total: total, Page: page, Size: size}, nil
}

// ExportUsersData 返回指定用户的 JSON 数据，由前端负责生成导出文件。
func (s *UserAdminService) ExportUsersData(ctx context.Context, ids []string) ([]AdminUserExportView, *apperrors.AppError) {
	if len(ids) == 0 {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "no users selected")
	}
	users, err := s.userRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load users for export", err)
	}
	byID := make(map[string]model.User, len(users))
	for _, user := range users {
		byID[strconv.FormatUint(user.ID, 10)] = user
	}
	items := make([]AdminUserExportView, 0, len(ids))
	for _, id := range ids {
		if user, ok := byID[strings.TrimSpace(id)]; ok {
			items = append(items, AdminUserExportView{Username: user.Username, Nickname: user.Nickname, Phone: user.Phone, Email: user.Email, RoleID: strconv.FormatUint(user.RoleID, 10), Role: string(user.Role), RoleName: user.RoleName, Status: string(user.Status), CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
		}
	}
	return items, nil
}

// CreateUser 创建新用户：规范化并校验全部字段、检查唯一性约束
// （用户名/邮箱/手机号）、哈希密码后落库。superadmin 用户名保留不可用。
func (s *UserAdminService) CreateUser(ctx context.Context, username, nickname, phone, email, role, password string) (*AdminUserView, *apperrors.AppError) {
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
	normalizedNickname, appErr := validation.NormalizeNickname(nickname)
	if appErr != nil {
		return nil, appErr
	}
	normalizedPhone, appErr := validation.NormalizePhone(phone)
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
	if exists, err := s.userRepo.ExistsByPhone(ctx, normalizedPhone); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if exists {
		return nil, apperrors.ErrDuplicateEntry
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	now := time.Now()
	user := &model.User{Username: normalizedUsername, Nickname: normalizedNickname, Phone: normalizedPhone, Email: normalizedEmail, PasswordHash: hash, Role: normalizedRole, Status: model.StatusActive, PasswordChangedAt: &now}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to create user", err)
	}
	view := s.toAdminUserView(user)
	s.logger.Info("created user", zap.String("userID", view.ID), zap.String("username", view.Username))
	return &view, nil
}

// UpdateUser 更新指定用户的受管字段（昵称/手机号/邮箱/角色/状态），
// 并可选强制关闭其二步验证。邮箱与手机号变更时检查唯一性。
func (s *UserAdminService) UpdateUser(ctx context.Context, id, nickname, phone, email, role, status string, disableTwoFactor bool) *apperrors.AppError {
	users, err := s.userRepo.FindByIDs(ctx, []string{id})
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", err)
	}
	if len(users) == 0 {
		return apperrors.ErrUserNotFound
	}
	currentRole := users[0].Role
	if currentRole == "" && users[0].RoleID != 0 {
		if roleModel, roleErr := s.userRepo.FindRoleByID(ctx, users[0].RoleID); roleErr == nil && roleModel != nil {
			currentRole = model.UserRole(roleModel.Role)
		}
	}
	normalizedEmail, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	normalizedNickname, appErr := validation.NormalizeNickname(nickname)
	if appErr != nil {
		return appErr
	}
	normalizedPhone, appErr := validation.NormalizePhone(phone)
	if appErr != nil {
		return appErr
	}
	requestedRole := strings.TrimSpace(role)
	var normalizedRole model.UserRole
	if requestedRole == string(currentRole) && requestedRole == string(model.RoleSuperAdmin) {
		// 超级管理员可以修改自己的资料；保留其保留角色，不将其当作普通角色重新分配。
		normalizedRole = model.RoleSuperAdmin
	} else {
		normalizedRole, appErr = validation.NormalizeRole(requestedRole, true)
		if appErr != nil {
			return appErr
		}
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
	if existing, err := s.userRepo.FindByPhone(ctx, normalizedPhone); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if existing != nil && strconv.FormatUint(existing.ID, 10) != id {
		return apperrors.ErrDuplicateEntry
	}
	if err := s.userRepo.UpdateManagedFields(ctx, id, normalizedNickname, normalizedPhone, normalizedEmail, normalizedRole, normalizedStatus); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update user", err)
	}
	if disableTwoFactor && users[0].TwoFactorEnabled {
		if err := s.userRepo.ResetTwoFactor(ctx, id); err != nil {
			return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to disable two-factor", err)
		}
		s.logger.Info("administrator disabled user two-factor", zap.String("userID", id))
	}
	return nil
}

// BatchUpdateRole 批量将多个用户改为同一角色，返回受影响行数。
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

// BatchDelete 批量删除指定用户，返回受影响行数。
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

// BatchResetPassword 批量重置用户密码为同一新密码，返回受影响行数。
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

// toAdminUserView 将 User 模型转换为安全视图（角色字段为原始标识）。
func toAdminUserView(u *model.User) AdminUserView {
	return AdminUserView{ID: strconv.FormatUint(u.ID, 10), Username: u.Username, Nickname: u.Nickname, Phone: u.Phone, Email: u.Email, Role: string(u.Role), RoleName: u.RoleName, Status: string(u.Status), TwoFactorEnabled: u.TwoFactorEnabled, CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
}

// toAdminUserView 转换为安全视图并将角色标识解析为显示名。
func (s *UserAdminService) toAdminUserView(u *model.User) AdminUserView {
	view := toAdminUserView(u)
	if view.RoleName == "" {
		view.RoleName = s.roleName(string(u.Role))
	}
	return view
}

// roleName 将角色标识解析为显示名；无解析器时原样返回标识。
func (s *UserAdminService) roleName(identifier string) string {
	if s.roleResolver == nil {
		return identifier
	}
	return s.roleResolver.RoleNameForIdentifier(identifier)
}
