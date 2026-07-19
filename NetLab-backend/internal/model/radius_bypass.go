package model

import "time"

// RADIUS 免认证规则的匹配类型。
const (
	// RadiusBypassTypeMac 按终端 MAC 匹配。
	RadiusBypassTypeMac = "mac"
	// RadiusBypassTypeIP 按终端 IP（单地址或 CIDR）匹配。
	RadiusBypassTypeIP = "ip"
)

// RADIUS 免认证规则状态。
const (
	// RadiusBypassStatusEnabled 规则生效。
	RadiusBypassStatusEnabled = "enabled"
	// RadiusBypassStatusDisabled 规则停用。
	RadiusBypassStatusDisabled = "disabled"
)

// RadiusBypass 是一条免认证规则：命中启用规则的 Access-Request 直接
// Access-Accept，不做用户查询与密码校验。(type, value) 全表唯一。
type RadiusBypass struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string    `gorm:"type:varchar(8);not null;uniqueIndex:idx_radius_bypass_type_value" json:"type"`
	Value     string    `gorm:"type:varchar(128);not null;uniqueIndex:idx_radius_bypass_type_value" json:"value"`
	Status    string    `gorm:"type:varchar(16);not null;default:'enabled'" json:"status"`
	Remark    string    `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

// TableName 指定 RadiusBypass 的数据库表名。
func (RadiusBypass) TableName() string { return "nb_radius_bypass" }
