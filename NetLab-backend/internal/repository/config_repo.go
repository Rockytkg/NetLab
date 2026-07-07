package repository

import (
	"context"
	"encoding/json"
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
	AuthURL string `json:"auth_url"`
}

// SystemConfigResult 是返回给客户端的、编译后的系统配置。
type SystemConfigResult struct {
	RegistrationEnabled bool            `json:"registration_enabled"`
	CaptchaEnabled      bool            `json:"captcha_enabled"`
	PasskeyEnabled      bool            `json:"passkey_enabled"`
	OAuthProviders      []OAuthProvider `json:"oauth_providers"`
	// ICPBeian 是登录页展示的中国大陆 ICP 备案号
	//（例如 "京ICP备12345678号-1"）。未配置时为空。前端
	// 根据固定模板拼接跳转链接，因此不存储 URL。
	ICPBeian string `json:"icp_beian"`
	// PoliceBeian 是公安备案号（如有）。前端
	// 根据固定模板拼接跳转链接，因此不存储 URL。
	PoliceBeian string `json:"police_beian"`
}

// ConfigRepository 处理系统配置。
type ConfigRepository struct {
	db *gorm.DB
}

// NewConfigRepository 创建一个新的 ConfigRepository。
func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// GetSystemConfig 返回用于登录页的、编译后的系统配置。
func (r *ConfigRepository) GetSystemConfig(ctx context.Context) (*SystemConfigResult, error) {
	configs, err := r.getAll(ctx)
	if err != nil {
		return nil, err
	}

	result := &SystemConfigResult{
		RegistrationEnabled: true,
		PasskeyEnabled:      true,
	}

	if v, ok := configs["registration_enabled"]; ok {
		result.RegistrationEnabled = parseBool(v)
	}
	if v, ok := configs["captcha_enabled"]; ok {
		result.CaptchaEnabled = parseBool(v)
	}
	if v, ok := configs["passkey_enabled"]; ok {
		result.PasskeyEnabled = parseBool(v)
	}
	if v, ok := configs["oauth_providers"]; ok {
		json.Unmarshal([]byte(v), &result.OAuthProviders)
	}
	if v, ok := configs["icp_beian"]; ok {
		result.ICPBeian = parseString(v)
	}
	if v, ok := configs["police_beian"]; ok {
		result.PoliceBeian = parseString(v)
	}

	return result, nil
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

func (r *ConfigRepository) getAll(ctx context.Context) (map[string]string, error) {
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

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "\"true\""
}

// parseString 解包以 jsonb 存储的字符串值。配置值存放在
// jsonb 列中，因此普通字符串会被加引号持久化（例如 `"京ICP备..."`）。
// 若不是合法的 JSON，则回退返回原始值。
func parseString(s string) string {
	if s == "" {
		return ""
	}
	var out string
	if err := json.Unmarshal([]byte(s), &out); err == nil {
		return out
	}
	return s
}
