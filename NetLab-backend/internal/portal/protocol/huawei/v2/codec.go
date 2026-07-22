package v2

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"net/netip"

	"netlab-backend/internal/portal/protocol"
)

var ipv6Marker = [4]byte{0xff, 0xff, 0xff, 0xff}

// Codec encodes and verifies Huawei V2 packets. Request verification uses a
// zero authenticator; response verification uses the paired request value.
type Codec struct{}

func NewCodec() Codec { return Codec{} }

func (Codec) Decode(raw []byte) (*protocol.Packet, error) {
	if len(raw) < HeaderLength {
		return nil, protocol.ErrPacketTooShort
	}
	if len(raw) > protocol.MaxPacketSize {
		return nil, protocol.ErrPacketTooLarge
	}
	if raw[0] != Version {
		return nil, fmt.Errorf("%w: %d", protocol.ErrUnsupportedVersion, raw[0])
	}
	if raw[3] != 0 {
		return nil, protocol.ErrInvalidReserved
	}

	packet := &protocol.Packet{Version: Version, Type: raw[1], AuthType: raw[2], SerialNo: binary.BigEndian.Uint16(raw[4:6]), RequestID: binary.BigEndian.Uint16(raw[6:8]), UserPort: binary.BigEndian.Uint16(raw[12:14]), ErrorCode: raw[14]}
	copy(packet.Authenticator[:], raw[16:32])
	if err := decodeAttributes(raw[HeaderLength:], int(raw[15]), packet); err != nil {
		return nil, err
	}

	if [4]byte(raw[8:12]) == ipv6Marker {
		value := packet.Attribute(protocol.AttrUserIPv6)
		if len(value) != 16 {
			return nil, protocol.ErrInvalidAttribute
		}
		packet.UserIP = netip.AddrFrom16([16]byte(value))
	} else {
		packet.UserIP = netip.AddrFrom4([4]byte(raw[8:12]))
	}
	if err := Validate(packet); err != nil {
		return nil, err
	}
	return packet, nil
}

func (Codec) Encode(packet *protocol.Packet, auth protocol.AuthContext) ([]byte, error) {
	if packet == nil {
		return nil, protocol.ErrPacketTooShort
	}
	attrs, normalized, err := normaliseAttributes(packet)
	if err != nil {
		return nil, err
	}
	validated := *packet
	validated.Attributes = normalized
	if err := Validate(&validated); err != nil {
		return nil, err
	}

	raw := make([]byte, HeaderLength+len(attrs))
	raw[0], raw[1], raw[2] = Version, packet.Type, packet.AuthType
	binary.BigEndian.PutUint16(raw[4:6], packet.SerialNo)
	binary.BigEndian.PutUint16(raw[6:8], packet.RequestID)
	if packet.UserIP.Is6() {
		copy(raw[8:12], ipv6Marker[:])
	} else {
		ip := packet.UserIP.As4()
		copy(raw[8:12], ip[:])
	}
	binary.BigEndian.PutUint16(raw[12:14], packet.UserPort)
	raw[14], raw[15] = packet.ErrorCode, byte(len(normalized))
	copy(raw[HeaderLength:], attrs)

	var seed [16]byte
	if auth.RequestAuthenticator != nil {
		seed = *auth.RequestAuthenticator
	}
	copy(raw[16:32], seed[:])
	sum := md5.Sum(append(append([]byte(nil), raw...), auth.SharedSecret...))
	copy(raw[16:32], sum[:])
	return raw, nil
}

func (c Codec) Verify(packet *protocol.Packet, auth protocol.AuthContext) error {
	if packet == nil {
		return protocol.ErrPacketTooShort
	}
	raw, err := c.Encode(packet, auth)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(raw[16:32], packet.Authenticator[:]) != 1 {
		return fmt.Errorf("portal: 华为V2 Authenticator校验失败")
	}
	return nil
}

