package model

import "time"

// RADIUS 认证结果。
const (
	// RadiusAuthResultAccept 认证通过。
	RadiusAuthResultAccept = "accept"
	// RadiusAuthResultReject 认证拒绝。
	RadiusAuthResultReject = "reject"
)

// RadiusOnline 表示一条在线会话（由记账报文驱动建立与更新）。
type RadiusOnline struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username            string    `gorm:"type:varchar(64);not null;default:'';index" json:"username"`
	NasId               string    `gorm:"type:varchar(128);not null;default:''" json:"nasId"`        // NAS-Identifier
	NasAddr             string    `gorm:"type:varchar(64);not null;default:'';index" json:"nasAddr"` // 配置中的 NAS IP
	NasPaddr            string    `gorm:"type:varchar(64);not null;default:''" json:"nasPaddr"`      // 报文实际源 IP
	SessionTimeout      int       `gorm:"not null;default:0" json:"sessionTimeout"`
	FramedIpaddr        string    `gorm:"type:varchar(64);not null;default:''" json:"framedIpaddr"`
	FramedNetmask       string    `gorm:"type:varchar(64);not null;default:''" json:"framedNetmask"`
	FramedIpv6Prefix    string    `gorm:"type:varchar(128);not null;default:''" json:"framedIpv6Prefix"`
	FramedIpv6Address   string    `gorm:"type:varchar(128);not null;default:''" json:"framedIpv6Address"`
	DelegatedIpv6Prefix string    `gorm:"type:varchar(128);not null;default:''" json:"delegatedIpv6Prefix"`
	MacAddr             string    `gorm:"type:varchar(32);not null;default:''" json:"macAddr"`
	NasPort             int64     `gorm:"not null;default:0" json:"nasPort"`
	NasClass            string    `gorm:"type:varchar(128);not null;default:''" json:"nasClass"` // Class 属性（NAS 回传）
	NasPortId           string    `gorm:"type:varchar(128);not null;default:''" json:"nasPortId"`
	NasPortType         int       `gorm:"not null;default:0" json:"nasPortType"`
	ServiceType         int       `gorm:"not null;default:0" json:"serviceType"`
	AcctSessionId       string    `gorm:"type:varchar(64);not null;default:'';uniqueIndex" json:"acctSessionId"`
	AcctSessionTime     int64     `gorm:"not null;default:0" json:"acctSessionTime"`
	AcctInputTotal      int64     `gorm:"not null;default:0" json:"acctInputTotal"`
	AcctOutputTotal     int64     `gorm:"not null;default:0" json:"acctOutputTotal"`
	AcctInputPackets    int64     `gorm:"not null;default:0" json:"acctInputPackets"`
	AcctOutputPackets   int64     `gorm:"not null;default:0" json:"acctOutputPackets"`
	AcctStartTime       time.Time `gorm:"type:timestamptz;not null;default:now();index" json:"acctStartTime"`
	LastUpdate          time.Time `gorm:"type:timestamptz;not null;default:now()" json:"lastUpdate"`
}

// TableName 指定 RadiusOnline 的数据库表名。
func (RadiusOnline) TableName() string { return "nb_radius_online" }

// RadiusAccounting 是一条已归档的记账记录（会话历史）。
type RadiusAccounting struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Username            string `gorm:"type:varchar(64);not null;default:'';index" json:"username"`
	NasId               string `gorm:"type:varchar(128);not null;default:''" json:"nasId"`
	NasAddr             string `gorm:"type:varchar(64);not null;default:''" json:"nasAddr"`
	NasPaddr            string `gorm:"type:varchar(64);not null;default:''" json:"nasPaddr"`
	SessionTimeout      int    `gorm:"not null;default:0" json:"sessionTimeout"`
	FramedIpaddr        string `gorm:"type:varchar(64);not null;default:''" json:"framedIpaddr"`
	FramedNetmask       string `gorm:"type:varchar(64);not null;default:''" json:"framedNetmask"`
	FramedIpv6Prefix    string `gorm:"type:varchar(128);not null;default:''" json:"framedIpv6Prefix"`
	FramedIpv6Address   string `gorm:"type:varchar(128);not null;default:''" json:"framedIpv6Address"`
	DelegatedIpv6Prefix string `gorm:"type:varchar(128);not null;default:''" json:"delegatedIpv6Prefix"`
	MacAddr             string `gorm:"type:varchar(32);not null;default:''" json:"macAddr"`
	NasPort             int64  `gorm:"not null;default:0" json:"nasPort"`
	NasClass            string `gorm:"type:varchar(128);not null;default:''" json:"nasClass"`
	NasPortId           string `gorm:"type:varchar(128);not null;default:''" json:"nasPortId"`
	NasPortType         int    `gorm:"not null;default:0" json:"nasPortType"`
	ServiceType         int    `gorm:"not null;default:0" json:"serviceType"`
	AcctSessionId       string `gorm:"type:varchar(64);not null;default:'';uniqueIndex" json:"acctSessionId"`
	AcctSessionTime     int64  `gorm:"not null;default:0" json:"acctSessionTime"`
	AcctInputTotal      int64  `gorm:"not null;default:0" json:"acctInputTotal"`
	AcctOutputTotal     int64  `gorm:"not null;default:0" json:"acctOutputTotal"`
	AcctInputPackets    int64  `gorm:"not null;default:0" json:"acctInputPackets"`
	AcctOutputPackets   int64  `gorm:"not null;default:0" json:"acctOutputPackets"`
	// AcctTerminateCause 是会话终止原因（RFC 2866 Acct-Terminate-Cause，仅存字符串名）。
	AcctTerminateCause string     `gorm:"type:varchar(64);not null;default:''" json:"acctTerminateCause"`
	AcctStartTime      time.Time  `gorm:"type:timestamptz;not null;default:now();index" json:"acctStartTime"`
	AcctStopTime       *time.Time `gorm:"type:timestamptz;index" json:"acctStopTime"`
	LastUpdate         time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"lastUpdate"`
}

// TableName 指定 RadiusAccounting 的数据库表名。
func (RadiusAccounting) TableName() string { return "nb_radius_accounting" }

// RadiusAuthLog 是一条 RADIUS 认证请求日志。
type RadiusAuthLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(64);not null;default:'';index" json:"username"`
	NasAddr   string    `gorm:"type:varchar(64);not null;default:''" json:"nasAddr"`
	NasPaddr  string    `gorm:"type:varchar(64);not null;default:''" json:"nasPaddr"`
	MacAddr   string    `gorm:"type:varchar(32);not null;default:''" json:"macAddr"`
	AuthType  string    `gorm:"type:varchar(16);not null;default:''" json:"authType"` // pap/chap/mschap/mac/eap
	Result    string    `gorm:"type:varchar(16);not null;default:'';index" json:"result"`
	Reason    string    `gorm:"type:varchar(255);not null;default:''" json:"reason"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();index" json:"createdAt"`
}

// TableName 指定 RadiusAuthLog 的数据库表名。
func (RadiusAuthLog) TableName() string { return "nb_radius_auth_logs" }
