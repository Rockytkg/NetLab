package request

import "time"

// RadiusUserUpsertRequest 是创建/更新 RADIUS 认证用户的请求体。
// Password 为空表示不修改（更新场景）；创建时必填。
// ExpireTime 为空时创建默认一年后过期、更新保留原值。
type RadiusUserUpsertRequest struct {
	Username                string     `json:"username" binding:"required,min=1,max=64"`
	Password                string     `json:"password" binding:"omitempty,min=1,max=128"`
	ProfileID               *uint64    `json:"profileId"`
	ProfileLinkMode         *int       `json:"profileLinkMode" binding:"omitempty,min=0,max=1"`
	Realname                string     `json:"realname" binding:"omitempty,max=64"`
	Email                   string     `json:"email" binding:"omitempty,max=255"`
	Mobile                  string     `json:"mobile" binding:"omitempty,max=20"`
	Address                 string     `json:"address" binding:"omitempty,max=255"`
	MacAddr                 string     `json:"macAddr" binding:"omitempty,max=2048"`
	Vlanid1                 *int       `json:"vlanid1" binding:"omitempty,min=0,max=4094"`
	Vlanid2                 *int       `json:"vlanid2" binding:"omitempty,min=0,max=4094"`
	BindMac                 *bool      `json:"bindMac"`
	BindVlan                *bool      `json:"bindVlan"`
	IpAddr                  string     `json:"ipAddr" binding:"omitempty,max=64"`
	IpV6Addr                string     `json:"ipV6Addr" binding:"omitempty,max=128"`
	AddrPool                string     `json:"addrPool" binding:"omitempty,max=64"`
	IPv6PrefixPool          string     `json:"ipv6PrefixPool" binding:"omitempty,max=64"`
	DelegatedIpv6Prefix     string     `json:"delegatedIpv6Prefix" binding:"omitempty,max=128"`
	DelegatedIpv6PrefixPool string     `json:"delegatedIpv6PrefixPool" binding:"omitempty,max=64"`
	ActiveNum               *int       `json:"activeNum" binding:"omitempty,min=0"`
	UpRate                  *int       `json:"upRate" binding:"omitempty,min=0"`
	DownRate                *int       `json:"downRate" binding:"omitempty,min=0"`
	Domain                  string     `json:"domain" binding:"omitempty,max=64"`
	ExpireTime              *time.Time `json:"expireTime" binding:"omitempty"`
	Status                  string     `json:"status" binding:"omitempty,oneof=enabled disabled"`
	Remark                  string     `json:"remark" binding:"omitempty,max=255"`
}

// RadiusProfileUpsertRequest 是创建/更新策略套餐的请求体。
type RadiusProfileUpsertRequest struct {
	Name                    string `json:"name" binding:"required,min=1,max=64"`
	Status                  string `json:"status" binding:"omitempty,oneof=enabled disabled"`
	AddrPool                string `json:"addrPool" binding:"omitempty,max=64"`
	IPv6PrefixPool          string `json:"ipv6PrefixPool" binding:"omitempty,max=64"`
	DelegatedIpv6PrefixPool string `json:"delegatedIpv6PrefixPool" binding:"omitempty,max=64"`
	ActiveNum               *int   `json:"activeNum" binding:"omitempty,min=0"`
	UpRate                  *int   `json:"upRate" binding:"omitempty,min=0"`
	DownRate                *int   `json:"downRate" binding:"omitempty,min=0"`
	Domain                  string `json:"domain" binding:"omitempty,max=64"`
	BindMac                 *bool  `json:"bindMac"`
	BindVlan                *bool  `json:"bindVlan"`
	Remark                  string `json:"remark" binding:"omitempty,max=255"`
}

// RadiusNasUpsertRequest 是创建/更新 NAS 设备的请求体。
// Secret 为空表示不修改（更新场景）；创建时必填。
type RadiusNasUpsertRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=64"`
	Identifier string `json:"identifier" binding:"omitempty,max=128"`
	Hostname   string `json:"hostname" binding:"omitempty,max=128"`
	Ipaddr     string `json:"ipaddr" binding:"required,max=64"`
	Secret     string `json:"secret" binding:"omitempty,min=1,max=128"`
	CoaPort    *int   `json:"coaPort" binding:"omitempty,min=1,max=65535"`
	Model      string `json:"model" binding:"omitempty,max=64"`
	VendorCode string `json:"vendorCode" binding:"omitempty,max=32"`
	Status     string `json:"status" binding:"omitempty,oneof=enabled disabled"`
	Tags       string `json:"tags" binding:"omitempty,max=255"`
	Remark     string `json:"remark" binding:"omitempty,max=255"`
}

