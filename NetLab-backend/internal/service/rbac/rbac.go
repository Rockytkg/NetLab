package rbac

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
	"netlab-backend/internal/permission"
)

// 内置角色标识常量。
const (
	// RoleSuperadmin 超级管理员角色标识（管理级别 100，对外隐藏）。
	RoleSuperadmin = "superadmin"
	// RoleAdmin 管理员角色标识（管理级别 80）。
	RoleAdmin = "admin"
)

// permissionCache 是单个角色的权限缓存。all 为 true 表示拥有全部权限
// （superadmin/admin），否则按 keys 精确匹配。
type permissionCache struct {
	all  bool
	keys map[string]struct{}
}

// Service 提供 RBAC 角色与权限管理能力：角色 CRUD、权限分配、
// 鉴权判定（带内存缓存）以及默认角色/权限的幂等种子填充。
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	mu     sync.RWMutex
	cache  map[uint64]permissionCache
}

// NewService 创建 RBAC 服务并同步默认角色与权限目录到数据库。
func NewService(db *gorm.DB, logger *zap.Logger) (*Service, error) {
	s := &Service{db: db, logger: logger, cache: make(map[uint64]permissionCache)}
	if err := s.seedDefaults(context.Background()); err != nil {
		return nil, fmt.Errorf("seed authorization defaults: %w", err)
	}
	return s, nil
}

// permissionKey 拼接 resource.action 形式的权限键。
func permissionKey(resource, action string) string { return resource + "." + action }

// seedDefaults 根据实际受保护路由的权限注册表同步权限目录、内置角色（superadmin/admin）、
// 默认自定义角色（viewer）及其权限关联。
func (s *Service) seedDefaults(ctx context.Context) error {
	if err := s.syncPermissionCatalog(ctx); err != nil {
		return err
	}
	if err := s.ensureBuiltinRole(ctx, RoleSuperadmin, "超级管理员", "系统超级管理员，拥有全部权限，角色对外隐藏", 100, true); err != nil {
		return err
	}
	if err := s.ensureBuiltinRole(ctx, RoleAdmin, "管理员", "系统管理员，拥有全部管理权限", 80, false); err != nil {
		return err
	}
	if err := s.ensureCustomRole(ctx, "viewer", "访客", "仅拥有只读访问权限", []string{permission.AuthRead, permission.RBACRead, permission.SettingRead, permission.UserRead}); err != nil {
		return err
	}
	return s.populateBuiltinPermissions(ctx)
}

