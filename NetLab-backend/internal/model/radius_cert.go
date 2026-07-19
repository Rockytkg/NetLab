package model

import "time"

// RADIUS TLS 证书类型。
const (
	// RadiusCertTypeServer 是服务端证书（必须带私钥，用于 EAP-TLS/PEAP/TTLS 隧道与 RadSec）。
	RadiusCertTypeServer = "server"
	// RadiusCertTypeCA 是 CA 证书（信任锚，用于校验 EAP-TLS 客户端证书或 RadSec 对端）。
	RadiusCertTypeCA = "ca"
)

// RadiusCert 是一张由系统托管的 TLS 证书（对应 toughradius 的 sys_cert）。
// CertPEM 为 PEM 编码证书（可为 bundle）；KeyPEM 为 AES-256-GCM 密文存储的
// PEM 私钥，永不通过 API 序列化输出。Subject/Issuer/Serial/Fingerprint 与
// 有效期在导入/替换时由后端从证书解析填充。
type RadiusCert struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(128);not null;uniqueIndex" json:"name"`
	CertType    string    `gorm:"type:varchar(16);not null;index" json:"certType"`
	CertPEM     string    `gorm:"type:text;not null" json:"certPem"`
	KeyPEM      string    `gorm:"type:text;not null;default:''" json:"-"`
	Subject     string    `gorm:"type:varchar(512);not null;default:''" json:"subject"`
	Issuer      string    `gorm:"type:varchar(512);not null;default:''" json:"issuer"`
	Serial      string    `gorm:"type:varchar(128);not null;default:''" json:"serial"`
	Fingerprint string    `gorm:"type:varchar(128);not null;default:''" json:"fingerprint"`
	NotBefore   time.Time `gorm:"type:timestamptz;not null" json:"notBefore"`
	NotAfter    time.Time `gorm:"type:timestamptz;not null;index" json:"notAfter"`
	Remark      string    `gorm:"type:varchar(512);not null;default:''" json:"remark"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updatedAt"`

	// HasKey 指示是否持有私钥，读取时计算，不落库。
	HasKey bool `gorm:"-" json:"hasKey"`
}

// TableName 指定 RadiusCert 的数据库表名。
func (RadiusCert) TableName() string { return "nb_radius_certs" }
