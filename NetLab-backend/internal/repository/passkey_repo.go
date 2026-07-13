package repository

import (
	"context"
	"errors"
	"time"

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
	cred.Provider = "passkey"
	if cred.ProviderUserID == "" {
		cred.ProviderUserID = "passkey_" + cred.CredentialID
	}
	return r.db.WithContext(ctx).Create(cred).Error
}

// FindByCredentialID 通过 WebAuthn 凭证 ID 查找 passkey。
func (r *PasskeyRepository) FindByCredentialID(ctx context.Context, credentialID string) (*model.PasskeyCredential, error) {
	var cred model.PasskeyCredential
	if err := r.db.WithContext(ctx).Where("provider = ? AND credential_id = ?", "passkey", credentialID).First(&cred).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, "passkey").Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

// Delete 删除一个 passkey 凭证。
func (r *PasskeyRepository) Delete(ctx context.Context, credentialID string) error {
	return r.db.WithContext(ctx).Where("provider = ? AND credential_id = ?", "passkey", credentialID).Delete(&model.PasskeyCredential{}).Error
}

// DeleteByIDForUser 按主键删除属于指定用户的 passkey 凭证。
// 返回受影响行数，用于区分“无此凭证”与“非本人凭证”。
func (r *PasskeyRepository) DeleteByIDForUser(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Where("provider = ?", "passkey").
		Delete(&model.PasskeyCredential{})
	return res.RowsAffected, res.Error
}

// UpdateSignCount 更新凭证的签名计数器与最近使用时间，并同步回写
// 序列化的凭证记录（含新的 authenticator 状态）。
func (r *PasskeyRepository) UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32, credentialJSON string, lastUsed time.Time) error {
	return r.db.WithContext(ctx).Model(&model.PasskeyCredential{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sign_count":   signCount,
			"credential":   credentialJSON,
			"last_used_at": lastUsed,
		}).Error
}

// DeleteAllForUser 删除某用户的所有 passkey 凭证。
func (r *PasskeyRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, "passkey").Delete(&model.PasskeyCredential{}).Error
}

// CountByUserID 返回某用户的 passkey 凭证数量。
func (r *PasskeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PasskeyCredential{}).Where("user_id = ? AND provider = ?", userID, "passkey").Count(&count).Error
	return count, err
}
