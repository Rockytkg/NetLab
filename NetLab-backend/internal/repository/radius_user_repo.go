package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"netlab-backend/internal/model"
)

// RadiusUserRepository 处理 RADIUS 认证用户与策略套餐的数据访问。
type RadiusUserRepository struct {
	db *gorm.DB
}

// NewRadiusUserRepository 创建一个新的 RadiusUserRepository。
func NewRadiusUserRepository(db *gorm.DB) *RadiusUserRepository {
	return &RadiusUserRepository{db: db}
}

// —— 认证用户 ——

// Create 创建认证用户。
func (r *RadiusUserRepository) Create(ctx context.Context, user *model.RadiusUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 全量更新认证用户（调用方负责装配字段）。
func (r *RadiusUserRepository) Update(ctx context.Context, user *model.RadiusUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 按 ID 删除认证用户。
func (r *RadiusUserRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusUser{}, id).Error
}

// GetByID 按 ID 查询认证用户；不存在时返回 (nil, nil)。
func (r *RadiusUserRepository) GetByID(ctx context.Context, id uint64) (*model.RadiusUser, error) {
	var user model.RadiusUser
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 按用户名查询认证用户；不存在时返回 (nil, nil)。
func (r *RadiusUserRepository) GetByUsername(ctx context.Context, username string) (*model.RadiusUser, error) {
	var user model.RadiusUser
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByMacAddr 按 MAC 地址查询认证用户（MAC 认证场景）；mac_addr 可能是逗号
// 分隔的多 MAC 列表，按列表元素精确匹配（大小写与 -/: 分隔符不敏感）。
// 不存在时返回 (nil, nil)。
func (r *RadiusUserRepository) GetByMacAddr(ctx context.Context, mac string) (*model.RadiusUser, error) {
	mac = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(mac), "-", ":"))
	var user model.RadiusUser
	err := r.db.WithContext(ctx).
		Where("lower(mac_addr) = ? OR (',' || lower(mac_addr) || ',') LIKE ?", mac, "%,"+mac+",%").
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// List 分页返回认证用户列表，可选按关键词（用户名/姓名/手机）与状态过滤。
// page 从 1 开始；size<=0 时使用默认值 20（上限 200）。
func (r *RadiusUserRepository) List(ctx context.Context, page, size int, keyword, status string) ([]model.RadiusUser, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusUser{})
		if keyword != "" {
			like := "%" + keyword + "%"
			q = q.Where("username ILIKE ? OR realname ILIKE ? OR mobile ILIKE ?", like, like, like)
		}
		if status != "" {
			q = q.Where("status = ?", status)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.RadiusUser
	if err := build().Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateMacAddr 更新用户最近看到的 MAC 地址（绑定学习）。
func (r *RadiusUserRepository) UpdateMacAddr(ctx context.Context, username, mac string) error {
	return r.db.WithContext(ctx).Model(&model.RadiusUser{}).
		Where("username = ?", username).Update("mac_addr", mac).Error
}

// UpdateVlanID 同时更新用户的两个 VLAN 字段（避免单列更新清零另一列）。
func (r *RadiusUserRepository) UpdateVlanID(ctx context.Context, username string, vlanid1, vlanid2 int) error {
	return r.db.WithContext(ctx).Model(&model.RadiusUser{}).
		Where("username = ?", username).
		Updates(map[string]any{"vlanid1": vlanid1, "vlanid2": vlanid2}).Error
}

// UpdateLastOnline 记录用户最近上线时间。
func (r *RadiusUserRepository) UpdateLastOnline(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Model(&model.RadiusUser{}).
		Where("username = ?", username).Update("last_online", time.Now()).Error
}

// CountByProfileID 统计引用指定套餐的用户数（删除套餐前的引用检查）。
func (r *RadiusUserRepository) CountByProfileID(ctx context.Context, profileID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RadiusUser{}).Where("profile_id = ?", profileID).Count(&count).Error
	return count, err
}

// CountByProfileIDs 按套餐 ID 集合分组统计引用用户数，返回 profileID→count
// 映射（套餐列表批量回填，避免逐行 N+1）。
func (r *RadiusUserRepository) CountByProfileIDs(ctx context.Context, ids []uint64) (map[uint64]int64, error) {
	counts := make(map[uint64]int64, len(ids))
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []struct {
		ProfileID uint64
		Count     int64
	}
	err := r.db.WithContext(ctx).Model(&model.RadiusUser{}).
		Select("profile_id, count(*) AS count").
		Where("profile_id IN ?", ids).
		Group("profile_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.ProfileID] = row.Count
	}
	return counts, nil
}

// —— 策略套餐 ——

// CreateProfile 创建套餐。
func (r *RadiusUserRepository) CreateProfile(ctx context.Context, profile *model.RadiusProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

// UpdateProfile 全量更新套餐。
func (r *RadiusUserRepository) UpdateProfile(ctx context.Context, profile *model.RadiusProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

// DeleteProfile 按 ID 删除套餐。
func (r *RadiusUserRepository) DeleteProfile(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.RadiusProfile{}, id).Error
}

// GetProfileByID 按 ID 查询套餐；不存在时返回 (nil, nil)。
func (r *RadiusUserRepository) GetProfileByID(ctx context.Context, id uint64) (*model.RadiusProfile, error) {
	var profile model.RadiusProfile
	if err := r.db.WithContext(ctx).First(&profile, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// GetProfileByName 按名称查询套餐；不存在时返回 (nil, nil)。
func (r *RadiusUserRepository) GetProfileByName(ctx context.Context, name string) (*model.RadiusProfile, error) {
	var profile model.RadiusProfile
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// GetProfilesByIDs 按 ID 集合批量查询套餐，返回 id→profile 映射
// （用户列表批量回填，避免逐行 N+1）；不存在的 ID 在映射中缺席。
func (r *RadiusUserRepository) GetProfilesByIDs(ctx context.Context, ids []uint64) (map[uint64]*model.RadiusProfile, error) {
	profiles := make(map[uint64]*model.RadiusProfile, len(ids))
	if len(ids) == 0 {
		return profiles, nil
	}
	var rows []model.RadiusProfile
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		profiles[rows[i].ID] = &rows[i]
	}
	return profiles, nil
}

// ListProfiles 分页返回套餐列表，可选按名称关键词过滤。
func (r *RadiusUserRepository) ListProfiles(ctx context.Context, page, size int, keyword string) ([]model.RadiusProfile, int64, error) {
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
		q := r.db.WithContext(ctx).Model(&model.RadiusProfile{})
		if keyword != "" {
			q = q.Where("name ILIKE ?", "%"+keyword+"%")
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var profiles []model.RadiusProfile
	if err := build().Order("id ASC").Limit(size).Offset((page - 1) * size).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}

// ListAllProfiles 返回全部套餐（供下拉选择等场景，数量级小）。
func (r *RadiusUserRepository) ListAllProfiles(ctx context.Context) ([]model.RadiusProfile, error) {
	var profiles []model.RadiusProfile
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}
