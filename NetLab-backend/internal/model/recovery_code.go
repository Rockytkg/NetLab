package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecoveryCode 是两步验证的一次性恢复码。
//
// 仅存储 SHA-256 哈希，明文仅在生成时返回给用户一次。验证时按哈希
// 定位并标记为已使用，确保每个恢复码只能使用一次。
type RecoveryCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	CodeHash  string     `gorm:"type:varchar(64);not null;index" json:"-"`
	Used      bool       `gorm:"type:boolean;not null;default:false" json:"used"`
	UsedAt    *time.Time `gorm:"type:timestamptz" json:"used_at"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (RecoveryCode) TableName() string {
	return "recovery_codes"
}

// BeforeCreate 在 UUID 未设置时进行初始化。
func (r *RecoveryCode) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
