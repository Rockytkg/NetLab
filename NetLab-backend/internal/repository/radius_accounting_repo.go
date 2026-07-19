package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// RadiusAccountingRepository 处理 RADIUS 记账历史的数据访问。
type RadiusAccountingRepository struct {
	db *gorm.DB
}

// NewRadiusAccountingRepository 创建一个新的 RadiusAccountingRepository。
func NewRadiusAccountingRepository(db *gorm.DB) *RadiusAccountingRepository {
	return &RadiusAccountingRepository{db: db}
}

// Create 插入一条记账记录（Accounting-Start 时建档）。
func (r *RadiusAccountingRepository) Create(ctx context.Context, accounting *model.RadiusAccounting) error {
	return r.db.WithContext(ctx).Create(accounting).Error
}

// UpdateStop 结算指定会话的记账记录（Accounting-Stop）。
func (r *RadiusAccountingRepository) UpdateStop(ctx context.Context, acctSessionID string, accounting *model.RadiusAccounting) error {
	result := r.db.WithContext(ctx).
		Model(&model.RadiusAccounting{}).
		Where("acct_session_id = ?", acctSessionID).
		Updates(map[string]any{
			"acct_stop_time":       time.Now(),
			"acct_input_total":     accounting.AcctInputTotal,
			"acct_output_total":    accounting.AcctOutputTotal,
			"acct_input_packets":   accounting.AcctInputPackets,
			"acct_output_packets":  accounting.AcctOutputPackets,
			"acct_session_time":    accounting.AcctSessionTime,
			"acct_terminate_cause": accounting.AcctTerminateCause,
			"last_update":          time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no accounting record with acct_session_id = %s", acctSessionID)
	}
	return nil
}

// List 分页返回记账记录，可选按用户名与时间范围过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusAccountingRepository) List(ctx context.Context, page, size int, username string, startTime, endTime *time.Time) ([]model.RadiusAccounting, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusAccounting{})
		if username != "" {
			q = q.Where("username ILIKE ?", "%"+username+"%")
		}
		if startTime != nil {
			q = q.Where("acct_start_time >= ?", *startTime)
		}
		if endTime != nil {
			q = q.Where("acct_start_time <= ?", *endTime)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusAccounting
	if err := build().Order("acct_start_time DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// PurgeBefore 分批删除早于 cutoff 的已结算记账记录（每批 purgeBatchSize 行），
// 返回删除条数。达到批数安全阀时记录日志并返回已删部分，剩余数据留待下一轮清理。
func (r *RadiusAccountingRepository) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var total int64
	for i := 0; i < purgeMaxBatches; i++ {
		sub := r.db.Model(&model.RadiusAccounting{}).
			Select("id").
			Where("acct_stop_time IS NOT NULL AND acct_stop_time < ?", cutoff).
			Limit(purgeBatchSize)
		result := r.db.WithContext(ctx).Where("id IN (?)", sub).Delete(&model.RadiusAccounting{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
		if result.RowsAffected < purgeBatchSize {
			return total, nil
		}
	}
	log.Printf("[DB] radius accounting purge reached batch cap (%d batches), remaining rows left for next tick", purgeMaxBatches)
	return total, nil
}