func Validate(packet *protocol.Packet) error {
	if packet == nil {
		return protocol.ErrPacketTooShort
	}
	if packet.Version != Version {
		return protocol.ErrUnsupportedVersion
	}
	if !packet.UserIP.IsValid() || packet.UserPort != 0 {
		return protocol.ErrInvalidUserPort
	}
	if len(packet.Attributes) > 255 {
		return protocol.ErrAttributeCount
	}
	hasIPv6 := false
	for _, attr := range packet.Attributes {
		if attr.Type == protocol.AttrUserIPv6 {
			hasIPv6 = true
		}
		if err := validateAttribute(packet.Type, attr); err != nil {
			return err
		}
	}
	if packet.UserIP.Is6() != hasIPv6 {
		return protocol.ErrInvalidAttribute
	}
	return nil
}

func validateAttribute(packetType byte, attr protocol.Attribute) error {
	switch attr.Type {
	case protocol.AttrUsername:
		if len(attr.Value) > 253 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrPassword:
		if len(attr.Value) > 16 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrChallenge:
		if len(attr.Value) != 16 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrCHAPPassword:
		if len(attr.Value) != 16 && len(attr.Value) != 17 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrTextInfo:
		if len(attr.Value) < 1 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrUplinkFlux, protocol.AttrDownlinkFlux:
		if packetType == protocol.TypeRequestInfo && len(attr.Value) != 0 {
			return protocol.ErrInvalidAttribute
		}
		if packetType == protocol.TypeAckInfo && len(attr.Value) != 8 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrPort:
		if len(attr.Value) > 35 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrBASIP:
		if len(attr.Value) != 4 && len(attr.Value) != 16 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrMAC:
		if len(attr.Value) != 6 && len(attr.Value) != 17 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrUserPrivateIP:
		if len(attr.Value) != 4 {
			return protocol.ErrInvalidAttribute
		}
	case protocol.AttrUserIPv6:
		if len(attr.Value) != 16 {
			return protocol.ErrInvalidAttribute
		}
	default:
		return protocol.ErrInvalidAttribute
	}
	return nil
}

func decodeAttributes(raw []byte, count int, packet *protocol.Packet) error {
	pos := 0
	for i := 0; i < count; i++ {
		if pos+2 > len(raw) {
			return protocol.ErrAttributeCount
		}
		length := int(raw[pos+1])
		if length < 2 || pos+length > len(raw) {
			return protocol.ErrInvalidAttribute
		}
		attr := protocol.Attribute{Type: raw[pos], Value: append([]byte(nil), raw[pos+2:pos+length]...)}
		if err := validateAttribute(packet.Type, attr); err != nil {
			return err
		}
		packet.Attributes = append(packet.Attributes, attr)
		pos += length
	}
	if pos != len(raw) {
		return protocol.ErrAttributeCount
	}
	return nil
}

func normaliseAttributes(packet *protocol.Packet) ([]byte, []protocol.Attribute, error) {
	attrs := append([]protocol.Attribute(nil), packet.Attributes...)
	if packet.UserIP.Is6() {
		value := packet.UserIP.As16()
		found := false
		for i := range attrs {
			if attrs[i].Type == protocol.AttrUserIPv6 {
				attrs[i].Value = value[:]
				found = true
			}
		}
		if !found {
			attrs = append(attrs, protocol.Attribute{Type: protocol.AttrUserIPv6, Value: value[:]})
		}
	}
	raw := make([]byte, 0)
	for _, attr := range attrs {
		if err := validateAttribute(packet.Type, attr); err != nil {
			return nil, nil, err
		}
		raw = append(raw, attr.Type, byte(len(attr.Value)+2))
		raw = append(raw, attr.Value...)
	}
	if HeaderLength+len(raw) > protocol.MaxPacketSize {
		return nil, nil, protocol.ErrPacketTooLarge
	}
	return raw, attrs, nil
}
