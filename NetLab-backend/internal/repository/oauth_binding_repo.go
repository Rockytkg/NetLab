package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// OAuthBindingRepository 处理用户与第三方 OAuth 身份的绑定关系存储。
type OAuthBindingRepository struct {
	db *gorm.DB
}

// NewOAuthBindingRepository 创建一个新的 OAuthBindingRepository。
func NewOAuthBindingRepository(db *gorm.DB) *OAuthBindingRepository {
	return &OAuthBindingRepository{db: db}
}

// Create 新增一条绑定记录。
func (r *OAuthBindingRepository) Create(ctx context.Context, binding *model.UserOAuthBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

// FindByProviderUID 通过 (provider, providerUserID) 查找绑定。
// 未找到时返回 (nil, nil)。
func (r *OAuthBindingRepository) FindByProviderUID(ctx context.Context, provider, providerUserID string) (*model.UserOAuthBinding, error) {
	var b model.UserOAuthBinding
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

// ListByUser 返回某用户的所有绑定。
func (r *OAuthBindingRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.UserOAuthBinding, error) {
	var out []model.UserOAuthBinding
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider <> ?", userID, "passkey").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ExistsForUser 检查某用户是否已绑定指定提供商。
func (r *OAuthBindingRepository) ExistsForUser(ctx context.Context, userID uuid.UUID, provider string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserOAuthBinding{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Count(&count).Error
	return count > 0, err
}

// CountByUser 返回某用户的绑定总数。
func (r *OAuthBindingRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserOAuthBinding{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// DeleteByProviderForUser 删除属于指定用户的某提供商绑定。
// 返回受影响行数，用于区分“未绑定”与“非本人绑定”。
func (r *OAuthBindingRepository) DeleteByProviderForUser(ctx context.Context, userID uuid.UUID, provider string) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&model.UserOAuthBinding{})
	return res.RowsAffected, res.Error
}
