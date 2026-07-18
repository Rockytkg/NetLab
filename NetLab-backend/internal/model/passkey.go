package model

import "time"

// Passkey 存储用户拥有的一条 WebAuthn 凭证。
type Passkey struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64     `gorm:"not null;index" json:"userId"`
	CredentialID string     `gorm:"type:varchar(512);uniqueIndex;not null" json:"credentialId"`
	Credential   string     `gorm:"type:text;not null" json:"-"`
	Name         string     `gorm:"type:varchar(128);not null" json:"name"`
	SignCount    uint32     `gorm:"not null;default:0" json:"-"`
	LastUsedAt   *time.Time `gorm:"type:timestamptz" json:"lastUsedAt"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

// TableName 指定 Passkey 的数据库表名。
func (Passkey) TableName() string { return "nb_passkeys" }
