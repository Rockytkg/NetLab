package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// RadiusBypassRepository 处理 RADIUS 免认证规则的数据访问。
type RadiusBypassRepository struct {
	db *gorm.DB
}

// NewRadiusBypassRepository 创建一个新的 RadiusBypassRepository。
func NewRadiusBypassRepository(db *gorm.DB) *RadiusBypassRepository {
	return &RadiusBypassRepository{db: db}
}

// Create 创建免认证规则。
func (r *RadiusBypassRepository) Create(ctx context.Context, rule *model.RadiusBypass) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// Update 全量更新免认证规则（调用方负责装配字段）。
func (r *RadiusBypassRepository) Update(ctx context.Context, rule *model.RadiusBypass) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 按 ID 删除免认证规则。
func (r *RadiusBypassRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusBypass{}, id).Error
}

// GetByID 按 ID 查询免认证规则；不存在时返回 (nil, nil)。
func (r *RadiusBypassRepository) GetByID(ctx context.Context, id uint64) (*model.RadiusBypass, error) {
	var rule model.RadiusBypass
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// GetByTypeValue 按 (类型, 取值) 查询免认证规则（唯一性前置检查）；
// 不存在时返回 (nil, nil)。
func (r *RadiusBypassRepository) GetByTypeValue(ctx context.Context, ruleType, value string) (*model.RadiusBypass, error) {
	var rule model.RadiusBypass
	if err := r.db.WithContext(ctx).Where("type = ? AND value = ?", ruleType, value).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// List 分页返回免认证规则列表，可选按关键词（取值/备注）过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusBypassRepository) List(ctx context.Context, page, size int, keyword string) ([]model.RadiusBypass, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusBypass{})
		if keyword != "" {
			like := "%" + keyword + "%"
			q = q.Where("value ILIKE ? OR remark ILIKE ?", like, like)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusBypass
	if err := build().Order("id ASC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListEnabled 返回全部启用状态的免认证规则（认证运行时热路径使用，数量级小）。
func (r *RadiusBypassRepository) ListEnabled(ctx context.Context) ([]model.RadiusBypass, error) {
	var items []model.RadiusBypass
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.RadiusBypassStatusEnabled).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CountByProfileID reports how many terminal-access rules reference a profile.
func (r *RadiusBypassRepository) CountByProfileID(ctx context.Context, profileID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RadiusBypass{}).
		Where("profile_id = ?", profileID).Count(&count).Error
	return count, err
}