// syncPermissionCatalog reconciles the database with the authoritative registry.
// Stale built-in permissions and their role links are removed transactionally;
// custom permission rows, if any, are left untouched for forward compatibility.
func (s *Service) syncPermissionCatalog(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		desired := make(map[string]permission.Definition, len(permission.Catalog))
		for _, p := range permission.Catalog {
			desired[permissionKey(p.Resource, p.Action)] = p
			var existing model.Permission
			err := tx.Where("resource = ? AND action = ?", p.Resource, p.Action).First(&existing).Error
			switch err {
			case nil:
				if err := tx.Model(&existing).Updates(map[string]any{"description": p.Description, "builtin": true}).Error; err != nil {
					return err
				}
			case gorm.ErrRecordNotFound:
				if err := tx.Create(&model.Permission{Resource: p.Resource, Action: p.Action, Description: p.Description, Builtin: true}).Error; err != nil {
					return err
				}
			default:
				return err
			}
		}

		var existing []model.Permission
		if err := tx.Where("builtin = ?", true).Find(&existing).Error; err != nil {
			return err
		}
		for _, p := range existing {
			if _, ok := desired[permissionKey(p.Resource, p.Action)]; ok {
				continue
			}
			if err := tx.Where("permission_id = ?", p.ID).Delete(&model.RolePermission{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&p).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ensureCustomRole 确保存在指定标识的自定义角色，并为其授予 codes 中列出的权限。
func (s *Service) ensureCustomRole(ctx context.Context, code, name, description string, codes []string) error {
	var role model.Role
	err := s.db.WithContext(ctx).Where("role = ?", code).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		if err := s.db.WithContext(ctx).Create(&model.Role{Role: code, RoleName: name, Description: description, RoleType: model.RoleTypeCustom, ManagementLevel: 0, Version: 1}).Error; err != nil {
			return err
		}
		if err := s.db.WithContext(ctx).Where("role = ?", code).First(&role).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var perms []model.Permission
	if err := s.db.WithContext(ctx).Where("builtin = ?", true).Find(&perms).Error; err != nil {
		return err
	}
	allowed := map[string]struct{}{}
	for _, code := range codes {
		allowed[code] = struct{}{}
	}
	for _, p := range perms {
		if _, ok := allowed[permissionKey(p.Resource, p.Action)]; ok {
			if err := s.db.WithContext(ctx).Where("role_id = ? AND permission_id = ?", role.ID, p.ID).FirstOrCreate(&model.RolePermission{RoleID: role.ID, PermissionID: p.ID}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureBuiltinRole 确保存在指定标识的内置角色；已存在时刷新其名称、描述、类型、级别与隐藏标记。
// 内置角色不可通过接口修改，因此每次启动都以这里的默认值为准。
func (s *Service) ensureBuiltinRole(ctx context.Context, code, name, description string, level int, hidden bool) error {
	var role model.Role
	err := s.db.WithContext(ctx).Where("role = ?", code).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(&model.Role{Role: code, RoleName: name, Description: description, RoleType: model.RoleTypeBuiltin, ManagementLevel: level, Hidden: hidden, Version: 1}).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", role.ID).Updates(map[string]any{"role_name": name, "description": description, "role_type": model.RoleTypeBuiltin, "management_level": level, "is_hidden": hidden, "version": gorm.Expr("version + 1")}).Error
}

// populateBuiltinPermissions 为内置角色（superadmin/admin）授予目录中的全部权限。
func (s *Service) populateBuiltinPermissions(ctx context.Context) error {
	var perms []model.Permission
	if err := s.db.WithContext(ctx).Where("builtin = ?", true).Find(&perms).Error; err != nil {
		return err
	}
	for _, code := range []string{RoleSuperadmin, RoleAdmin} {
		var role model.Role
		if err := s.db.WithContext(ctx).Where("role = ?", code).First(&role).Error; err != nil {
			return err
		}
		for _, p := range perms {
			if err := s.db.WithContext(ctx).Where("role_id = ? AND permission_id = ?", role.ID, p.ID).FirstOrCreate(&model.RolePermission{RoleID: role.ID, PermissionID: p.ID}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// invalidate 清除指定角色的权限缓存。
func (s *Service) invalidate(roleID uint64) {
	s.mu.Lock()
	delete(s.cache, roleID)
	s.mu.Unlock()
}

// buildCache 从数据库构建指定角色的权限缓存；内置管理员角色拥有全部权限。
func (s *Service) buildCache(ctx context.Context, roleID uint64) (permissionCache, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return permissionCache{}, err
	}
	if role.Role == RoleSuperadmin || role.Role == RoleAdmin {
		return permissionCache{all: true, keys: map[string]struct{}{}}, nil
	}
	var rows []struct {
		Resource string
		Action   string
	}
	if err := s.db.WithContext(ctx).Table("nb_role_permissions rp").Select("p.resource, p.action").Joins("JOIN nb_permissions p ON p.id = rp.permission_id").Where("rp.role_id = ?", roleID).Scan(&rows).Error; err != nil {
		return permissionCache{}, err
	}
	keys := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		keys[permissionKey(row.Resource, row.Action)] = struct{}{}
	}
	return permissionCache{keys: keys}, nil
}

// Allow 判定角色是否拥有指定权限（permission 为 resource.action 格式）。
// 结果带缓存；缓存未命中时回源数据库。实现 middleware.Authorizer 接口。
func (s *Service) Allow(ctx context.Context, roleID string, permission string) bool {
	id, err := strconv.ParseUint(roleID, 10, 64)
	if err != nil {
		return false
	}
	s.mu.RLock()
	cached, ok := s.cache[id]
	s.mu.RUnlock()
	if !ok {
		cached, err = s.buildCache(ctx, id)
		if err != nil {
			return false
		}
		s.mu.Lock()
		s.cache[id] = cached
		s.mu.Unlock()
	}
	return cached.all || func() bool { _, ok := cached.keys[permission]; return ok }()
}

// RoleNameForIdentifier 按角色标识（如 "admin"）查询显示名；未命中时原样返回标识。
func (s *Service) RoleNameForIdentifier(identifier string) string {
	var role model.Role
	if err := s.db.Select("role_name").Where("role = ?", identifier).First(&role).Error; err != nil || role.RoleName == "" {
		return identifier
	}
	return role.RoleName
}

// RoleNameForID 按角色 ID 查询显示名；未命中时返回空字符串。
func (s *Service) RoleNameForID(roleID string) string {
	var role model.Role
	if err := s.db.Select("role_name").Where("id = ?", roleID).First(&role).Error; err != nil || role.RoleName == "" {
		return ""
	}
	return role.RoleName
}

// PermissionKeysForRoleID 返回指定角色拥有的全部权限键（resource.action 格式，已排序）。
func (s *Service) PermissionKeysForRoleID(roleID string) []string {
	id, err := strconv.ParseUint(roleID, 10, 64)
	if err != nil {
		return []string{}
	}
	var perms []model.Permission
	if err := s.db.Table("nb_permissions p").Select("p.*").Joins("JOIN nb_role_permissions rp ON rp.permission_id = p.id").Where("rp.role_id = ?", id).Order("p.resource, p.action").Find(&perms).Error; err != nil {
		return []string{}
	}
	keys := make([]string, 0, len(perms))
	for _, p := range perms {
		keys = append(keys, permissionKey(p.Resource, p.Action))
	}
	return keys
}

// ListRoles 返回管理级别不高于操作者的角色；隐藏角色仅对其自身可见。
func (s *Service) ListRoles(ctx context.Context, actorRoleID string) ([]model.Role, error) {
	level, err := s.RoleLevel(ctx, actorRoleID)
	if err != nil {
		return nil, err
	}
	var roles []model.Role
	err = s.db.WithContext(ctx).
		Where("(is_hidden = ? OR id = ?) AND management_level <= ?", false, actorRoleID, level).
		Order("management_level DESC, created_at ASC").Find(&roles).Error
	return roles, err
}

// GetRole 按 ID 查询角色；未找到时返回 (nil, nil)。
func (s *Service) GetRole(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	err := s.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &role, err
}

// CreateRole 创建自定义角色并分配权限。角色标识不允许包含空格和点号。
func (s *Service) CreateRole(ctx context.Context, code, name, description string, permissions []string) (*model.Role, error) {
	code = strings.TrimSpace(code)
	if code == "" || strings.ContainsAny(code, " .") {
		return nil, fmt.Errorf("invalid role code")
	}
	role := &model.Role{Role: code, RoleName: strings.TrimSpace(name), Description: strings.TrimSpace(description), RoleType: model.RoleTypeCustom, ManagementLevel: 0, Version: 1}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		return s.setPermissionsTx(ctx, tx, role.ID, permissions)
	}); err != nil {
		return nil, err
	}
	s.invalidate(role.ID)
	return role, nil
}

// UpdateRole 更新自定义角色的名称与描述；内置角色不可修改。
func (s *Service) UpdateRole(ctx context.Context, id, name, description string) error {
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return gorm.ErrRecordNotFound
	}
	if role.RoleType == model.RoleTypeBuiltin {
		return fmt.Errorf("built-in role is immutable")
	}
	return s.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", id).Updates(map[string]any{"role_name": strings.TrimSpace(name), "description": strings.TrimSpace(description), "version": gorm.Expr("version + 1")}).Error
}

// DeleteRole 删除自定义角色及其权限关联；内置角色或仍被用户引用的角色不可删除。
func (s *Service) DeleteRole(ctx context.Context, id string) error {
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return nil
	}
	if role.RoleType == model.RoleTypeBuiltin {
		return fmt.Errorf("built-in role is immutable")
	}
	var userCount int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("role_id = ?", role.ID).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return fmt.Errorf("role is still assigned to %d user(s)", userCount)
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Role{}, role.ID).Error
	})
}

// ListPermissions 返回权限目录中的全部内置权限，按资源和操作排序。
func (s *Service) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var perms []model.Permission
	err := s.db.WithContext(ctx).Where("builtin = ?", true).Order("resource, action").Find(&perms).Error
	return perms, err
}

// GetRolePermissions 返回指定角色当前拥有的权限列表。
func (s *Service) GetRolePermissions(ctx context.Context, id string) ([]model.Permission, error) {
	var perms []model.Permission
	err := s.db.WithContext(ctx).Table("nb_permissions p").Select("p.*").Joins("JOIN nb_role_permissions rp ON rp.permission_id = p.id").Where("rp.role_id = ?", id).Order("p.resource, p.action").Find(&perms).Error
	return perms, err
}

// setPermissionsTx 在事务中为角色写入权限关联；codes 中存在无效权限时整体失败。
func (s *Service) setPermissionsTx(ctx context.Context, tx *gorm.DB, roleID uint64, codes []string) error {
	var perms []model.Permission
	if len(codes) > 0 {
		if err := tx.WithContext(ctx).Where("builtin = ? AND (resource, action) IN ?", true, pairs(codes)).Find(&perms).Error; err != nil {
			return err
		}
	}
	if len(perms) != len(uniqueCodes(codes)) {
		return fmt.Errorf("one or more permissions are invalid")
	}
	for _, p := range perms {
		if err := tx.WithContext(ctx).Create(&model.RolePermission{RoleID: roleID, PermissionID: p.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

// pairs 将 resource.action 格式的权限键拆分为 (resource, action) 二元组列表。
func pairs(codes []string) [][]string {
	result := make([][]string, 0, len(codes))
	for _, code := range uniqueCodes(codes) {
		p := strings.SplitN(code, ".", 2)
		if len(p) == 2 {
			result = append(result, p)
		}
	}
	return result
}

// uniqueCodes 去重权限键列表，保持原有顺序。
func uniqueCodes(codes []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, code := range codes {
		if _, ok := seen[code]; !ok {
			seen[code] = struct{}{}
			out = append(out, code)
		}
	}
	return out
}

// SetRolePermissions 全量替换自定义角色的权限列表并递增版本号；内置角色不可修改。
func (s *Service) SetRolePermissions(ctx context.Context, id string, codes []string) error {
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return gorm.ErrRecordNotFound
	}
	if role.RoleType == model.RoleTypeBuiltin {
		return fmt.Errorf("built-in role is immutable")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if err := s.setPermissionsTx(ctx, tx, role.ID, codes); err != nil {
			return err
		}
		return tx.Model(&model.Role{}).Where("id = ?", role.ID).Update("version", gorm.Expr("version + 1")).Error
	}); err != nil {
		return err
	}
	s.invalidate(role.ID)
	return nil
}

// RoleLevel 返回指定角色的管理级别。
func (s *Service) RoleLevel(ctx context.Context, roleID string) (int, error) {
	var role model.Role
	id, err := strconv.ParseUint(roleID, 10, 64)
	if err != nil {
		return 0, err
	}
	err = s.db.WithContext(ctx).Select("management_level").First(&role, id).Error
	return role.ManagementLevel, err
}

// CanManageRole 判定操作者能否管理目标角色：目标角色管理级别不得高于操作者。
func (s *Service) CanManageRole(ctx context.Context, actorRoleID, targetRoleID string) bool {
	actor, err := s.RoleLevel(ctx, actorRoleID)
	if err != nil {
		return false
	}
	target, err := s.RoleLevel(ctx, targetRoleID)
	if err != nil {
		return false
	}
	return actor >= target
}

// CanManageUser 判定操作者能否管理目标用户（按用户所属角色的管理级别比较）。
func (s *Service) CanManageUser(ctx context.Context, actorRoleID, userID string) bool {
	var user model.User
	if err := s.db.WithContext(ctx).Select("role_id").First(&user, "id = ?", userID).Error; err != nil {
		return false
	}
	return s.CanManageRole(ctx, actorRoleID, strconv.FormatUint(user.RoleID, 10))
}

// CanAssignRole 判定操作者能否将指定角色标识分配给他人。
func (s *Service) CanAssignRole(ctx context.Context, actorRoleID, code string) bool {
	var role model.Role
	if err := s.db.WithContext(ctx).Select("id").Where("role = ?", code).First(&role).Error; err != nil {
		return false
	}
	return s.CanManageRole(ctx, actorRoleID, strconv.FormatUint(role.ID, 10))
}
