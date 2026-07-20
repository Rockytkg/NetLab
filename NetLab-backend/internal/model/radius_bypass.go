package model

import "time"

// RADIUS 终端准入规则的匹配类型。
const (
	// RadiusBypassTypeMac 按终端 MAC 匹配。
	RadiusBypassTypeMac = "mac"
	// RadiusBypassTypeIP 按 NAS 上报的 Framed-IP-Address 精确匹配。
	RadiusBypassTypeIP = "ip"
)

// RADIUS 免认证规则状态。
const (
	// RadiusBypassStatusEnabled 规则生效。
	RadiusBypassStatusEnabled = "enabled"
	// RadiusBypassStatusDisabled 规则停用。
	RadiusBypassStatusDisabled = "disabled"
)

// RadiusBypass 是一条哑终端准入规则，支持 MAC 或 NAS 上报的 IPv4 精确匹配，
// 并绑定一个策略套餐。IP 规则必须限定 NasID；MAC 规则可在所有已信任 NAS 生效。
// 命中后跳过口令校验，但仍执行该套餐的并发数等策略并下发响应属性。
type RadiusBypass struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Type       string     `gorm:"type:varchar(8);not null;uniqueIndex:idx_radius_bypass_type_value" json:"type"`
	Value      string     `gorm:"type:varchar(128);not null;uniqueIndex:idx_radius_bypass_type_value" json:"value"`
	ProfileID  uint64     `gorm:"not null;default:0;index" json:"profileId"`
	NasID      *uint64    `gorm:"index" json:"nasId"`
	ExpireTime *time.Time `gorm:"type:timestamptz;index" json:"expireTime"`
	Status     string     `gorm:"type:varchar(16);not null;default:'enabled'" json:"status"`
	Remark     string     `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

// TableName 指定 RadiusBypass 的数据库表名。
func (RadiusBypass) TableName() string { return "nb_radius_bypass" }
