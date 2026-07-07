package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// PasskeyRepository 处理 WebAuthn 凭证的存储。
type PasskeyRepository struct {
	db *gorm.DB
}

// NewPasskeyRepository 创建一个新的 PasskeyRepository。
func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository {
	return &PasskeyRepository{db: db}
}

// Create 存储一个新的 passkey 凭证。
func (r *PasskeyRepository) Create(ctx context.Context, cred *model.PasskeyCredential) error {
	return r.db.WithContext(ctx).Create(cred).Error
}

// FindByCredentialID 通过 WebAuthn 凭证 ID 查找 passkey。
func (r *PasskeyRepository) FindByCredentialID(ctx context.Context, credentialID string) (*model.PasskeyCredential, error) {
	var cred model.PasskeyCredential
	if err := r.db.WithContext(ctx).Where("credential_id = ?", credentialID).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// FindByUserID 返回某用户的所有 passkey 凭证。
func (r *PasskeyRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.PasskeyCredential, error) {
	var creds []model.PasskeyCredential
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

// Delete 删除一个 passkey 凭证。
func (r *PasskeyRepository) Delete(ctx context.Context, credentialID string) error {
	return r.db.WithContext(ctx).Where("credential_id = ?", credentialID).Delete(&model.PasskeyCredential{}).Error
}

// DeleteAllForUser 删除某用户的所有 passkey 凭证。
func (r *PasskeyRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.PasskeyCredential{}).Error
}

// CountByUserID 返回某用户的 passkey 凭证数量。
func (r *PasskeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PasskeyCredential{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
