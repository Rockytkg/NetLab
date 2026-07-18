package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// PasskeyRepository 负责专用 passkey 表中 WebAuthn 凭证的数据访问。
type PasskeyRepository struct {
	db *gorm.DB
}

// NewPasskeyRepository 创建一个新的 PasskeyRepository。
func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository {
	return &PasskeyRepository{db: db}
}

// FindByCredentialID 查找拥有指定凭证的用户。
func (r *PasskeyRepository) FindByCredentialID(ctx context.Context, credentialID string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Table("nb_users AS u").
		Joins("JOIN nb_passkeys AS p ON p.user_id = u.id").
		Where("p.credential_id = ?", credentialID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// HasPasskey 报告用户是否已注册至少一个通行密钥。
func (r *PasskeyRepository) HasPasskey(ctx context.Context, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Passkey{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

// Save 为用户创建一条新的 WebAuthn 凭证记录。
func (r *PasskeyRepository) Save(ctx context.Context, userID uint64, credentialID, credentialJSON, name string, signCount uint32) error {
	return r.db.WithContext(ctx).Create(&model.Passkey{
		UserID:       userID,
		CredentialID: credentialID,
		Credential:   credentialJSON,
		Name:         name,
		SignCount:    signCount,
	}).Error
}

// PasskeyInfo 是通行密钥的列表展示信息。
type PasskeyInfo struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// CredentialData 承载一条凭证的原始数据及其数据库 ID。
type CredentialData struct {
	CredentialID string
	Credential   string
}

// GetCredentials 返回用户的全部凭证，用于 WebAuthn 注册/登录流程。
func (r *PasskeyRepository) GetCredentials(ctx context.Context, userID uint64) ([]CredentialData, error) {
	var rows []CredentialData
	err := r.db.WithContext(ctx).Model(&model.Passkey{}).
		Select("credential_id, credential").
		Where("user_id = ?", userID).
		Order("id ASC").
		Scan(&rows).Error
	return rows, err
}

// List 返回用户的通行密钥元数据，按创建时间排序。
func (r *PasskeyRepository) List(ctx context.Context, userID uint64) ([]PasskeyInfo, error) {
	var rows []model.Passkey
	if err := r.db.WithContext(ctx).Model(&model.Passkey{}).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]PasskeyInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, PasskeyInfo{
			ID:         row.CredentialID,
			Name:       row.Name,
			CreatedAt:  row.CreatedAt,
			LastUsedAt: row.LastUsedAt,
		})
	}
	return result, nil
}

// DeleteByCredentialID 仅删除该用户名下选定的凭证。
func (r *PasskeyRepository) DeleteByCredentialID(ctx context.Context, userID uint64, credentialID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND credential_id = ?", userID, credentialID).
		Delete(&model.Passkey{})
	return result.RowsAffected, result.Error
}

// UpdateSignCount 更新凭证的签名计数器（防重放检测）。
func (r *PasskeyRepository) UpdateSignCount(ctx context.Context, credentialID string, signCount uint32, credentialJSON string, lastUsed time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Passkey{}).
		Where("credential_id = ?", credentialID).
		Updates(map[string]interface{}{
			"sign_count":   signCount,
			"credential":   credentialJSON,
			"last_used_at": lastUsed,
			"updated_at":   time.Now(),
		}).Error
}
