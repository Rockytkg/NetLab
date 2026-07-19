package repository

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// 保留期清理的分批参数：单批上限避免长事务与大锁，批数上限是安全阀
// （达到上限时记录日志，剩余数据留待下一轮清理）。
const (
	purgeBatchSize  = 5000
	purgeMaxBatches = 200
)

// RadiusAuthLogRepository 处理 RADIUS 认证日志的数据访问。
type RadiusAuthLogRepository struct {
	db *gorm.DB
}

// NewRadiusAuthLogRepository 创建一个新的 RadiusAuthLogRepository。
func NewRadiusAuthLogRepository(db *gorm.DB) *RadiusAuthLogRepository {
	return &RadiusAuthLogRepository{db: db}
}

// Create 插入一条认证日志。
func (r *RadiusAuthLogRepository) Create(ctx context.Context, entry *model.RadiusAuthLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// CreateBatch 批量插入认证日志（运行时日志 worker 聚合落库）。
func (r *RadiusAuthLogRepository) CreateBatch(ctx context.Context, entries []*model.RadiusAuthLog) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(entries, len(entries)).Error
}

// List 分页返回认证日志，可选按关键词（用户名/MAC）与结果过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusAuthLogRepository) List(ctx context.Context, page, size int, keyword, result string) ([]model.RadiusAuthLog, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusAuthLog{})
		if keyword != "" {
			like := "%" + keyword + "%"
			q = q.Where("username ILIKE ? OR mac_addr ILIKE ?", like, like)
		}
		if result != "" {
			q = q.Where("result = ?", result)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusAuthLog
	if err := build().Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Delete 按 ID 批量删除日志，返回实际删除条数。
func (r *RadiusAuthLogRepository) Delete(ctx context.Context, ids []uint64) (int64, error) {
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.RadiusAuthLog{})
	return result.RowsAffected, result.Error
}

// PurgeBefore 分批删除早于 cutoff 的日志（每批 purgeBatchSize 行），返回删除条数。
// 达到批数安全阀时记录日志并返回已删部分，剩余数据留待下一轮清理。
func (r *RadiusAuthLogRepository) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var total int64
	for i := 0; i < purgeMaxBatches; i++ {
		sub := r.db.Model(&model.RadiusAuthLog{}).
			Select("id").
			Where("created_at < ?", cutoff).
			Limit(purgeBatchSize)
		result := r.db.WithContext(ctx).Where("id IN (?)", sub).Delete(&model.RadiusAuthLog{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
		if result.RowsAffected < purgeBatchSize {
			return total, nil
		}
	}
	log.Printf("[DB] radius auth log purge reached batch cap (%d batches), remaining rows left for next tick", purgeMaxBatches)
	return total, nil
}

// CountSince 统计指定时间之后的认证日志数（按结果分组供概览统计使用）。
func (r *RadiusAuthLogRepository) CountSince(ctx context.Context, since time.Time, result string) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&model.RadiusAuthLog{}).Where("created_at >= ?", since)
	if result != "" {
		q = q.Where("result = ?", result)
	}
	err := q.Count(&count).Error
	return count, err
}
