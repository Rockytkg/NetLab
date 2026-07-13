package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthBinding 记录用户账号与第三方认证身份的统一绑定关系。
//
// 登录时以 (Provider, ProviderUserID) 作为查找键定位本地用户，从而将
// 第三方身份与账号解耦。Passkey 作为 provider=passkey 的特殊第三方身份，
// 其 WebAuthn 凭证数据保存在 credential 相关字段。
type AuthBinding struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	// Provider 是提供商 ID（github、google、linuxdo、wechat、qq）。
	Provider string `gorm:"type:varchar(32);not null;uniqueIndex:idx_oauth_provider_uid" json:"provider"`
	// ProviderUserID 是提供商侧的稳定用户标识（已带提供商前缀）。
	ProviderUserID string `gorm:"type:varchar(191);not null;uniqueIndex:idx_oauth_provider_uid" json:"provider_user_id"`
	// Email 记录绑定时第三方返回的邮箱，仅供展示与审计。
	Email string `gorm:"type:varchar(255)" json:"email"`
	// CredentialID/Credential/SignCount 仅用于 Passkey。
	CredentialID string     `gorm:"type:text;uniqueIndex" json:"-"`
	Credential   string     `gorm:"type:jsonb" json:"-"`
	Name         string     `gorm:"type:varchar(128)" json:"name"`
	SignCount    uint32     `gorm:"type:bigint;not null;default:0" json:"-"`
	LastUsedAt   *time.Time `gorm:"type:timestamptz" json:"last_used_at"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (AuthBinding) TableName() string {
	return "auth_bindings"
}

// BeforeCreate 在 UUID 未设置时进行初始化。
func (b *AuthBinding) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type UserOAuthBinding = AuthBinding
