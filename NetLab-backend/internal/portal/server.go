package portal

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"go.uber.org/zap"

	"netlab-backend/internal/portal/protocol"
)

// DatagramHandler processes one UDP datagram. It must not access PostgreSQL;
// protocol handlers should use injected cache-backed services on this path.
type DatagramHandler func(context.Context, []byte, netip.AddrPort) error

// ServerConfig configures the bounded UDP processing pool.
type ServerConfig struct {
	BindHost  string
	Port      int
	Workers   int
	QueueSize int
}

type udpBuffer struct {
	data []byte
	n    int
	peer netip.AddrPort
}

// Server is a bounded Portal UDP server. The reader and workers share pooled
// packet buffers, so packet load cannot create unbounded goroutines or buffers.
type Server struct {
	mu      sync.Mutex
	cfg     ServerConfig
	logger  *zap.Logger
	handler DatagramHandler
	conn    *net.UDPConn
	ctx     context.Context
	cancel  context.CancelFunc
	jobs    chan *udpBuffer
	pool    sync.Pool
	wg      sync.WaitGroup
}

// NewServer builds a UDP server but does not bind the listening socket.
func NewServer(cfg ServerConfig, logger *zap.Logger, handler DatagramHandler) *Server {
	if cfg.Workers <= 0 {
		cfg.Workers = defaultUDPWorkers
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = cfg.Workers * queueWorkersFactor
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Server{cfg: cfg, logger: logger, handler: handler}
	s.pool.New = func() any { return &udpBuffer{data: make([]byte, protocol.MaxPacketSize)} }
	return s
}

// Start binds the listener and starts the fixed reader/worker pool once.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handler == nil {
		return errors.New("portal: udp handler is required")
	}
	if s.ctx != nil {
		return errors.New("portal: udp server already started")
	}
	addr := &net.UDPAddr{IP: net.ParseIP(s.cfg.BindHost), Port: s.cfg.Port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("监听Portal UDP地址: %w", err)
	}
	s.conn = conn
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.jobs = make(chan *udpBuffer, s.cfg.QueueSize)
	for i := 0; i < s.cfg.Workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	s.wg.Add(1)
	go s.readLoop()
	s.logger.Info("Portal UDP监听器启动", zap.String("addr", conn.LocalAddr().String()), zap.Int("workers", s.cfg.Workers))
	return nil
}

// Addr returns the bound UDP address after Start succeeds.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	return s.conn.LocalAddr()
}
func (s *Server) readLoop() {
	defer s.wg.Done()
	defer close(s.jobs)
	for {
		buffer := s.pool.Get().(*udpBuffer)
		n, peer, err := s.conn.ReadFromUDP(buffer.data)
		if err != nil {
			s.pool.Put(buffer)
			if errors.Is(err, net.ErrClosed) || s.ctx.Err() != nil {
				return
			}
			s.logger.Warn("Portal UDP读取失败", zap.Error(err))
			continue
		}
		ip, ok := netip.AddrFromSlice(peer.IP)
		if !ok {
			s.pool.Put(buffer)
			s.logger.Warn("Portal UDP来源地址非法")
			continue
		}
		buffer.n = n
		buffer.peer = netip.AddrPortFrom(ip.Unmap(), uint16(peer.Port))
		select {
		case s.jobs <- buffer:
		case <-s.ctx.Done():
			s.pool.Put(buffer)
			return
		}
	}
}
func (s *Server) worker() {
	defer s.wg.Done()
	for buffer := range s.jobs {
		if err := s.handler(s.ctx, buffer.data[:buffer.n], buffer.peer); err != nil && s.ctx.Err() == nil {
			s.logger.Warn("Portal报文处理失败", zap.String("peer", buffer.peer.String()), zap.Error(err))
		}
		buffer.n = 0
		s.pool.Put(buffer)
	}
}

// Shutdown stops the reader, waits for fixed workers, and honours ctx.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.ctx == nil {
		s.mu.Unlock()
		return nil
	}
	cancel, conn := s.cancel, s.conn
	s.mu.Unlock()
	cancel()
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		s.logger.Warn("关闭Portal UDP监听器失败", zap.Error(err))
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
