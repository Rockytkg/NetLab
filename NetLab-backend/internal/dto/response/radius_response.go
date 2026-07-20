package response

import "time"

// RadiusUserItem 是 RADIUS 认证用户的列表/详情视图。
type RadiusUserItem struct {
	ID                      uint64     `json:"id"`
	Username                string     `json:"username"`
	ProfileID               *uint64    `json:"profileId"`
	ProfileName             string     `json:"profileName"`
	ProfileLinkMode         int        `json:"profileLinkMode"`
	Realname                string     `json:"realname"`
	Email                   string     `json:"email"`
	Mobile                  string     `json:"mobile"`
	Address                 string     `json:"address"`
	MacAddr                 string     `json:"macAddr"`
	Vlanid1                 int        `json:"vlanid1"`
	Vlanid2                 int        `json:"vlanid2"`
	BindMac                 bool       `json:"bindMac"`
	BindVlan                bool       `json:"bindVlan"`
	IpAddr                  string     `json:"ipAddr"`
	IpV6Addr                string     `json:"ipV6Addr"`
	AddrPool                string     `json:"addrPool"`
	IPv6PrefixPool          string     `json:"ipv6PrefixPool"`
	DelegatedIpv6Prefix     string     `json:"delegatedIpv6Prefix"`
	DelegatedIpv6PrefixPool string     `json:"delegatedIpv6PrefixPool"`
	ActiveNum               int        `json:"activeNum"`
	UpRate                  int        `json:"upRate"`
	DownRate                int        `json:"downRate"`
	Domain                  string     `json:"domain"`
	ExpireTime              time.Time  `json:"expireTime"`
	Status                  string     `json:"status"`
	Remark                  string     `json:"remark"`
	OnlineCount             int        `json:"onlineCount"`
	LastOnline              *time.Time `json:"lastOnline"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

// RadiusUserListResult 是认证用户分页结果。
type RadiusUserListResult struct {
	Items []RadiusUserItem `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// RadiusProfileItem 是策略套餐视图。
type RadiusProfileItem struct {
	ID                      uint64    `json:"id"`
	Name                    string    `json:"name"`
	Status                  string    `json:"status"`
	AddrPool                string    `json:"addrPool"`
	IPv6PrefixPool          string    `json:"ipv6PrefixPool"`
	DelegatedIpv6PrefixPool string    `json:"delegatedIpv6PrefixPool"`
	ActiveNum               int       `json:"activeNum"`
	UpRate                  int       `json:"upRate"`
	DownRate                int       `json:"downRate"`
	Domain                  string    `json:"domain"`
	BindMac                 bool      `json:"bindMac"`
	BindVlan                bool      `json:"bindVlan"`
	Remark                  string    `json:"remark"`
	UserCount               int64     `json:"userCount"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

// RadiusProfileListResult 是套餐分页结果。
type RadiusProfileListResult struct {
	Items []RadiusProfileItem `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// RadiusProfileOption 是套餐下拉选项。
type RadiusProfileOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// RadiusNasItem 是 NAS 设备视图（Secret 永不下发）。
type RadiusNasItem struct {
	ID         uint64    `json:"id"`
	Name       string    `json:"name"`
	Identifier string    `json:"identifier"`
	Hostname   string    `json:"hostname"`
	Ipaddr     string    `json:"ipaddr"`
	CoaPort    int       `json:"coaPort"`
	Model      string    `json:"model"`
	VendorCode string    `json:"vendorCode"`
	Status     string    `json:"status"`
	Tags       string    `json:"tags"`
	Remark     string    `json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// RadiusNasListResult 是 NAS 分页结果。
type RadiusNasListResult struct {
	Items []RadiusNasItem `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// RadiusSessionItem 是在线会话视图。
type RadiusSessionItem struct {
	ID                  uint64    `json:"id"`
	Username            string    `json:"username"`
	NasId               string    `json:"nasId"`
	NasAddr             string    `json:"nasAddr"`
	NasPaddr            string    `json:"nasPaddr"`
	NasClass            string    `json:"nasClass"`
	NasPort             int64     `json:"nasPort"`
	NasPortId           string    `json:"nasPortId"`
	NasPortType         int       `json:"nasPortType"`
	ServiceType         int       `json:"serviceType"`
	FramedIpaddr        string    `json:"framedIpaddr"`
	FramedNetmask       string    `json:"framedNetmask"`
	FramedIpv6Prefix    string    `json:"framedIpv6Prefix"`
	FramedIpv6Address   string    `json:"framedIpv6Address"`
	DelegatedIpv6Prefix string    `json:"delegatedIpv6Prefix"`
	MacAddr             string    `json:"macAddr"`
	AcctSessionId       string    `json:"acctSessionId"`
	AcctSessionTime     int64     `json:"acctSessionTime"`
	AcctInputTotal      int64     `json:"acctInputTotal"`
	AcctOutputTotal     int64     `json:"acctOutputTotal"`
	AcctInputPackets    int64     `json:"acctInputPackets"`
	AcctOutputPackets   int64     `json:"acctOutputPackets"`
	AcctStartTime       time.Time `json:"acctStartTime"`
	LastUpdate          time.Time `json:"lastUpdate"`
}

// RadiusSessionListResult 是在线会话分页结果。
type RadiusSessionListResult struct {
	Items []RadiusSessionItem `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// RadiusAccountingItem 是记账记录视图。
type RadiusAccountingItem struct {
	ID                  uint64     `json:"id"`
	Username            string     `json:"username"`
	NasId               string     `json:"nasId"`
	NasAddr             string     `json:"nasAddr"`
	NasPaddr            string     `json:"nasPaddr"`
	NasClass            string     `json:"nasClass"`
	NasPort             int64      `json:"nasPort"`
	NasPortId           string     `json:"nasPortId"`
	NasPortType         int        `json:"nasPortType"`
	ServiceType         int        `json:"serviceType"`
	FramedIpaddr        string     `json:"framedIpaddr"`
	FramedNetmask       string     `json:"framedNetmask"`
	FramedIpv6Prefix    string     `json:"framedIpv6Prefix"`
	FramedIpv6Address   string     `json:"framedIpv6Address"`
	DelegatedIpv6Prefix string     `json:"delegatedIpv6Prefix"`
	MacAddr             string     `json:"macAddr"`
	AcctSessionId       string     `json:"acctSessionId"`
	AcctSessionTime     int64      `json:"acctSessionTime"`
	AcctInputTotal      int64      `json:"acctInputTotal"`
	AcctOutputTotal     int64      `json:"acctOutputTotal"`
	AcctInputPackets    int64      `json:"acctInputPackets"`
	AcctOutputPackets   int64      `json:"acctOutputPackets"`
	AcctTerminateCause  string     `json:"acctTerminateCause"`
	AcctStartTime       time.Time  `json:"acctStartTime"`
	AcctStopTime        *time.Time `json:"acctStopTime"`
}

// RadiusAccountingListResult 是记账记录分页结果。
type RadiusAccountingListResult struct {
	Items []RadiusAccountingItem `json:"items"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
}

// RadiusAuthLogItem 是认证日志视图。
type RadiusAuthLogItem struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	NasAddr   string    `json:"nasAddr"`
	NasPaddr  string    `json:"nasPaddr"`
	MacAddr   string    `json:"macAddr"`
	AuthType  string    `json:"authType"`
	Result    string    `json:"result"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

// RadiusAuthLogListResult 是认证日志分页结果。
type RadiusAuthLogListResult struct {
	Items []RadiusAuthLogItem `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// RadiusBypassItem 是免认证规则视图。
type RadiusBypassItem struct {
	ID          uint64     `json:"id"`
	Type        string     `json:"type"`
	Value       string     `json:"value"`
	ProfileID   uint64     `json:"profileId"`
	ProfileName string     `json:"profileName"`
	NasID       *uint64    `json:"nasId"`
	ExpireTime  *time.Time `json:"expireTime"`
	Status      string     `json:"status"`
	Remark      string     `json:"remark"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// RadiusBypassListResult 是免认证规则分页结果。
type RadiusBypassListResult struct {
	Items []RadiusBypassItem `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

// RadiusKickResult 是踢下线/CoA 操作结果。
type RadiusKickResult struct {
	Success        bool   `json:"success"`
	ResponseCode   string `json:"responseCode"`
	Target         string `json:"target"`
	Message        string `json:"message"`
	ErrorCause     int    `json:"errorCause,omitempty"`
	ErrorCauseText string `json:"errorCauseText,omitempty"`
	RttMs          int64  `json:"rttMs,omitempty"`
}

// RadiusCertItem 是 TLS 证书视图（KeyPEM 永不下发，hasKey 指示是否持有私钥）。
type RadiusCertItem struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	CertType    string    `json:"certType"`
	CertPEM     string    `json:"certPem"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	Serial      string    `json:"serial"`
	Fingerprint string    `json:"fingerprint"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	HasKey      bool      `json:"hasKey"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// RadiusCertListResult 是证书分页结果。
type RadiusCertListResult struct {
	Items []RadiusCertItem `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// RadiusSystemSettings 是 RADIUS 系统（监听器）设置的生效值视图。
type RadiusSystemSettings struct {
	Enabled        bool   `json:"enabled"`
	BindHost       string `json:"bindHost"`
	AuthPort       int    `json:"authPort"`
	AcctPort       int    `json:"acctPort"`
	RadsecEnabled  bool   `json:"radsecEnabled"`
	RadsecPort     int    `json:"radsecPort"`
	RadsecCertID   uint64 `json:"radsecCertId"`
	RadsecCACertID uint64 `json:"radsecCaCertId"`
}

// RadiusListenerSettings 是系统设置中展示的 RADIUS 基础监听配置。
type RadiusListenerSettings struct {
	Enabled  bool   `json:"enabled"`
	BindHost string `json:"bindHost"`
	AuthPort int    `json:"authPort"`
	AcctPort int    `json:"acctPort"`
}

// RadiusServerSettings 是 RADIUS 服务器策略设置的生效值视图。
type RadiusServerSettings struct {
	MessageAuthMode          string `json:"messageAuthMode"`
	IgnorePassword           bool   `json:"ignorePassword"`
	SessionTimeout           int    `json:"sessionTimeout"`
	AcctInterimInterval      int    `json:"acctInterimInterval"`
	HistoryDays              int    `json:"historyDays"`
	RejectDelayMaxRejects    int    `json:"rejectDelayMaxRejects"`
	RejectDelayWindowSeconds int    `json:"rejectDelayWindowSeconds"`
}

// RadiusEapSettings 是 RADIUS EAP（802.1X）设置的生效值视图。
type RadiusEapSettings struct {
	Enabled         bool   `json:"enabled"`
	Method          string `json:"method"`
	EnabledHandlers string `json:"enabledHandlers"`
	TLSServerCertID uint64 `json:"tlsServerCertId"`
	TLSClientCAID   uint64 `json:"tlsClientCaId"`
	TLSMinVersion   string `json:"tlsMinVersion"`
}

// RadiusSettingsResponse 是 RADIUS 三段设置的生效值视图
// （env 配置与管理端 DB 配置合并后的结果）。
type RadiusSettingsResponse struct {
	System RadiusSystemSettings `json:"system"`
	Server RadiusServerSettings `json:"server"`
	Eap    RadiusEapSettings    `json:"eap"`
}
