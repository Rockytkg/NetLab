package portal

import (
	"context"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestServerDispatchesDatagram(t *testing.T) {
	seen := make(chan netip.AddrPort, 1)
	server := NewServer(ServerConfig{BindHost: "127.0.0.1", Port: 0, Workers: 2}, zap.NewNop(), func(_ context.Context, raw []byte, peer netip.AddrPort) error {
		if string(raw) != "portal" {
			t.Errorf("payload=%q", raw)
		}
		seen <- peer
		return nil
	})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown(context.Background())
	conn, err := net.DialUDP("udp", nil, server.Addr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.Write([]byte("portal")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-seen:
	case <-time.After(time.Second):
		t.Fatal("packet was not dispatched")
	}
}

func BenchmarkServerDispatch(b *testing.B) {
	var count atomic.Int64
	done := make(chan struct{})
	server := NewServer(ServerConfig{BindHost: "127.0.0.1", Port: 0, Workers: 8, QueueSize: 256}, zap.NewNop(), func(_ context.Context, _ []byte, _ netip.AddrPort) error {
		if count.Add(1) == int64(b.N) {
			close(done)
		}
		return nil
	})
	if err := server.Start(); err != nil {
		b.Fatal(err)
	}
	defer server.Shutdown(context.Background())
	conn, err := net.DialUDP("udp", nil, server.Addr().(*net.UDPAddr))
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()
	payload := []byte{1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := conn.Write(payload); err != nil {
			b.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		b.Fatal("dispatch timeout")
	}
}
