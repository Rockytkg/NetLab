package model

// RecoveryCode 保留不删除，供 contextkeys 等引用。实际存储使用 User.RecoveryCodes (JSONB)。
// Deprecated: 恢复码已嵌入 User 模型的 JSONB 字段。
type RecoveryCode struct {
	ID       uint64 `gorm:"-"`
	UserID   uint64 `gorm:"-"`
	CodeHash string `gorm:"-"`
	Used     bool   `gorm:"-"`
}
