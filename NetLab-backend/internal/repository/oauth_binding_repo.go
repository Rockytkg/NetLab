package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// OAuthBindingRepository 处理 OAuth 绑定关系。
// 每个提供商在 User 表有独立字段对 (oauth_{provider}_id, oauth_{provider}_email)。
// 同一 provider_id 列通过 uniqueIndex 确保跨用户互斥。
type OAuthBindingRepository struct {
	db *gorm.DB
}

// NewOAuthBindingRepository 创建一个新的 OAuthBindingRepository。
func NewOAuthBindingRepository(db *gorm.DB) *OAuthBindingRepository {
	return &OAuthBindingRepository{db: db}
}

// supportedProviders 列出所有支持的 OAuth 提供商以及对应的列后缀。
var supportedProviders = []string{"github", "google", "linuxdo", "wechat", "qq"}

// oauthColumns 返回指定提供商的 ID 和 Email 列名。
func oauthColumns(provider string) (idCol, emailCol string) {
	return "oauth_" + provider + "_id", "oauth_" + provider + "_email"
}

// providerColMap 构建 provider -> {idCol, emailCol} 的映射，方便遍历时检查。
func providerColMap() map[string][2]string {
	m := make(map[string][2]string, len(supportedProviders))
	for _, p := range supportedProviders {
		idCol, emailCol := oauthColumns(p)
		m[p] = [2]string{idCol, emailCol}
	}
	return m
}

// colExists 检查 provider 对应的列是否受支持。
func colExists(provider string) bool {
	for _, p := range supportedProviders {
		if p == provider {
			return true
		}
	}
	return false
}

// FindByProviderUID 通过 (provider, providerUserID) 查找绑定的用户。
func (r *OAuthBindingRepository) FindByProviderUID(ctx context.Context, provider, providerUserID string) (*model.User, error) {
	idCol, _ := oauthColumns(provider)
	if !colExists(provider) {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
	var user model.User
	if err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ?", idCol), providerUserID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ListByUser 返回某用户已绑定的所有 OAuth 提供商。
func (r *OAuthBindingRepository) ListByUser(ctx context.Context, userID uint64) ([]model.OAuthProviderInfo, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}
	var result []model.OAuthProviderInfo
	cols := providerColMap()
	for provider, pair := range cols {
		idCol, _ := pair[0], pair[1]
		// 从 user 行中读取 provider 列的值
		var providerID string
		row := r.db.WithContext(ctx).Model(&model.User{}).
			Select(idCol).
			Where("id = ?", userID).
			Row()
		if err := row.Scan(&providerID); err != nil || providerID == "" {
			continue
		}
		result = append(result, model.OAuthProviderInfo{
			Provider:  provider,
			CreatedAt: user.CreatedAt,
		})
	}
	// 顺便读取 email
	for i, info := range result {
		_, emailCol := oauthColumns(info.Provider)
		var email string
		row := r.db.WithContext(ctx).Model(&model.User{}).
			Select(emailCol).
			Where("id = ?", userID).
			Row()
		if err := row.Scan(&email); err == nil {
			result[i].Email = email
		}
	}
	if result == nil {
		result = []model.OAuthProviderInfo{}
	}
	return result, nil
}

// HasAny 检查用户是否绑定过任意 OAuth 提供商。
func (r *OAuthBindingRepository) HasAny(ctx context.Context, userID uint64) (bool, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return false, err
	}
	cols := providerColMap()
	for provider := range cols {
		idCol, _ := oauthColumns(provider)
		var providerID string
		row := r.db.WithContext(ctx).Model(&model.User{}).
			Select(idCol).
			Where("id = ?", userID).
			Row()
		if err := row.Scan(&providerID); err != nil || providerID == "" {
			continue
		}
		return true, nil
	}
	return false, nil
}

// Bind 为用户绑定 OAuth 提供商（通过独立的 provider 列）。
// 若该第三方身份已被其他用户绑定，返回 gorm.ErrDuplicatedKey。
func (r *OAuthBindingRepository) Bind(ctx context.Context, userID uint64, provider, providerUserID, email string) error {
	idCol, emailCol := oauthColumns(provider)
	if !colExists(provider) {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	existing, err := r.FindByProviderUID(ctx, provider, providerUserID)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != userID {
		return gorm.ErrDuplicatedKey
	}

	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		idCol:    providerUserID,
		emailCol: email,
	}).Error
}

// Unbind 清除用户指定提供商的绑定。
func (r *OAuthBindingRepository) Unbind(ctx context.Context, userID uint64, provider string) error {
	idCol, emailCol := oauthColumns(provider)
	if !colExists(provider) {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		idCol:    "",
		emailCol: "",
	}).Error
}

// GetBinding 返回用户指定提供商的绑定信息。
func (r *OAuthBindingRepository) GetBinding(ctx context.Context, userID uint64, provider string) (*model.OAuthProviderInfo, error) {
	if !colExists(provider) {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
	idCol, emailCol := oauthColumns(provider)

	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	var providerID, providerEmail string
	row := r.db.WithContext(ctx).Model(&model.User{}).
		Select(fmt.Sprintf("%s, %s", idCol, emailCol)).
		Where("id = ?", userID).
		Row()
	if err := row.Scan(&providerID, &providerEmail); err != nil || providerID == "" {
		return nil, nil
	}
	return &model.OAuthProviderInfo{
		Provider:  provider,
		Email:     providerEmail,
		CreatedAt: user.CreatedAt,
	}, nil
}
