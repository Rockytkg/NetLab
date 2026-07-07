package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SystemConfig 存储键值形式的系统配置。
type SystemConfig struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key         string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:jsonb;not null" json:"value"`
	Description string    `gorm:"type:varchar(512)" json:"description,omitempty"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}

// BeforeCreate 在 UUID 未设置时进行初始化。
func (s *SystemConfig) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
