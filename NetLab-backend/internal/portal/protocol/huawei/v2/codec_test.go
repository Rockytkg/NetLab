package v2

import (
	"net/netip"
	"netlab-backend/internal/portal/protocol"
	"testing"
)

func TestCodecAuthenticators(t *testing.T) {
	codec := NewCodec()
	request := &protocol.Packet{Version: Version, Type: protocol.TypeRequestChallenge, UserIP: netip.MustParseAddr("192.0.2.10"), SerialNo: 9}
	requestRaw, err := codec.Encode(request, protocol.AuthContext{SharedSecret: []byte("shared")})
	if err != nil {
		t.Fatal(err)
	}
	decodedRequest, err := codec.Decode(requestRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err = codec.Verify(decodedRequest, protocol.AuthContext{SharedSecret: []byte("shared")}); err != nil {
		t.Fatal(err)
	}
	response := &protocol.Packet{Version: Version, Type: protocol.TypeAckChallenge, UserIP: netip.MustParseAddr("192.0.2.10"), SerialNo: 9, RequestID: 1, Attributes: []protocol.Attribute{{Type: protocol.AttrChallenge, Value: make([]byte, 16)}}}
	responseRaw, err := codec.Encode(response, protocol.AuthContext{SharedSecret: []byte("shared"), RequestAuthenticator: &decodedRequest.Authenticator})
	if err != nil {
		t.Fatal(err)
	}
	decodedResponse, err := codec.Decode(responseRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err = codec.Verify(decodedResponse, protocol.AuthContext{SharedSecret: []byte("shared"), RequestAuthenticator: &decodedRequest.Authenticator}); err != nil {
		t.Fatal(err)
	}
	if err = codec.Verify(decodedResponse, protocol.AuthContext{SharedSecret: []byte("shared")}); err == nil {
		t.Fatal("expected response authenticator failure")
	}
}
func TestCodecIPv6AutoAttribute(t *testing.T) {
	codec := NewCodec()
	packet := &protocol.Packet{Version: Version, Type: protocol.TypeRequestChallenge, UserIP: netip.MustParseAddr("2001:db8::1")}
	raw, err := codec.Encode(packet, protocol.AuthContext{SharedSecret: []byte("shared")})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.UserIP.String() != "2001:db8::1" {
		t.Fatalf("ip=%s", decoded.UserIP)
	}
}
