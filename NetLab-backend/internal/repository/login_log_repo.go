package repository

import (
	"context"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// LoginLogRepository 处理登录日志数据访问。
type LoginLogRepository struct {
	db *gorm.DB
}

// NewLoginLogRepository 创建一个新的 LoginLogRepository。
func NewLoginLogRepository(db *gorm.DB) *LoginLogRepository {
	return &LoginLogRepository{db: db}
}

// Create 插入一条登录日志。
func (r *LoginLogRepository) Create(ctx context.Context, entry *model.LoginLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// List 分页返回登录日志列表，可选按关键词（用户名/IP）、状态和登录方式过滤。
// 仅返回目标用户角色管理级别不超过 maxLevel 的日志；
// 无法关联到用户或角色的日志（user_id 为空、用户已删除）对所有级别可见。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *LoginLogRepository) List(ctx context.Context, maxLevel int, page, size int, keyword, status, loginType string) ([]model.LoginLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}

	var total int64
	if err := r.buildListQuery(ctx, maxLevel, keyword, status, loginType).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.LoginLog
	if err := r.buildListQuery(ctx, maxLevel, keyword, status, loginType).
		Select("nb_login_logs.*").
		Order("nb_login_logs.created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// buildListQuery 构建带过滤条件的登录日志查询。
// Count 与 Find 各自调用一次，避免复用同一 *gorm.DB 导致语句污染。
func (r *LoginLogRepository) buildListQuery(ctx context.Context, maxLevel int, keyword, status, loginType string) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.LoginLog{}).
		Joins("LEFT JOIN nb_users ON nb_users.id = nb_login_logs.user_id").
		Joins("LEFT JOIN nb_roles ON nb_roles.id = nb_users.role_id").
		Where("(nb_login_logs.user_id IS NULL OR nb_users.id IS NULL OR nb_roles.management_level <= ?)", maxLevel)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(nb_login_logs.username ILIKE ? OR nb_login_logs.ip ILIKE ?)", like, like)
	}
	if status != "" {
		q = q.Where("nb_login_logs.status = ?", status)
	}
	if loginType != "" {
		q = q.Where("nb_login_logs.login_type = ?", loginType)
	}
	return q
}

// Delete 按 ID 批量删除日志，仅删除目标用户角色管理级别不超过 maxLevel 的记录
// （与 List 的可见性规则一致），返回实际删除条数。
func (r *LoginLogRepository) Delete(ctx context.Context, maxLevel int, ids []uint64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("(user_id IS NULL OR user_id NOT IN (SELECT id FROM nb_users) OR user_id IN (SELECT u.id FROM nb_users u JOIN nb_roles r ON r.id = u.role_id WHERE r.management_level <= ?))", maxLevel).
		Delete(&model.LoginLog{})
	return result.RowsAffected, result.Error
}
