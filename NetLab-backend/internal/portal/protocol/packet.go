package protocol

import "net/netip"

const (
	VersionV1 byte = 0x01
	VersionV2 byte = 0x02

	TypeRequestChallenge byte = 0x01
	TypeAckChallenge     byte = 0x02
	TypeRequestAuth      byte = 0x03
	TypeAckAuth          byte = 0x04
	TypeRequestLogout    byte = 0x05
	TypeAckLogout        byte = 0x06
	TypeAffAckAuth       byte = 0x07
	TypeNotifyLogout     byte = 0x08
	TypeRequestInfo      byte = 0x09
	TypeAckInfo          byte = 0x0a

	AuthCHAP byte = 0x00
	AuthPAP  byte = 0x01

	AttrUsername      byte = 0x01
	AttrPassword      byte = 0x02
	AttrChallenge     byte = 0x03
	AttrCHAPPassword  byte = 0x04
	AttrTextInfo      byte = 0x05
	AttrUplinkFlux    byte = 0x06
	AttrDownlinkFlux  byte = 0x07
	AttrPort          byte = 0x08
	AttrBASIP         byte = 0x0a
	AttrMAC           byte = 0x0b
	AttrUserPrivateIP byte = 0x0d
	AttrUserIPv6      byte = 0xf1

	HeaderV1Length = 16
	HeaderV2Length = 32
	MaxPacketSize  = 1024
)

type Attribute struct {
	Type  byte
	Value []byte
}

// Packet represents the common Portal header. V2 codecs populate Authenticator.
type Packet struct {
	Version       byte
	Type          byte
	AuthType      byte
	SerialNo      uint16
	RequestID     uint16
	UserIP        netip.Addr
	UserPort      uint16
	ErrorCode     byte
	Attributes    []Attribute
	Authenticator [16]byte
}

func (p *Packet) Attribute(typ byte) []byte {
	for _, attr := range p.Attributes {
		if attr.Type == typ {
			return attr.Value
		}
	}
	return nil
}
