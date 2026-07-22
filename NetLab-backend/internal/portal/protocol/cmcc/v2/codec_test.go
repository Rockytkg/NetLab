package v2

import (
	"net/netip"
	"testing"

	"netlab-backend/internal/portal/protocol"
)

func TestCodecRoundTrip(t *testing.T) {
	codec := NewCodec()
	input := &protocol.Packet{Version: protocol.VersionV1, Type: protocol.TypeRequestAuth, AuthType: protocol.AuthPAP, SerialNo: 7, UserIP: netip.MustParseAddr("10.1.2.3"), Attributes: []protocol.Attribute{{Type: protocol.AttrUsername, Value: []byte("13800000000")}, {Type: protocol.AttrPassword, Value: []byte("secret")}}}
	raw, err := codec.Encode(input, protocol.AuthContext{})
	if err != nil {
		t.Fatal(err)
	}
	output, err := codec.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if output.SerialNo != input.SerialNo || output.UserIP != input.UserIP || string(output.Attribute(protocol.AttrUsername)) != "13800000000" {
		t.Fatalf("unexpected packet: %#v", output)
	}
}
func TestCodecRejectsReservedAndTLV(t *testing.T) {
	codec := NewCodec()
	raw := make([]byte, 16)
	raw[0] = protocol.VersionV1
	raw[3] = 1
	if _, err := codec.Decode(raw); err != protocol.ErrInvalidReserved {
		t.Fatalf("error=%v", err)
	}
	raw[3] = 0
	raw[15] = 1
	if _, err := codec.Decode(raw); err == nil {
		t.Fatal("expected attribute error")
	}
}
func BenchmarkCodecRoundTrip(b *testing.B) {
	codec := NewCodec()
	packet := &protocol.Packet{Version: protocol.VersionV1, Type: protocol.TypeRequestChallenge, UserIP: netip.MustParseAddr("10.1.2.3")}
	raw, _ := codec.Encode(packet, protocol.AuthContext{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := codec.Decode(raw); err != nil {
			b.Fatal(err)
		}
	}
}
