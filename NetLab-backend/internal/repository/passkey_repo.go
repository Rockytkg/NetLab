package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// PasskeyRepository 处理 WebAuthn 凭证的存储。
// 凭证数据直接存储在 User 表的 passkey_* 字段。
type PasskeyRepository struct {
	db *gorm.DB
}

// NewPasskeyRepository 创建一个新的 PasskeyRepository。
func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository {
	return &PasskeyRepository{db: db}
}

// FindByCredentialID 通过 WebAuthn 凭证 ID 查找用户。
func (r *PasskeyRepository) FindByCredentialID(ctx context.Context, credentialID string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).
		Where("passkey_credential_id = ?", credentialID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// HasPasskey 检查用户是否注册了 passkey。
func (r *PasskeyRepository) HasPasskey(ctx context.Context, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND passkey_credential_id <> ''", userID).
		Count(&count).Error
	return count > 0, err
}

// Save 保存 passkey 凭证到用户记录。
func (r *PasskeyRepository) Save(ctx context.Context, userID uint64, credentialID, credentialJSON, name string, signCount uint32) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"passkey_credential_id": credentialID,
		"passkey_credential":    credentialJSON,
		"passkey_name":          name,
		"passkey_sign_count":    signCount,
		"passkey_last_used_at":  nil,
		"updated_at":            time.Now(),
	}).Error
}

// PasskeyInfo 是存储在 User 表中的 passkey 元数据。
type PasskeyInfo struct {
	CredentialID string
	Name         string
	LastUsedAt   *time.Time
}

// CredentialData 是存储在 User 表中的 passkey 凭证原始数据。
type CredentialData struct {
	CredentialID string
	Credential   string
}

// GetCredential 读取用户当前 passkey 凭证的完整数据（ID + JSON），用于 WebAuthn 认证。
func (r *PasskeyRepository) GetCredential(ctx context.Context, userID uint64) (*CredentialData, error) {
	var data CredentialData
	row := r.db.WithContext(ctx).Model(&model.User{}).
		Select("passkey_credential_id, passkey_credential").
		Where("id = ?", userID).
		Row()
	if err := row.Scan(&data.CredentialID, &data.Credential); err != nil {
		return nil, err
	}
	if data.CredentialID == "" {
		return nil, nil
	}
	return &data, nil
}

// GetInfo 读取用户当前 passkey 凭证的元数据（若无 passkey 则返回 nil）。
func (r *PasskeyRepository) GetInfo(ctx context.Context, userID uint64) (*PasskeyInfo, error) {
	var info PasskeyInfo
	row := r.db.WithContext(ctx).Model(&model.User{}).
		Select("passkey_credential_id, passkey_name, passkey_last_used_at").
		Where("id = ?", userID).
		Row()
	if err := row.Scan(&info.CredentialID, &info.Name, &info.LastUsedAt); err != nil {
		return nil, err
	}
	if info.CredentialID == "" {
		return nil, nil
	}
	return &info, nil
}

// DeleteByUserID 清空某用户的 passkey 凭证。
func (r *PasskeyRepository) DeleteByUserID(ctx context.Context, userID uint64) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"passkey_credential_id": "",
		"passkey_credential":    "",
		"passkey_name":          "",
		"passkey_sign_count":    0,
		"passkey_last_used_at":  nil,
		"updated_at":            time.Now(),
	})
	return res.RowsAffected, res.Error
}

// UpdateSignCount 更新凭证的签名计数器与最近使用时间。
func (r *PasskeyRepository) UpdateSignCount(ctx context.Context, userID uint64, signCount uint32, credentialJSON string, lastUsed time.Time) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"passkey_sign_count":   signCount,
		"passkey_credential":   credentialJSON,
		"passkey_last_used_at": lastUsed,
		"updated_at":           time.Now(),
	}).Error
}

// DeleteCredentialByID 按 credentialID 删除 passkey 凭证（独立于 userID）。
func (r *PasskeyRepository) DeleteCredentialByID(ctx context.Context, credentialID string) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("passkey_credential_id = ?", credentialID).Updates(map[string]interface{}{
		"passkey_credential_id": "",
		"passkey_credential":    "",
		"passkey_name":          "",
		"passkey_sign_count":    0,
		"passkey_last_used_at":  nil,
		"updated_at":            time.Now(),
	})
	return res.RowsAffected, res.Error
}
