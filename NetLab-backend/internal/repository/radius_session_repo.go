package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"netlab-backend/internal/model"
)

// RadiusSessionRepository 处理 RADIUS 在线会话的数据访问。
type RadiusSessionRepository struct {
	db *gorm.DB
}

// NewRadiusSessionRepository 创建一个新的 RadiusSessionRepository。
func NewRadiusSessionRepository(db *gorm.DB) *RadiusSessionRepository {
	return &RadiusSessionRepository{db: db}
}

// Create 幂等插入在线会话：依赖 acct_session_id 唯一索引与
// ON CONFLICT DO NOTHING，重传的 Accounting-Start 不会产生重复行。
// 返回 created=false 表示行已存在。
func (r *RadiusSessionRepository) Create(ctx context.Context, session *model.RadiusOnline) (bool, error) {
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "acct_session_id"}},
			DoNothing: true,
		}).
		Create(session)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdateCounters 按会话 ID 更新计数器与时间戳（Interim-Update）。
func (r *RadiusSessionRepository) UpdateCounters(ctx context.Context, session *model.RadiusOnline) error {
	return r.db.WithContext(ctx).
		Model(&model.RadiusOnline{}).
		Where("acct_session_id = ?", session.AcctSessionId).
		Updates(map[string]any{
			"acct_input_total":    session.AcctInputTotal,
			"acct_output_total":   session.AcctOutputTotal,
			"acct_input_packets":  session.AcctInputPackets,
			"acct_output_packets": session.AcctOutputPackets,
			"acct_session_time":   session.AcctSessionTime,
			"last_update":         time.Now(),
		}).Error
}

// Delete 按会话 ID 删除在线会话。
func (r *RadiusSessionRepository) Delete(ctx context.Context, acctSessionID string) error {
	return r.db.WithContext(ctx).
		Where("acct_session_id = ?", acctSessionID).
		Delete(&model.RadiusOnline{}).Error
}

// DeleteByID 按主键删除在线会话。
func (r *RadiusSessionRepository) DeleteByID(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusOnline{}, id).Error
}

// GetBySessionID 按会话 ID 查询在线会话；不存在时返回 (nil, nil)。
func (r *RadiusSessionRepository) GetBySessionID(ctx context.Context, acctSessionID string) (*model.RadiusOnline, error) {
	var session model.RadiusOnline
	if err := r.db.WithContext(ctx).Where("acct_session_id = ?", acctSessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// GetByID 按主键查询在线会话；不存在时返回 (nil, nil)。
func (r *RadiusSessionRepository) GetByID(ctx context.Context, id uint64) (*model.RadiusOnline, error) {
	var session model.RadiusOnline
	if err := r.db.WithContext(ctx).First(&session, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// CountByUsername 统计指定用户的并发在线数。
func (r *RadiusSessionRepository) CountByUsername(ctx context.Context, username string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.RadiusOnline{}).
		Where("username = ?", username).
		Count(&count).Error
	return int(count), err
}

// CountByUsernames 按用户名集合分组统计在线会话数，返回 username→count 映射
// （列表页批量回填，避免逐行 N+1）。
func (r *RadiusSessionRepository) CountByUsernames(ctx context.Context, usernames []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(usernames))
	if len(usernames) == 0 {
		return counts, nil
	}
	var rows []struct {
		Username string
		Count    int64
	}
	err := r.db.WithContext(ctx).Model(&model.RadiusOnline{}).
		Select("username, count(*) AS count").
		Where("username IN ?", usernames).
		Group("username").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.Username] = row.Count
	}
	return counts, nil
}

// DeleteByUsername 删除指定用户的全部在线会话（删除认证用户后调用）。
func (r *RadiusSessionRepository) DeleteByUsername(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Where("username = ?", username).Delete(&model.RadiusOnline{}).Error
}

// BatchDeleteByNas 清空指定 NAS 的全部在线会话（Accounting-On/Off）。
func (r *RadiusSessionRepository) BatchDeleteByNas(ctx context.Context, nasAddr string) error {
	if nasAddr == "" {
		return nil
	}
	return r.db.WithContext(ctx).Where("nas_addr = ?", nasAddr).Delete(&model.RadiusOnline{}).Error
}

// List 分页返回在线会话列表，可选按用户名/NAS/MAC 过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusSessionRepository) List(ctx context.Context, page, size int, username, nasAddr, macAddr string) ([]model.RadiusOnline, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}

	build := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&model.RadiusOnline{})
		if username != "" {
			q = q.Where("username ILIKE ?", "%"+username+"%")
		}
		if nasAddr != "" {
			q = q.Where("nas_addr = ?", nasAddr)
		}
		if macAddr != "" {
			q = q.Where("mac_addr ILIKE ?", "%"+macAddr+"%")
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusOnline
	if err := build().Order("acct_start_time DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Count 返回在线会话总数。
func (r *RadiusSessionRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RadiusOnline{}).Count(&count).Error
	return count, err
}

// DeleteZombie 删除超过 threshold 未更新的僵尸在线会话，返回删除条数。
func (r *RadiusSessionRepository) DeleteZombie(ctx context.Context, threshold time.Duration) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("last_update < ?", time.Now().Add(-threshold)).
		Delete(&model.RadiusOnline{})
	return result.RowsAffected, result.Error
}
