package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

type PortalNasRepository struct{ db *gorm.DB }

func NewPortalNasRepository(db *gorm.DB) *PortalNasRepository { return &PortalNasRepository{db: db} }
func (r *PortalNasRepository) Create(ctx context.Context, nas *model.PortalNas) error {
	return r.db.WithContext(ctx).Create(nas).Error
}
func (r *PortalNasRepository) Update(ctx context.Context, nas *model.PortalNas) error {
	return r.db.WithContext(ctx).Save(nas).Error
}
func (r *PortalNasRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.PortalNas{}, id).Error
}
func (r *PortalNasRepository) GetByID(ctx context.Context, id uint64) (*model.PortalNas, error) {
	var nas model.PortalNas
	if err := r.db.WithContext(ctx).First(&nas, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &nas, nil
}
func (r *PortalNasRepository) GetByIdentifier(ctx context.Context, identifier string) (*model.PortalNas, error) {
	var nas model.PortalNas
	if err := r.db.WithContext(ctx).Where("identifier = ?", identifier).First(&nas).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &nas, nil
}
func (r *PortalNasRepository) ListAll(ctx context.Context) ([]model.PortalNas, error) {
	var rows []model.PortalNas
	return rows, r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
}
func (r *PortalNasRepository) List(ctx context.Context, page, size int, keyword string) ([]model.PortalNas, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	q := r.db.WithContext(ctx).Model(&model.PortalNas{})
	if keyword != "" {
		q = q.Where("name ILIKE ? OR source_ip ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PortalNas
	err := q.Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error
	return rows, total, err
}

type PortalSessionRepository struct{ db *gorm.DB }

func NewPortalSessionRepository(db *gorm.DB) *PortalSessionRepository {
	return &PortalSessionRepository{db: db}
}
func (r *PortalSessionRepository) Create(ctx context.Context, session *model.PortalSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}
func (r *PortalSessionRepository) GetByID(ctx context.Context, id string) (*model.PortalSession, error) {
	var s model.PortalSession
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
func (r *PortalSessionRepository) List(ctx context.Context, page, size int, username, nasID string) ([]model.PortalSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	q := r.db.WithContext(ctx).Model(&model.PortalSession{})
	if username != "" {
		q = q.Where("username ILIKE ?", "%"+username+"%")
	}
	if nasID != "" {
		q = q.Where("portal_nas_id::text = ?", nasID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PortalSession
	err := q.Order("authenticated_at DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error
	return rows, total, err
}
func (r *PortalSessionRepository) Terminate(ctx context.Context, id, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.PortalSession{}).Where("id = ? AND state = ?", id, model.PortalSessionActive).Updates(map[string]any{"state": model.PortalSessionTerminated, "terminated_at": now, "terminate_reason": reason, "last_seen_at": now}).Error
}
func (r *PortalSessionRepository) TerminateByNasAndClientIP(ctx context.Context, nasID uint64, clientIP, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.PortalSession{}).Where("portal_nas_id = ? AND client_ip = ? AND state = ?", nasID, clientIP, model.PortalSessionActive).Updates(map[string]any{"state": model.PortalSessionTerminated, "terminated_at": now, "terminate_reason": reason, "last_seen_at": now}).Error
}
