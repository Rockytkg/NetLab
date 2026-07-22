package request

type PortalNasUpsertRequest struct {
	Name            string  `json:"name" binding:"required,min=1,max=64"`
	Identifier      string  `json:"identifier" binding:"required,min=1,max=128"`
	Vendor          string  `json:"vendor" binding:"required,oneof=mobile huawei"`
	ProtocolProfile string  `json:"protocolProfile" binding:"required,oneof=mobile-v2 cmcc-v1 cmcc-v2 huawei-v1 huawei-v2"`
	SourceIP        string  `json:"sourceIp" binding:"required,ip"`
	ACPort          int     `json:"acPort" binding:"omitempty,min=1,max=65535"`
	SharedSecret    string  `json:"sharedSecret" binding:"omitempty,min=1,max=128"`
	RadiusNasID     *uint64 `json:"radiusNasId" binding:"omitempty,min=1"`
	CoAEnabled      bool    `json:"coaEnabled"`
	Status          string  `json:"status" binding:"omitempty,oneof=enabled disabled"`
	Remark          string  `json:"remark" binding:"omitempty,max=255"`
}

// PortalSystemSettingsRequest updates Portal listener runtime configuration.
type PortalSystemSettingsRequest struct {
	Enabled    bool   `json:"enabled"`
	BindHost   string `json:"bindHost" binding:"required"`
	NotifyPort int    `json:"notifyPort" binding:"required,min=1,max=65535"`
}

// BillingSettingsRequest updates the RADIUS and Portal listener settings together.
type BillingSettingsRequest struct {
	Radius RadiusListenerSettingsRequest `json:"radius" binding:"required"`
	Portal BillingPortalSettingsRequest  `json:"portal" binding:"required"`
}

// BillingPortalSettingsRequest shares its bind address with Radius in billing settings.
type BillingPortalSettingsRequest struct {
	Enabled    bool `json:"enabled"`
	NotifyPort int  `json:"notifyPort" binding:"required,min=1,max=65535"`
}

type PortalAuthenticationRequest struct {
	NASIdentifier string `json:"wlanacname" binding:"required,min=1,max=128"`
	ClientIP      string `json:"wlanuserip" binding:"required,ip"`
	Username      string `json:"username" binding:"omitempty,max=253"`
	Password      string `json:"password" binding:"omitempty,max=128"`
	AuthType      string `json:"authType" binding:"omitempty,oneof=chap pap"`
}
