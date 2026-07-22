package portal

import (
	"context"
	"net/netip"

	"go.uber.org/zap"
)

// NotificationServer receives NAS-initiated NTF_LOGOUT packets. The callback
// performs cache-backed NAS lookup and authenticator verification before state change.
type NotificationServer struct {
	server *Server
}

// StartNotificationServer starts the fixed worker-pool listener for NTF_LOGOUT.
func StartNotificationServer(host string, port int, logger *zap.Logger, handler func(context.Context, string, []byte) error) (*NotificationServer, error) {
	server := NewServer(ServerConfig{BindHost: host, Port: port}, logger, func(ctx context.Context, raw []byte, peer netip.AddrPort) error {
		return handler(ctx, peer.Addr().String(), raw)
	})
	if err := server.Start(); err != nil {
		return nil, err
	}
	return &NotificationServer{server: server}, nil
}

// Shutdown stops the notification listener within the standard timeout.
func (s *NotificationServer) Shutdown() error {
	if s == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}