// RadiusIDsRequest 是批量按 ID 操作的请求体。
type RadiusIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"required,min=1,max=500"`
}

// RadiusBypassUpsertRequest 是创建/更新哑终端准入规则的请求体。
// IP 规则只接受单个 IPv4，且必须限定 NAS；两种规则均必须关联策略套餐。
type RadiusBypassUpsertRequest struct {
	Type       string     `json:"type" binding:"required,oneof=mac ip"`
	Value      string     `json:"value" binding:"required,min=1,max=128"`
	ProfileID  uint64     `json:"profileId" binding:"required,min=1"`
	NasID      *uint64    `json:"nasId" binding:"omitempty,min=1"`
	ExpireTime *time.Time `json:"expireTime"`
	Status     string     `json:"status" binding:"omitempty,oneof=enabled disabled"`
	Remark     string     `json:"remark" binding:"omitempty,max=255"`
}

// RadiusCoARequest 是 CoA 动态授权变更（会话策略下发）的请求体。
// SessionTimeout 与 FilterID 至少一项有效。
type RadiusCoARequest struct {
	SessionTimeout *int   `json:"sessionTimeout" binding:"omitempty,min=0"`
	FilterID       string `json:"filterId" binding:"omitempty,max=253"`
}

// RadiusCertUpsertRequest 是创建/更新 RADIUS TLS 证书的请求体。
// CertPEM 为 PEM 编码证书（可为 bundle）；KeyPEM 为 PEM 私钥，
// 更新时留空表示不变。创建服务器（server）证书时必须提供私钥。
type RadiusCertUpsertRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=128"`
	CertType string `json:"certType" binding:"omitempty,oneof=server ca"`
	CertPEM  string `json:"certPem"`
	KeyPEM   string `json:"keyPem"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

// RadiusSystemSettingsRequest 是 RADIUS 系统（监听器）设置的请求体。
type RadiusSystemSettingsRequest struct {
	Enabled        bool   `json:"enabled"`
	BindHost       string `json:"bindHost" binding:"required,max=64"`
	AuthPort       int    `json:"authPort" binding:"min=1,max=65535"`
	AcctPort       int    `json:"acctPort" binding:"min=1,max=65535"`
	RadsecEnabled  bool   `json:"radsecEnabled"`
	RadsecPort     int    `json:"radsecPort" binding:"omitempty,min=1,max=65535"`
	RadsecCertID   uint64 `json:"radsecCertId"`
	RadsecCACertID uint64 `json:"radsecCaCertId"`
}

// RadiusListenerSettingsRequest 是系统设置中管理的 RADIUS 基础监听配置。
// RadSec 配置仍由认证计费服务设置单独管理。
type RadiusListenerSettingsRequest struct {
	Enabled  bool   `json:"enabled"`
	BindHost string `json:"bindHost" binding:"required,max=64"`
	AuthPort int    `json:"authPort" binding:"min=1,max=65535"`
	AcctPort int    `json:"acctPort" binding:"min=1,max=65535"`
}

// RadiusServerSettingsRequest 是 RADIUS 服务器策略设置的请求体。
type RadiusServerSettingsRequest struct {
	MessageAuthMode          string `json:"messageAuthMode" binding:"required,oneof=disabled warn enforce"`
	IgnorePassword           bool   `json:"ignorePassword"`
	SessionTimeout           int    `json:"sessionTimeout" binding:"min=0"`
	AcctInterimInterval      int    `json:"acctInterimInterval" binding:"min=30"`
	HistoryDays              int    `json:"historyDays" binding:"min=0"`
	RejectDelayMaxRejects    int    `json:"rejectDelayMaxRejects" binding:"min=1,max=1000"`
	RejectDelayWindowSeconds int    `json:"rejectDelayWindowSeconds" binding:"min=1,max=3600"`
}

// RadiusEapSettingsRequest 是 RADIUS EAP（802.1X）设置的请求体。
type RadiusEapSettingsRequest struct {
	Enabled         bool   `json:"enabled"`
	Method          string `json:"method" binding:"required,oneof=eap-md5 eap-mschapv2 eap-tls eap-peap eap-ttls"`
	EnabledHandlers string `json:"enabledHandlers" binding:"omitempty,max=128"`
	TLSServerCertID uint64 `json:"tlsServerCertId"`
	TLSClientCAID   uint64 `json:"tlsClientCaId"`
	TLSMinVersion   string `json:"tlsMinVersion" binding:"required,oneof=1.2 1.3"`
}
