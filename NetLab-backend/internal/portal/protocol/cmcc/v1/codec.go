package v1

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"netlab-backend/internal/portal/protocol"
)

// Codec implements the CMCC V1 basic-attribute compatibility profile.
// It accepts the historical 17-byte CHAP attribute on decode, while outgoing
// packets use the 16-byte MD5 value specified by the supplied CMCC document.
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
	if binary.BigEndian.Uint16(raw[12:14]) != 0 {
		return nil, protocol.ErrInvalidUserPort
	}
	p := &protocol.Packet{Version: Version, Type: raw[1], AuthType: raw[2], SerialNo: binary.BigEndian.Uint16(raw[4:6]), RequestID: binary.BigEndian.Uint16(raw[6:8]), UserIP: netip.AddrFrom4([4]byte(raw[8:12])), ErrorCode: raw[14]}
	if err := decodeAttributes(raw[HeaderLength:], int(raw[15]), p); err != nil {
		return nil, err
	}
	if err := Validate(p); err != nil {
		return nil, err
	}
	return p, nil
}
func (Codec) Encode(packet *protocol.Packet, _ protocol.AuthContext) ([]byte, error) {
	if err := Validate(packet); err != nil {
		return nil, err
	}
	attrs, err := encodeAttributes(packet.Attributes)
	if err != nil {
		return nil, err
	}
	raw := make([]byte, HeaderLength+len(attrs))
	raw[0], raw[1], raw[2] = Version, packet.Type, packet.AuthType
	binary.BigEndian.PutUint16(raw[4:6], packet.SerialNo)
	binary.BigEndian.PutUint16(raw[6:8], packet.RequestID)
	ip := packet.UserIP.As4()
	copy(raw[8:12], ip[:])
	raw[14] = packet.ErrorCode
	raw[15] = byte(len(packet.Attributes))
	copy(raw[HeaderLength:], attrs)
	return raw, nil
}
func (Codec) Verify(packet *protocol.Packet, _ protocol.AuthContext) error { return Validate(packet) }

func Validate(packet *protocol.Packet) error {
	if packet == nil {
		return protocol.ErrPacketTooShort
	}
	if packet.Version != Version {
		return protocol.ErrUnsupportedVersion
	}
	if !packet.UserIP.Is4() || packet.UserPort != 0 {
		return protocol.ErrInvalidUserPort
	}
	if packet.Type < protocol.TypeRequestChallenge || packet.Type > protocol.TypeNotifyLogout {
		return protocol.ErrUnsupportedVersion
	}
	if len(packet.Attributes) > 255 {
		return protocol.ErrAttributeCount
	}
	for _, attr := range packet.Attributes {
		if err := validateAttribute(attr); err != nil {
			return err
		}
	}
	return nil
}
func validateAttribute(attr protocol.Attribute) error {
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
		if err := validateAttribute(attr); err != nil {
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
func encodeAttributes(attributes []protocol.Attribute) ([]byte, error) {
	raw := make([]byte, 0)
	for _, attr := range attributes {
		if err := validateAttribute(attr); err != nil {
			return nil, err
		}
		if attr.Type == protocol.AttrCHAPPassword && len(attr.Value) == 17 {
			return nil, protocol.ErrInvalidAttribute
		}
		raw = append(raw, attr.Type, byte(len(attr.Value)+2))
		raw = append(raw, attr.Value...)
	}
	if HeaderLength+len(raw) > protocol.MaxPacketSize {
		return nil, protocol.ErrPacketTooLarge
	}
	return raw, nil
}
