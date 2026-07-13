package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// UserRepository 处理用户数据访问。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建一个新的 UserRepository。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 插入一个新用户。
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID 通过 ID 获取用户。
func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername 通过用户名获取用户。
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail 通过 email 获取用户。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ExistsByUsername 检查用户名是否已被占用。
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// ExistsByEmail 检查 email 是否已被占用。
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// Update 更新用户字段。
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLoginSuccess 记录一次成功登录。
func (r *UserRepository) UpdateLoginSuccess(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
}

// UpdatePassword 修改用户密码。
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hash":         passwordHash,
			"password_changed_at":   now,
			"force_password_change": false,
			"updated_at":            now,
		}).Error
}

// List 分页返回用户列表，可选按用户名/邮箱、状态和角色过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *UserRepository) List(ctx context.Context, page, size int, keyword, status, role string) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}

	q := r.db.WithContext(ctx).Model(&model.User{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username ILIKE ? OR email ILIKE ?", like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if role != "" {
		q = q.Where("role = ?", role)
	}
	q = q.Where("username <> ?", "admin").
		Where("role <> ?", model.RoleSuperAdmin)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	if err := q.Order("created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateManagedFields 更新管理端允许编辑的用户字段。
func (r *UserRepository) UpdateManagedFields(ctx context.Context, userID, email string, role model.UserRole, status model.UserStatus) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email":      email,
			"role":       role,
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// UpdateEmail 修改单个用户邮箱。
func (r *UserRepository) UpdateEmail(ctx context.Context, userID, email string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email":              email,
			"force_email_change": false,
			"updated_at":         time.Now(),
		}).Error
}

// BatchDelete 删除一组用户。
func (r *UserRepository) BatchDelete(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.User{})
	return res.RowsAffected, res.Error
}

// FindByIDs 按主键批量获取用户。
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) ([]model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// BatchUpdateRole 批量更新用户角色。
func (r *UserRepository) BatchUpdateRole(ctx context.Context, ids []string, role model.UserRole) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"role":       role,
			"updated_at": time.Now(),
		})
	return res.RowsAffected, res.Error
}

// BatchUpdatePassword 批量重置用户密码为同一哈希，并清除锁定状态。
func (r *UserRepository) BatchUpdatePassword(ctx context.Context, ids []string, passwordHash string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"password_hash":         passwordHash,
			"password_changed_at":   time.Now(),
			"force_password_change": true,
			"updated_at":            time.Now(),
		})
	return res.RowsAffected, res.Error
}

// EnableTwoFactor 启用两步验证并保存（已加密的）TOTP 密钥。
func (r *UserRepository) EnableTwoFactor(ctx context.Context, userID, encryptedSecret string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"two_factor_secret":  encryptedSecret,
			"two_factor_enabled": true,
			"updated_at":         time.Now(),
		}).Error
}

// DisableTwoFactor 关闭两步验证并清除 TOTP 密钥。
func (r *UserRepository) DisableTwoFactor(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"two_factor_secret":  "",
			"two_factor_enabled": false,
			"updated_at":         time.Now(),
		}).Error
}

// SetPreferredAuthMethod 更新用户的两步验证首选方式（totp / passkey）。
func (r *UserRepository) SetPreferredAuthMethod(ctx context.Context, userID, method string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"preferred_auth_method": method,
			"updated_at":            time.Now(),
		}).Error
}

// DeleteRecoveryCodes 删除用户全部恢复码（关闭 2FA 或重新生成时调用）。
func (r *UserRepository) DeleteRecoveryCodes(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).
		Delete(&model.RecoveryCode{}).Error
}

// StoreRecoveryCodes 替换式写入一批恢复码哈希：先清空既有记录，再批量插入。
func (r *UserRepository) StoreRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", uid).Delete(&model.RecoveryCode{}).Error; err != nil {
			return err
		}
		if len(hashes) == 0 {
			return nil
		}
		codes := make([]model.RecoveryCode, 0, len(hashes))
		for _, h := range hashes {
			codes = append(codes, model.RecoveryCode{UserID: uid, CodeHash: h})
		}
		return tx.Create(&codes).Error
	})
}

// ConsumeRecoveryCode 原子地消费一个恢复码：仅当存在匹配且未使用的记录时
// 将其标记为已使用，返回是否成功消费。
func (r *UserRepository) ConsumeRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.RecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used = ?", userID, codeHash, false).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
