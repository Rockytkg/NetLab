package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// OAuthProvider 表示一个已配置的 OAuth 提供商。
type OAuthProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
	AuthURL string `json:"authUrl"`
}

// SystemConfigResult 是返回给客户端的、编译后的系统配置。
type SystemConfigResult struct {
	RegistrationEnabled bool `json:"registrationEnabled"`
	CaptchaEnabled      bool `json:"captchaEnabled"`
	PasskeyEnabled      bool `json:"passkeyEnabled"`
	// PasswordResetEnabled 控制登录页是否展示“忘记密码”入口。
	PasswordResetEnabled bool            `json:"passwordResetEnabled"`
	TwoFactorRequired    bool            `json:"twoFactorRequired"`
	OAuthProviders       []OAuthProvider `json:"oauthProviders"`
	// ICPBeian 是登录页展示的中国大陆 ICP 备案号
	//（例如 "京ICP备12345678号-1"）。未配置时为空。前端
	// 根据固定模板拼接跳转链接，因此不存储 URL。
	ICPBeian string `json:"icpBeian"`
	// PoliceBeian 是公安备案号（如有）。前端
	// 根据固定模板拼接跳转链接，因此不存储 URL。
	PoliceBeian string `json:"policeBeian"`
}

// ConfigRepository 处理系统配置。
type ConfigRepository struct {
	db *gorm.DB
}

// NewConfigRepository 创建一个新的 ConfigRepository。
func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// GetValue 按 key 返回原始配置值。
func (r *ConfigRepository) GetValue(ctx context.Context, key string) (string, error) {
	var cfg model.SystemConfig
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return cfg.Value, nil
}

// SetValue 更新或插入（upsert）一个配置值。
func (r *ConfigRepository) SetValue(ctx context.Context, key, value, description string) error {
	var cfg model.SystemConfig
	result := r.db.WithContext(ctx).Where("key = ?", key).First(&cfg)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(&model.SystemConfig{
			Key:         key,
			Value:       value,
			Description: description,
		}).Error
	}
	if result.Error != nil {
		return result.Error
	}

	cfg.Value = value
	if description != "" {
		cfg.Description = description
	}
	return r.db.WithContext(ctx).Save(&cfg).Error
}

// GetAllValues 返回所有配置的原始 key→value 映射。
func (r *ConfigRepository) GetAllValues(ctx context.Context) (map[string]string, error) {
	var configs []model.SystemConfig
	if err := r.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(configs))
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	return result, nil
}
