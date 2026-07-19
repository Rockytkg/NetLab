package model

import "time"

// RADIUS 认证用户状态。
const (
	// RadiusUserStatusEnabled 正常可用。
	RadiusUserStatusEnabled = "enabled"
	// RadiusUserStatusDisabled 停用（认证一律拒绝）。
	RadiusUserStatusDisabled = "disabled"
)

// RADIUS 用户与套餐的关联模式。
const (
	// RadiusLinkModeStatic 仅使用用户本行配置（静态快照）。
	RadiusLinkModeStatic = 0
	// RadiusLinkModeDynamic 本行字段为空/0 时动态继承套餐配置。
	RadiusLinkModeDynamic = 1
)

// NAS 状态。
const (
	// RadiusNasStatusEnabled 允许接入。
	RadiusNasStatusEnabled = "enabled"
	// RadiusNasStatusDisabled 拒绝接入。
	RadiusNasStatusDisabled = "disabled"
)

// RadiusProfile 是 RADIUS 策略套餐，定义一组可复用的限速/并发/绑定策略。
type RadiusProfile struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"type:varchar(64);not null;uniqueIndex" json:"name"`
	Status    string `gorm:"type:varchar(16);not null;default:'enabled';index" json:"status"`
	AddrPool  string `gorm:"type:varchar(64);not null;default:''" json:"addrPool"`
	ActiveNum int    `gorm:"not null;default:0" json:"activeNum"` // 并发在线数限制，0=不限
	UpRate    int    `gorm:"not null;default:0" json:"upRate"`    // 上行限速 Kbps，0=不限
	DownRate  int    `gorm:"not null;default:0" json:"downRate"`  // 下行限速 Kbps，0=不限
	Domain    string `gorm:"type:varchar(64);not null;default:''" json:"domain"`
	// IPv6PrefixPool 是 NAS 侧 IPv6 前缀池名（Framed-IPv6-Pool，用于 SLAAC）。
	IPv6PrefixPool string `gorm:"type:varchar(64);not null;default:''" json:"ipv6PrefixPool"`
	// DelegatedIpv6PrefixPool 是 DHCPv6-PD 委派前缀池名（RFC 6911），
	// 与 IPv6PrefixPool 用途不同，不可复用同一取值。
	DelegatedIpv6PrefixPool string    `gorm:"type:varchar(64);not null;default:''" json:"delegatedIpv6PrefixPool"`
	BindMac                 bool      `gorm:"not null;default:false" json:"bindMac"`
	BindVlan                bool      `gorm:"not null;default:false" json:"bindVlan"`
	Remark                  string    `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt               time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt               time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`

	// 运行时回填字段，不落库。
	UserCount int64 `gorm:"-" json:"userCount"`
}

// TableName 指定 RadiusProfile 的数据库表名。
func (RadiusProfile) TableName() string { return "nb_radius_profiles" }

// RadiusUser 是 RADIUS 认证用户，与后台登录用户（nb_users）完全独立。
// Password 为 AES-256-GCM 密文（运行时解密），因此可支持 PAP/CHAP/MS-CHAPv2/EAP。
type RadiusUser struct {
	ID              uint64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Username        string  `gorm:"type:varchar(64);not null;uniqueIndex" json:"username"`
	Password        string  `gorm:"type:varchar(255);not null;default:''" json:"-"`
	ProfileID       *uint64 `gorm:"index" json:"profileId"`
	ProfileLinkMode int     `gorm:"not null;default:1" json:"profileLinkMode"`
	Realname        string  `gorm:"type:varchar(64);not null;default:''" json:"realname"`
	Email           string  `gorm:"type:varchar(255);not null;default:''" json:"email"`
	Mobile          string  `gorm:"type:varchar(20);not null;default:''" json:"mobile"`
	Address         string  `gorm:"type:varchar(255);not null;default:''" json:"address"`
	// MacAddr 是逗号分隔的绑定 MAC 列表（归一化为小写冒号分隔，见 NormalizeMacList）。
	MacAddr   string `gorm:"type:varchar(1024);not null;default:''" json:"macAddr"`
	Vlanid1   int    `gorm:"not null;default:0" json:"vlanid1"`
	Vlanid2   int    `gorm:"not null;default:0" json:"vlanid2"`
	BindMac   bool   `gorm:"not null;default:false" json:"bindMac"`
	BindVlan  bool   `gorm:"not null;default:false" json:"bindVlan"`
	IpAddr    string `gorm:"type:varchar(64);not null;default:''" json:"ipAddr"`
	IpV6Addr  string `gorm:"type:varchar(128);not null;default:''" json:"ipV6Addr"`
	AddrPool  string `gorm:"type:varchar(64);not null;default:''" json:"addrPool"`
	ActiveNum int    `gorm:"not null;default:0" json:"activeNum"`
	UpRate    int    `gorm:"not null;default:0" json:"upRate"`
	DownRate  int    `gorm:"not null;default:0" json:"downRate"`
	Domain    string `gorm:"type:varchar(64);not null;default:''" json:"domain"`
	// IPv6PrefixPool 是 NAS 侧 IPv6 前缀池名。
	IPv6PrefixPool string `gorm:"type:varchar(64);not null;default:''" json:"ipv6PrefixPool"`
	// DelegatedIpv6Prefix 是静态委派给用户的 IPv6 前缀（RFC 4818）。
	DelegatedIpv6Prefix string `gorm:"type:varchar(128);not null;default:''" json:"delegatedIpv6Prefix"`
	// DelegatedIpv6PrefixPool 是 DHCPv6-PD 委派前缀池名（RFC 6911）。
	DelegatedIpv6PrefixPool string     `gorm:"type:varchar(64);not null;default:''" json:"delegatedIpv6PrefixPool"`
	ExpireTime              time.Time  `gorm:"type:timestamptz;not null;index" json:"expireTime"`
	Status                  string     `gorm:"type:varchar(16);not null;default:'enabled';index" json:"status"`
	Remark                  string     `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	LastOnline              *time.Time `gorm:"type:timestamptz" json:"lastOnline"`
	CreatedAt               time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt               time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`

	// 运行时回填字段，不落库。
	Profile     *RadiusProfile `gorm:"-" json:"-"`
	OnlineCount int            `gorm:"-" json:"onlineCount"`
}

// TableName 指定 RadiusUser 的数据库表名。
func (RadiusUser) TableName() string { return "nb_radius_users" }

// —— 套餐继承取值器：本行字段优先，为空/0 且动态关联模式时回落到套餐配置。——

// GetUpRate 返回生效的上行限速（Kbps）。
func (u *RadiusUser) GetUpRate() int {
	if u.UpRate > 0 || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.UpRate
	}
	return u.Profile.UpRate
}

// GetDownRate 返回生效的下行限速（Kbps）。
func (u *RadiusUser) GetDownRate() int {
	if u.DownRate > 0 || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.DownRate
	}
	return u.Profile.DownRate
}

// GetActiveNum 返回生效的并发在线数限制。
func (u *RadiusUser) GetActiveNum() int {
	if u.ActiveNum > 0 || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.ActiveNum
	}
	return u.Profile.ActiveNum
}

// GetAddrPool 返回生效的地址池名。
func (u *RadiusUser) GetAddrPool() string {
	if u.AddrPool != "" || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.AddrPool
	}
	return u.Profile.AddrPool
}

// GetDomain 返回生效的域（如华为 domain）。
func (u *RadiusUser) GetDomain() string {
	if u.Domain != "" || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.Domain
	}
	return u.Profile.Domain
}

// GetBindMac 返回是否启用 MAC 绑定校验。
func (u *RadiusUser) GetBindMac() bool {
	if u.BindMac || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.BindMac
	}
	return u.Profile.BindMac
}

// GetBindVlan 返回是否启用 VLAN 绑定校验。
func (u *RadiusUser) GetBindVlan() bool {
	if u.BindVlan || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.BindVlan
	}
	return u.Profile.BindVlan
}

// GetIPv6PrefixPool 返回生效的 IPv6 前缀池名。
func (u *RadiusUser) GetIPv6PrefixPool() string {
	if u.IPv6PrefixPool != "" || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.IPv6PrefixPool
	}
	return u.Profile.IPv6PrefixPool
}

// GetDelegatedIpv6PrefixPool 返回生效的 DHCPv6-PD 委派前缀池名。
func (u *RadiusUser) GetDelegatedIpv6PrefixPool() string {
	if u.DelegatedIpv6PrefixPool != "" || u.ProfileLinkMode == RadiusLinkModeStatic || u.Profile == nil {
		return u.DelegatedIpv6PrefixPool
	}
	return u.Profile.DelegatedIpv6PrefixPool
}

// RadiusNas 是一台 RADIUS 接入设备（NAS/BRAS/AC/交换机）。
// Secret 为 AES-256-GCM 密文存储。
type RadiusNas struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"type:varchar(64);not null;default:''" json:"name"`
	Identifier string    `gorm:"type:varchar(128);not null;default:'';index" json:"identifier"` // NAS-Identifier 兜底匹配
	Hostname   string    `gorm:"type:varchar(128);not null;default:''" json:"hostname"`
	Ipaddr     string    `gorm:"type:varchar(64);not null;default:'';index" json:"ipaddr"` // 按源 IP 匹配
	Secret     string    `gorm:"type:varchar(255);not null;default:''" json:"-"`
	CoaPort    int       `gorm:"not null;default:3799" json:"coaPort"`
	Model      string    `gorm:"type:varchar(64);not null;default:''" json:"model"`
	VendorCode string    `gorm:"type:varchar(32);not null;default:''" json:"vendorCode"`
	Status     string    `gorm:"type:varchar(16);not null;default:'enabled'" json:"status"`
	Tags       string    `gorm:"type:varchar(255);not null;default:''" json:"tags"`
	Remark     string    `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

// TableName 指定 RadiusNas 的数据库表名。
func (RadiusNas) TableName() string { return "nb_radius_nas" }
