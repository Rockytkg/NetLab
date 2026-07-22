package v1

import (
	"net/netip"
	"testing"

	"netlab-backend/internal/portal/protocol"
)

func TestCodecRoundTripBasicAttributes(t *testing.T) {
	codec := NewCodec()
	packet := &protocol.Packet{Version: Version, Type: protocol.TypeRequestAuth, AuthType: protocol.AuthCHAP, SerialNo: 9, RequestID: 1, UserIP: netip.MustParseAddr("192.0.2.1"), Attributes: []protocol.Attribute{{Type: protocol.AttrUsername, Value: []byte("user")}, {Type: protocol.AttrCHAPPassword, Value: make([]byte, 16)}}}
	raw, err := codec.Encode(packet, protocol.AuthContext{})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.RequestID != 1 || len(decoded.Attributes) != 2 {
		t.Fatalf("decoded=%#v", decoded)
	}
}
func TestCodecRejectsV2Extensions(t *testing.T) {
	codec := NewCodec()
	packet := &protocol.Packet{Version: Version, Type: protocol.TypeAckAuth, UserIP: netip.MustParseAddr("192.0.2.1"), Attributes: []protocol.Attribute{{Type: protocol.AttrTextInfo, Value: []byte("rejected")}}}
	if _, err := codec.Encode(packet, protocol.AuthContext{}); err == nil {
		t.Fatal("expected extension rejection")
	}
}
func TestCodecAcceptsHistoricalCHAP17OnDecode(t *testing.T) {
	codec := NewCodec()
	raw := make([]byte, HeaderLength+19)
	raw[0] = Version
	raw[1] = protocol.TypeRequestAuth
	raw[2] = protocol.AuthCHAP
	raw[8] = 192
	raw[10] = 2
	raw[11] = 1
	raw[15] = 1
	raw[16] = protocol.AttrCHAPPassword
	raw[17] = 19
	if _, err := codec.Decode(raw); err != nil {
		t.Fatal(err)
	}
}
