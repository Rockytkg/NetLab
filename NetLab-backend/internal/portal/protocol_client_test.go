package portal

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"netlab-backend/internal/portal/protocol"
	cmccv2 "netlab-backend/internal/portal/protocol/cmcc/v2"
)

func TestProtocolClientExchangeCMCCV2(t *testing.T) {
	registry := protocol.NewRegistry()
	if err := registry.Register(cmccv2.NewHandler(nil)); err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	codec := cmccv2.NewCodec()
	secret := "test-secret"
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, protocol.MaxPacketSize)
		n, peer, readErr := conn.ReadFromUDP(buf)
		if readErr != nil {
			return
		}
		request, decodeErr := codec.Decode(buf[:n])
		if decodeErr != nil {
			return
		}
		response := &protocol.Packet{Version: protocol.VersionV1, Type: protocol.TypeAckAuth, AuthType: request.AuthType, SerialNo: request.SerialNo, RequestID: request.RequestID, UserIP: request.UserIP}
		raw, encodeErr := codec.Encode(response, protocol.AuthContext{SharedSecret: []byte(secret)})
		if encodeErr == nil {
			_, _ = conn.WriteToUDP(raw, peer)
		}
	}()
	client := NewProtocolClient(registry, nil)
	client.timeout = time.Second
	client.port = conn.LocalAddr().(*net.UDPAddr).Port
	packet := &protocol.Packet{Version: protocol.VersionV1, Type: protocol.TypeRequestAuth, AuthType: protocol.AuthPAP, SerialNo: 7, UserIP: netip.MustParseAddr("192.0.2.11"), Attributes: []protocol.Attribute{{Type: protocol.AttrUsername, Value: []byte("alice")}, {Type: protocol.AttrPassword, Value: []byte("password")}}}
	response, err := client.Exchange(context.Background(), protocol.ProfileCMCCV2, "127.0.0.1", client.port, secret, packet)
	if err != nil {
		t.Fatal(err)
	}
	if response.Type != protocol.TypeAckAuth || response.SerialNo != packet.SerialNo {
		t.Fatalf("unexpected response: %#v", response)
	}
	<-done
}
