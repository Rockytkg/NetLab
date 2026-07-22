package v1

import (
	"net/netip"
	"netlab-backend/internal/portal/protocol"
	"testing"
)

func TestCodecRoundTripWithPrivateAttributes(t *testing.T) {
	codec := NewCodec()
	packet := &protocol.Packet{Version: Version, Type: protocol.TypeRequestLogout, UserIP: netip.MustParseAddr("192.0.2.10"), Attributes: []protocol.Attribute{{Type: protocol.AttrMAC, Value: []byte{0, 1, 2, 3, 4, 5}}, {Type: protocol.AttrBASIP, Value: []byte{10, 0, 0, 1}}}}
	raw, err := codec.Encode(packet, protocol.AuthContext{})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Attributes) != 2 {
		t.Fatalf("attributes=%d", len(decoded.Attributes))
	}
}
