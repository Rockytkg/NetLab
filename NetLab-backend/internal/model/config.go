package model

import (
	"time"
)

// SystemConfig 存储键值形式的系统配置。
type SystemConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:jsonb;not null" json:"value"`
	Description string    `gorm:"type:varchar(512)" json:"description,omitempty"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (SystemConfig) TableName() string { return "nb_system_configs" }
