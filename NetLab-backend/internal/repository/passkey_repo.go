package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// PasskeyRepository handles WebAuthn credentials in the dedicated passkey table.
type PasskeyRepository struct {
	db *gorm.DB
}

func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository {
	return &PasskeyRepository{db: db}
}

// FindByCredentialID finds the user that owns a credential.
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

func (r *PasskeyRepository) HasPasskey(ctx context.Context, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Passkey{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

// Save creates a new credential for a user.
func (r *PasskeyRepository) Save(ctx context.Context, userID uint64, credentialID, credentialJSON, name string, signCount uint32) error {
	return r.db.WithContext(ctx).Create(&model.Passkey{
		UserID:       userID,
		CredentialID: credentialID,
		Credential:   credentialJSON,
		Name:         name,
		SignCount:    signCount,
	}).Error
}

type PasskeyInfo struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type CredentialData struct {
	CredentialID string
	Credential   string
}

// GetCredentials returns all credentials for a user for WebAuthn registration/login.
func (r *PasskeyRepository) GetCredentials(ctx context.Context, userID uint64) ([]CredentialData, error) {
	var rows []CredentialData
	err := r.db.WithContext(ctx).Model(&model.Passkey{}).
		Select("credential_id, credential").
		Where("user_id = ?", userID).
		Order("id ASC").
		Scan(&rows).Error
	return rows, err
}

// List returns passkey metadata ordered by creation time.
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

// DeleteByCredentialID removes only the selected credential owned by the user.
func (r *PasskeyRepository) DeleteByCredentialID(ctx context.Context, userID uint64, credentialID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND credential_id = ?", userID, credentialID).
		Delete(&model.Passkey{})
	return result.RowsAffected, result.Error
}

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
