package model

import "time"

const (
	PortalNasStatusEnabled  = "enabled"
	PortalNasStatusDisabled = "disabled"
	PortalSessionActive     = "active"
	PortalSessionTerminated = "terminated"
)

// PortalNas is a Portal-capable access device. It is intentionally separate
// from RadiusNas because the two protocols have different listener and logout
// semantics. RadiusNasID is only an optional, explicit CoA association.
type PortalNas struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"type:varchar(64);not null" json:"name"`
	Identifier      string    `gorm:"type:varchar(128);not null;default:'';index" json:"identifier"`
	Vendor          string    `gorm:"type:varchar(32);not null;index:idx_portal_nas_source" json:"vendor"`
	ProtocolProfile string    `gorm:"type:varchar(32);not null;index" json:"protocolProfile"`
	SourceIP        string    `gorm:"type:varchar(64);not null;index:idx_portal_nas_source" json:"sourceIp"`
	ACPort          int       `gorm:"not null;default:2000" json:"acPort"`
	SharedSecret    string    `gorm:"type:varchar(255);not null;default:''" json:"-"`
	RadiusNasID     *uint64   `gorm:"index" json:"radiusNasId"`
	CoAEnabled      bool      `gorm:"not null;default:false" json:"coaEnabled"`
	Status          string    `gorm:"type:varchar(16);not null;default:'enabled';index" json:"status"`
	Remark          string    `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`
}

func (PortalNas) TableName() string { return "nb_portal_nas" }

// PortalSession is Portal's durable session projection. Redis is the runtime
// authority; this table supports management, recovery, and audit.
type PortalSession struct {
	ID                string     `gorm:"primaryKey;type:uuid" json:"id"`
	PortalNasID       uint64     `gorm:"not null;index" json:"portalNasId"`
	RadiusUserID      *uint64    `gorm:"index" json:"radiusUserId"`
	ProtocolSessionID string     `gorm:"type:varchar(128);not null;uniqueIndex" json:"protocolSessionId"`
	Username          string     `gorm:"type:varchar(64);not null;default:'';index" json:"username"`
	MacAddr           string     `gorm:"type:varchar(32);not null;default:'';index" json:"macAddr"`
	ClientIP          string     `gorm:"type:varchar(64);not null;default:''" json:"clientIp"`
	State             string     `gorm:"type:varchar(16);not null;default:'active';index" json:"state"`
	AuthenticatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now();index" json:"authenticatedAt"`
	LastSeenAt        time.Time  `gorm:"type:timestamptz;not null;default:now();index" json:"lastSeenAt"`
	TerminatedAt      *time.Time `gorm:"type:timestamptz" json:"terminatedAt"`
	TerminateReason   string     `gorm:"type:varchar(64);not null;default:''" json:"terminateReason"`
}

func (PortalSession) TableName() string { return "nb_portal_sessions" }
