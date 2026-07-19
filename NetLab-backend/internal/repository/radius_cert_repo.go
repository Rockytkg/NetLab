package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// RadiusCertRepository 处理 RADIUS TLS 证书（nb_radius_certs）的数据访问。
type RadiusCertRepository struct {
	db *gorm.DB
}

// NewRadiusCertRepository 创建一个新的 RadiusCertRepository。
func NewRadiusCertRepository(db *gorm.DB) *RadiusCertRepository {
	return &RadiusCertRepository{db: db}
}

// Create 创建证书。
func (r *RadiusCertRepository) Create(ctx context.Context, cert *model.RadiusCert) error {
	return r.db.WithContext(ctx).Create(cert).Error
}

// Update 全量更新证书。
func (r *RadiusCertRepository) Update(ctx context.Context, cert *model.RadiusCert) error {
	return r.db.WithContext(ctx).Save(cert).Error
}

// Delete 按 ID 删除证书。
func (r *RadiusCertRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusCert{}, id).Error
}

// GetByID 按 ID 查询证书；不存在时返回 (nil, nil)。
func (r *RadiusCertRepository) GetByID(ctx context.Context, id uint64) (*model.RadiusCert, error) {
	var cert model.RadiusCert
	if err := r.db.WithContext(ctx).First(&cert, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

// GetByName 按名称查询证书；不存在时返回 (nil, nil)。
func (r *RadiusCertRepository) GetByName(ctx context.Context, name string) (*model.RadiusCert, error) {
	var cert model.RadiusCert
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&cert).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

// List 分页返回证书列表，可选按关键词（名称/主题）与类型过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。按 id 倒序排列。
func (r *RadiusCertRepository) List(ctx context.Context, page, size int, keyword, certType string) ([]model.RadiusCert, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusCert{})
		if keyword != "" {
			like := "%" + keyword + "%"
			q = q.Where("name ILIKE ? OR subject ILIKE ?", like, like)
		}
		if certType != "" {
			q = q.Where("cert_type = ?", certType)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RadiusCert
	if err := build().Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
