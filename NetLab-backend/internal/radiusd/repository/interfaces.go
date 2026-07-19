// Package repository 定义 RADIUS 运行时依赖的存储接口。
// 由 internal/repository 的具体 GORM 实现满足，便于运行时与插件单测注入 fake。
package repository

import (
	"context"
	"time"

	"netlab-backend/internal/model"
)

// SessionRepository 是在线会话存储。
type SessionRepository interface {
	// Create 幂等插入在线会话，返回是否新建（重传 Start 返回 false）。
	Create(ctx context.Context, session *model.RadiusOnline) (bool, error)
	// UpdateCounters 更新会话计数器与时间戳（Interim-Update）。
	UpdateCounters(ctx context.Context, session *model.RadiusOnline) error
	// Delete 按会话 ID 删除在线会话。
	Delete(ctx context.Context, acctSessionID string) error
	// GetBySessionID 按会话 ID 查询在线会话，不存在返回 (nil, nil)。
	GetBySessionID(ctx context.Context, acctSessionID string) (*model.RadiusOnline, error)
	// CountByUsername 统计用户的并发在线数。
	CountByUsername(ctx context.Context, username string) (int, error)
	// BatchDeleteByNas 清空指定 NAS 的全部在线会话。
	BatchDeleteByNas(ctx context.Context, nasAddr string) error
	// DeleteZombie 删除超过 threshold 未更新的僵尸在线会话，返回删除条数。
	DeleteZombie(ctx context.Context, threshold time.Duration) (int64, error)
}

// AccountingRepository 是记账历史存储。
type AccountingRepository interface {
	// Create 插入记账记录（Start 建档）。
	Create(ctx context.Context, accounting *model.RadiusAccounting) error
	// UpdateStop 结算会话记账记录（Stop）。
	UpdateStop(ctx context.Context, acctSessionID string, accounting *model.RadiusAccounting) error
	// PurgeBefore 删除早于 cutoff 的已结算记账记录，返回删除条数。
	PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// UserRepository 是 RADIUS 认证用户存储。
type UserRepository interface {
	// GetByUsername 按用户名查询，不存在返回 (nil, nil)。
	GetByUsername(ctx context.Context, username string) (*model.RadiusUser, error)
	// GetByMacAddr 按 MAC 查询（MAC 认证），不存在返回 (nil, nil)。
	GetByMacAddr(ctx context.Context, mac string) (*model.RadiusUser, error)
	// GetProfileByID 按 ID 查询套餐，不存在返回 (nil, nil)。
	GetProfileByID(ctx context.Context, id uint64) (*model.RadiusProfile, error)
	// UpdateMacAddr 更新用户最近 MAC（绑定学习）。
	UpdateMacAddr(ctx context.Context, username, mac string) error
	// UpdateVlanID 同时更新两个 VLAN 字段。
	UpdateVlanID(ctx context.Context, username string, vlanid1, vlanid2 int) error
	// UpdateLastOnline 记录最近上线时间。
	UpdateLastOnline(ctx context.Context, username string) error
}

// NasRepository 是 NAS 设备存储。
type NasRepository interface {
	// GetByIPOrIdentifier 按源 IP（优先）或 Identifier 匹配启用状态的 NAS，
	// 不存在返回 (nil, nil)。
	GetByIPOrIdentifier(ctx context.Context, ip, identifier string) (*model.RadiusNas, error)
}

// BypassRepository 是免认证规则存储。
type BypassRepository interface {
	// ListEnabled 返回全部启用状态的免认证规则。
	ListEnabled(ctx context.Context) ([]model.RadiusBypass, error)
}
