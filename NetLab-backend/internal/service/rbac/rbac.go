package rbac

import (
	"context"
	"fmt"
	"strconv"

	"github.com/casbin/casbin/v3"
	casbinmodel "github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (r.obj == p.obj || p.obj == "*" || keyMatch2(r.obj, p.obj)) && (r.act == p.act || p.act == "*" || keyMatch2(r.act, p.act))
`

// 内置权限定义（资源:操作）。
var defaultPermissions = []model.Permission{
	{Resource: "*", Action: "*", Description: "All operations on all resources"},

	{Resource: "user", Action: "create", Description: "Create users"},
	{Resource: "user", Action: "read", Description: "View users"},
	{Resource: "user", Action: "update", Description: "Update users"},
	{Resource: "user", Action: "delete", Description: "Delete users"},
	{Resource: "user", Action: "import", Description: "Import users"},
	{Resource: "user", Action: "export", Description: "Export users"},

	{Resource: "device", Action: "create", Description: "Create devices"},
	{Resource: "device", Action: "read", Description: "View devices"},
	{Resource: "device", Action: "update", Description: "Update devices"},
	{Resource: "device", Action: "delete", Description: "Delete devices"},
	{Resource: "device", Action: "import", Description: "Import devices"},
	{Resource: "device", Action: "export", Description: "Export devices"},

	{Resource: "alert", Action: "create", Description: "Create alert rules"},
	{Resource: "alert", Action: "read", Description: "View alerts"},
	{Resource: "alert", Action: "update", Description: "Update alert rules"},
	{Resource: "alert", Action: "delete", Description: "Delete alert rules"},

	{Resource: "syslog", Action: "read", Description: "View syslog"},
	{Resource: "syslog", Action: "export", Description: "Export syslog"},

	{Resource: "setting", Action: "read", Description: "View system settings"},
	{Resource: "setting", Action: "update", Description: "Update system settings"},

	{Resource: "dashboard", Action: "read", Description: "View dashboards"},
	{Resource: "dashboard", Action: "update", Description: "Configure dashboards"},

	{Resource: "audit_log", Action: "read", Description: "View audit logs"},
	{Resource: "audit_log", Action: "export", Description: "Export audit logs"},

	{Resource: "group", Action: "create", Description: "Create device groups"},
	{Resource: "group", Action: "read", Description: "View device groups"},
	{Resource: "group", Action: "update", Description: "Update device groups"},
	{Resource: "group", Action: "delete", Description: "Delete device groups"},

	{Resource: "rbac", Action: "read", Description: "View RBAC configuration"},
	{Resource: "rbac", Action: "write", Description: "Modify RBAC configuration"},
	{Resource: "auth", Action: "read", Description: "Read the current account"},
	{Resource: "auth", Action: "update", Description: "Update the current account"},
}

// 内置角色及权限映射。
type roleDef struct {
	Role        string
	RoleName    string
	Description string
	PermKeys    []string // "resource:action"
}

var defaultRoles = []roleDef{
	{
		Role: "super_admin", RoleName: "Super Administrator", Description: "Super administrator with full system access",
		PermKeys: []string{"*:*"},
	},
	{
		Role: "admin", RoleName: "Administrator", Description: "Administrator with full management access",
		PermKeys: []string{"*:*"},
	},
	{
		Role: "editor", RoleName: "Editor", Description: "Operator with full operational access",
		PermKeys: []string{
			"auth:read", "auth:update",
			"user:read",
			"device:create", "device:read", "device:update", "device:delete",
			"alert:create", "alert:read", "alert:update", "alert:delete",
			"syslog:read", "syslog:export",
			"dashboard:read", "dashboard:update",
			"group:read",
			"rbac:read",
		},
	},
	{
		Role: "viewer", RoleName: "Viewer", Description: "Read-only access to operational resources",
		PermKeys: []string{
			"auth:read", "auth:update",
			"device:read", "alert:read", "syslog:read", "dashboard:read", "group:read",
		},
	},
}

// NewEnforcer 创建基于 GORM 的 Casbin Enforcer。
// 策略表使用统一表名 nb_policies，并启用决策缓存提升并发性能。
func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	m, err := casbinmodel.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("load rbac model: %w", err)
	}

	adapter, err := gormadapter.NewAdapterByDBUseTableName(db, "nb_", "policies")
	if err != nil {
		return nil, fmt.Errorf("create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	return enforcer, nil
}

// Service 提供 RBAC 管理操作：角色/权限的 CRUD 及与 Casbin 策略的同步。
type Service struct {
	db       *gorm.DB
	enforcer *casbin.Enforcer
	logger   *zap.Logger
}

// NewService 创建 RBAC 管理服务，并种子默认角色与权限。
func NewService(db *gorm.DB, enforcer *casbin.Enforcer, logger *zap.Logger) (*Service, error) {
	s := &Service{
		db:       db,
		enforcer: enforcer,
		logger:   logger,
	}
	if err := s.seedDefaults(); err != nil {
		return nil, fmt.Errorf("seed rbac defaults: %w", err)
	}
	return s, nil
}

// Enforcer 返回底层 Casbin Enforcer 供中间件使用。
func (s *Service) Enforcer() *casbin.Enforcer {
	return s.enforcer
}

// ─── 种子数据 ──────────────────────────────────────────────────────────────

func (s *Service) seedDefaults() error {
	ctx := context.Background()
	permMap, err := s.seedPermissions(ctx)
	if err != nil {
		return err
	}
	for _, rd := range defaultRoles {
		role, err := s.seedRole(ctx, rd)
		if err != nil {
			return err
		}
		permIDs := make([]string, 0, len(rd.PermKeys))
		for _, key := range rd.PermKeys {
			if id, ok := permMap[key]; ok {
				permIDs = append(permIDs, id)
			}
		}
		if err := s.syncRolePermissions(ctx, strconv.FormatUint(role.ID, 10), permIDs); err != nil {
			return err
		}
	}
	// 首次种子时同步 Casbin 策略
	return s.SyncCasbinPolicies(ctx)
}

func (s *Service) seedPermissions(ctx context.Context) (map[string]string, error) {
	permMap := make(map[string]string, len(defaultPermissions))
	for _, p := range defaultPermissions {
		key := p.Resource + ":" + p.Action
		var existing model.Permission
		err := s.db.WithContext(ctx).Where("resource = ? AND action = ?", p.Resource, p.Action).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.WithContext(ctx).Create(&p).Error; err != nil {
				return nil, fmt.Errorf("create permission %s: %w", key, err)
			}
			permMap[key] = strconv.FormatUint(p.ID, 10)
		} else if err != nil {
			return nil, err
		} else {
			permMap[key] = strconv.FormatUint(existing.ID, 10)
		}
	}
	return permMap, nil
}

func (s *Service) seedRole(ctx context.Context, rd roleDef) (*model.Role, error) {
	var role model.Role
	err := s.db.WithContext(ctx).Where("role = ?", rd.Role).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		role = model.Role{Role: rd.Role, RoleName: rd.RoleName, Description: rd.Description}
		if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
			return nil, fmt.Errorf("create role %s: %w", rd.Role, err)
		}
	} else if err != nil {
		return nil, err
	}
	return &role, nil
}

// ─── 角色权限同步 ───────────────────────────────────────────────────────────

// syncRolePermissions 覆盖式写入角色的权限集（数据库层）。
func (s *Service) syncRolePermissions(ctx context.Context, roleID string, permIDs []string) error {
	rid, err := strconv.ParseUint(roleID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid role id: %w", err)
	}

	// 查现有关联
	var existing []model.RolePermission
	if err := s.db.WithContext(ctx).Where("role_id = ?", rid).Find(&existing).Error; err != nil {
		return err
	}
	existingMap := make(map[uint64]bool, len(existing))
	for _, rp := range existing {
		existingMap[rp.PermissionID] = true
	}

	toAdd := make([]uint64, 0)
	toRemove := make([]uint64, 0)
	newSet := make(map[uint64]bool, len(permIDs))
	for _, pidStr := range permIDs {
		pid, e := strconv.ParseUint(pidStr, 10, 64)
		if e != nil {
			return fmt.Errorf("invalid permission id: %w", e)
		}
		newSet[pid] = true
		if !existingMap[pid] {
			toAdd = append(toAdd, pid)
		}
	}
	for _, rp := range existing {
		if !newSet[rp.PermissionID] {
			toRemove = append(toRemove, rp.PermissionID)
		}
	}

	if len(toAdd) == 0 && len(toRemove) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(toRemove) > 0 {
			if err := tx.Where("role_id = ? AND permission_id IN ?", rid, toRemove).Delete(&model.RolePermission{}).Error; err != nil {
				return err
			}
		}
		for _, pid := range toAdd {
			if err := tx.Create(&model.RolePermission{RoleID: rid, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SyncCasbinPolicies 从数据库重新加载 Casbin 策略。
func (s *Service) SyncCasbinPolicies(ctx context.Context) error {
	s.enforcer.ClearPolicy()

	var rps []struct {
		RoleID   string
		Resource string
		Action   string
	}
	if err := s.db.WithContext(ctx).
		Table("nb_role_permissions").
		Select("CAST(r.id AS TEXT) AS role_id, p.resource, p.action").
		Joins("JOIN nb_roles r ON r.id = nb_role_permissions.role_id").
		Joins("JOIN nb_permissions p ON p.id = nb_role_permissions.permission_id").
		Scan(&rps).Error; err != nil {
		return fmt.Errorf("load role permissions: %w", err)
	}

	for _, rp := range rps {
		if _, err := s.enforcer.AddPolicy(rp.RoleID, rp.Resource, rp.Action); err != nil {
			return fmt.Errorf("add policy %s %s %s: %w", rp.RoleID, rp.Resource, rp.Action, err)
		}
	}

	s.logger.Info("casbin policies synced", zap.Int("count", len(rps)))

	return nil
}

// ─── 角色管理 ───────────────────────────────────────────────────────────────

func (s *Service) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	if err := s.db.WithContext(ctx).Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *Service) GetRole(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (s *Service) CreateRole(ctx context.Context, roleValue, roleName, description string) (*model.Role, error) {
	role := &model.Role{Role: roleValue, RoleName: roleName, Description: description}
	if err := s.db.WithContext(ctx).Create(role).Error; err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) UpdateRole(ctx context.Context, id, roleValue, roleName, description string) error {
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return gorm.ErrRecordNotFound
	}
	if err := s.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", id).
		Updates(map[string]interface{}{"role": roleValue, "role_name": roleName, "description": description}).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteRole(ctx context.Context, id string) error {
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Role{}, "id = ?", id).Error
	})
}

// PermKeysForRoleID 返回指定角色 ID 的权限键列表（"resource:action" 格式）。
func (s *Service) PermKeysForRoleID(roleID string) []string {
	var rps []struct {
		Resource string
		Action   string
	}
	if err := s.db.Raw(`
		SELECT DISTINCT p.resource, p.action
		FROM nb_role_permissions rp
		JOIN nb_permissions p ON p.id = rp.permission_id
		JOIN nb_roles r ON r.id = rp.role_id
		WHERE rp.role_id = ?
	`, roleID).Scan(&rps).Error; err != nil {
		s.logger.Warn("PermKeysForRoleID query failed", zap.String("roleID", roleID), zap.Error(err))
		return []string{}
	}

	keys := make([]string, 0, len(rps))
	for _, rp := range rps {
		keys = append(keys, rp.Resource+":"+rp.Action)
	}
	return keys
}

// RoleNameForID 返回角色 ID 对应的展示名称。
func (s *Service) RoleNameForID(roleID string) string {
	var role model.Role
	if err := s.db.Select("role_name").Where("id = ?", roleID).First(&role).Error; err != nil || role.RoleName == "" {
		return ""
	}
	return role.RoleName
}

// RoleNameForIdentifier 返回角色标识对应的展示名称，用于用户创建/导入等输入流程。
func (s *Service) RoleNameForIdentifier(identifier string) string {
	var role model.Role
	if err := s.db.Select("role_name").Where("role = ?", identifier).First(&role).Error; err != nil || role.RoleName == "" {
		return identifier
	}
	return role.RoleName
}

// ─── 权限管理 ───────────────────────────────────────────────────────────────

func (s *Service) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var perms []model.Permission
	if err := s.db.WithContext(ctx).Order("resource ASC, action ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (s *Service) GetRolePermissionIDs(ctx context.Context, roleID string) ([]string, error) {
	var rps []model.RolePermission
	if err := s.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&rps).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(rps))
	for i, rp := range rps {
		ids[i] = strconv.FormatUint(rp.PermissionID, 10)
	}
	return ids, nil
}

// SetRolePermissions 覆盖设置角色的权限集，并同步 Casbin 策略。
func (s *Service) SetRolePermissions(ctx context.Context, roleID string, permIDs []string) error {
	role, err := s.GetRole(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return fmt.Errorf("role not found")
	}
	if err := s.syncRolePermissions(ctx, roleID, permIDs); err != nil {
		return err
	}
	return s.SyncCasbinPolicies(ctx)
}
