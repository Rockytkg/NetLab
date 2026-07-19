package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// RadiusNasRepository 处理 RADIUS 接入设备（NAS）的数据访问。
type RadiusNasRepository struct {
	db *gorm.DB
}

// NewRadiusNasRepository 创建一个新的 RadiusNasRepository。
func NewRadiusNasRepository(db *gorm.DB) *RadiusNasRepository {
	return &RadiusNasRepository{db: db}
}

// Create 创建 NAS 设备。
func (r *RadiusNasRepository) Create(ctx context.Context, nas *model.RadiusNas) error {
	return r.db.WithContext(ctx).Create(nas).Error
}

// Update 全量更新 NAS 设备。
func (r *RadiusNasRepository) Update(ctx context.Context, nas *model.RadiusNas) error {
	return r.db.WithContext(ctx).Save(nas).Error
}

// Delete 按 ID 删除 NAS 设备。
func (r *RadiusNasRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusNas{}, id).Error
}

// GetByID 按 ID 查询 NAS；不存在时返回 (nil, nil)。
func (r *RadiusNasRepository) GetByID(ctx context.Context, id uint64) (*model.RadiusNas, error) {
	var nas model.RadiusNas
	if err := r.db.WithContext(ctx).First(&nas, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &nas, nil
}

// GetByIPOrIdentifier 按源 IP（优先）或 NAS-Identifier 匹配启用状态的 NAS；
// 不存在时返回 (nil, nil)。
func (r *RadiusNasRepository) GetByIPOrIdentifier(ctx context.Context, ip, identifier string) (*model.RadiusNas, error) {
	var nas model.RadiusNas
	q := r.db.WithContext(ctx).Where("status = ?", model.RadiusNasStatusEnabled)
	if ip != "" {
		q = q.Where("ipaddr = ? OR (identifier <> '' AND identifier = ?)", ip, identifier)
	} else if identifier != "" {
		q = q.Where("identifier <> '' AND identifier = ?", identifier)
	} else {
		return nil, nil
	}
	// 源 IP 精确匹配优先于 identifier 匹配。
	if err := q.Order(gorm.Expr("CASE WHEN ipaddr = ? THEN 0 ELSE 1 END", ip)).First(&nas).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &nas, nil
}

// List 分页返回 NAS 列表，可选按关键词（名称/IP/identifier）过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusNasRepository) List(ctx context.Context, page, size int, keyword string) ([]model.RadiusNas, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusNas{})
		if keyword != "" {
			like := "%" + keyword + "%"
			q = q.Where("name ILIKE ? OR ipaddr ILIKE ? OR identifier ILIKE ?", like, like, like)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusNas
	if err := build().Order("id ASC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Count 返回 NAS 总数。
func (r *RadiusNasRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RadiusNas{}).Count(&count).Error
	return count, err
}
