package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasskeyCredential 存储 WebAuthn 凭证数据。
type PasskeyCredential struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	CredentialID    string    `gorm:"type:text;uniqueIndex;not null" json:"-"`
	PublicKey       string    `gorm:"type:text;not null" json:"-"`
	AttestationType string    `gorm:"type:varchar(64);not null" json:"-"`
	Transports      string    `gorm:"type:jsonb" json:"-"`
	Flags           string    `gorm:"type:jsonb" json:"-"`
	Authenticator   string    `gorm:"type:jsonb" json:"-"`
	CreatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

// BeforeCreate 在 UUID 未设置时进行初始化。
func (p *PasskeyCredential) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
