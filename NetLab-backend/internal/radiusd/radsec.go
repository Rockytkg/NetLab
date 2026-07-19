package radiusd

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"layeh.com/radius"

	"netlab-backend/config"
)

// defaultRadsecWorkers 是 RadSec 处理协程数的默认上限（配置 <=0 时生效）。
const defaultRadsecWorkers = 100

// maxRadiusPacketLength 是 RFC 2865 §3 约定的最大报文长度（4096），
// 超出即视为畸形帧，用于约束单报文分配。
const maxRadiusPacketLength = 4096

// errMalformedFrame 表示长度前缀无法界定帧：报文体未被消费，流已失步，
// 调用方必须关闭连接而不是继续解析。
var errMalformedFrame = errors.New("radius: malformed packet framing")

// radsecResponseWriter 把响应包写回 TLS 连接。
type radsecResponseWriter struct {
	conn net.Conn
}

func (w *radsecResponseWriter) Write(packet *radius.Packet) error {
	encoded, err := packet.Encode()
	if err != nil {
		return err
	}
	_, err = w.conn.Write(encoded)
	return err
}

// RadsecPacketServer 是 RADIUS over TLS（RFC 6614 RadSec）服务器。
type RadsecPacketServer struct {
	Addr string

	// SecretSource 按对端地址解析共享密钥。
	SecretSource radius.SecretSource
	// Handler 处理解析后的请求。
	Handler radius.Handler
	// Worker 是并发处理协程数上限。
	Worker int

	logger *zap.Logger

	shutdownRequested int32

	mu          sync.Mutex
	ctx         context.Context
	ctxDone     context.CancelFunc
	listeners   map[net.Conn]uint
	lastActive  chan struct{}
	activeCount int32
	workerPool  chan struct{}
}

func (s *RadsecPacketServer) initLocked() {
	if s.ctx == nil {
		workers := s.Worker
		if workers <= 0 {
			workers = defaultRadsecWorkers
		}
		s.ctx, s.ctxDone = context.WithCancel(context.Background())
		s.listeners = make(map[net.Conn]uint)
		s.lastActive = make(chan struct{})
		s.workerPool = make(chan struct{}, workers)
	}
}

// acquireWorkerSlot 在派生处理协程前占用工作池槽位；服务关闭中返回 false。
// 在读循环中同步占槽可真正约束在途协程数：池满时读循环停止拉取，
// 形成自然的 TCP 背压。
func (s *RadsecPacketServer) acquireWorkerSlot() bool {
	if s.ctx.Err() != nil {
		return false
	}
	select {
	case s.workerPool <- struct{}{}:
		return s.keepSlotUnlessShutdown()
	default:
	}
	select {
	case s.workerPool <- struct{}{}:
		return s.keepSlotUnlessShutdown()
	case <-s.ctx.Done():
		return false
	}
}

func (s *RadsecPacketServer) keepSlotUnlessShutdown() bool {
	if s.ctx.Err() != nil {
		s.releaseWorkerSlot()
		return false
	}
	return true
}

func (s *RadsecPacketServer) releaseWorkerSlot() {
	<-s.workerPool
}

func (s *RadsecPacketServer) activeAdd() {
	atomic.AddInt32(&s.activeCount, 1)
}

func (s *RadsecPacketServer) activeDone() {
	if atomic.AddInt32(&s.activeCount, -1) == -1 {
		close(s.lastActive)
	}
}

// tcpPacketBufferPool 复用单报文载荷缓冲，降低高频分配。
var tcpPacketBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 512)
		return &b
	},
}

// parseTcpPacket 从 TLS 流中解析单个 RADIUS 报文。
func parseTcpPacket(r io.Reader, secret []byte) (*radius.Packet, error) {
	var header struct {
		Code       uint8
		Identifier uint8
		Length     uint16
	}

	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return nil, err
	}

	headerSize := uint16(unsafe.Sizeof(header))
	if header.Length < headerSize || header.Length > maxRadiusPacketLength {
		return nil, fmt.Errorf("%w: length %d", errMalformedFrame, header.Length)
	}
	dataLen := int(header.Length - headerSize)
	if dataLen < 16 {
		return nil, fmt.Errorf("%w: payload shorter than authenticator", errMalformedFrame)
	}

	bufp := tcpPacketBufferPool.Get().(*[]byte)
	defer tcpPacketBufferPool.Put(bufp)
	if cap(*bufp) < dataLen {
		*bufp = make([]byte, dataLen)
	}
	data := (*bufp)[:dataLen]

	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	attrs, err := radius.ParseAttributes(data[16:])
	if err != nil {
		return nil, err
	}

	packet := &radius.Packet{
		Code:       radius.Code(header.Code),
		Identifier: header.Identifier,
		Secret:     secret,
		Attributes: attrs,
	}
	copy(packet.Authenticator[:], data[0:16])
	return packet, nil
}

// Serve 处理单条 TLS 连接上的连续报文流。
func (s *RadsecPacketServer) Serve(conn net.Conn) error {
	defer func() { _ = conn.Close() }() //nolint:errcheck

	if s.Handler == nil {
		return errors.New("radius: nil handler")
	}
	if s.SecretSource == nil {
		return errors.New("radius: nil secret source")
	}

	s.mu.Lock()
	s.initLocked()
	if atomic.LoadInt32(&s.shutdownRequested) == 1 {
		s.mu.Unlock()
		return radius.ErrServerShutdown
	}
	s.listeners[conn]++
	s.mu.Unlock()

	type requestKey struct {
		IP         string
		Identifier byte
	}
	var (
		requestsLock sync.Mutex
		requests     = map[requestKey]struct{}{}
	)

	s.activeAdd()
	defer func() {
		s.mu.Lock()
		s.listeners[conn]--
		if s.listeners[conn] == 0 {
			delete(s.listeners, conn)
		}
		s.mu.Unlock()
		s.activeDone()
	}()

	secret, err := s.SecretSource.RADIUSSecret(s.ctx, conn.RemoteAddr())
	if err != nil {
		s.logger.Error("radsec 获取共享密钥失败", zap.Error(err))
		return err
	}
	if len(secret) == 0 {
		return errors.New("radius: empty secret from secret source")
	}

	r := bufio.NewReader(conn)
	for {
		pkt, err := parseTcpPacket(r, secret)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return err
			}
			var netErr net.Error
			if errors.As(err, &netErr) {
				return err
			}
			if errors.Is(err, errMalformedFrame) {
				s.logger.Error("radsec 畸形帧，关闭连接",
					zap.String("remote", conn.RemoteAddr().String()),
					zap.Error(err),
				)
				return err
			}
			s.logger.Error("radsec 报文解析失败", zap.Error(err))
			continue
		}

		if !s.acquireWorkerSlot() {
			return radius.ErrServerShutdown
		}

		s.activeAdd()
		go func(packet *radius.Packet, c net.Conn) {
			defer s.activeDone()
			defer s.releaseWorkerSlot()

			key := requestKey{IP: c.RemoteAddr().String(), Identifier: packet.Identifier}
			requestsLock.Lock()
			if _, ok := requests[key]; ok {
				requestsLock.Unlock()
				return
			}
			requests[key] = struct{}{}
			requestsLock.Unlock()
			defer func() {
				requestsLock.Lock()
				delete(requests, key)
				requestsLock.Unlock()
			}()

			writer := &radsecResponseWriter{conn: c}
			request := radius.Request{
				LocalAddr:  c.LocalAddr(),
				RemoteAddr: c.RemoteAddr(),
				Packet:     packet,
			}
			s.Handler.ServeRADIUS(writer, &request)
		}(pkt, conn)
	}
}

// initTLSConfig 加载服务端证书与可选的客户端 CA（双向 TLS）。
// certID/caID 引用 nb_radius_certs 中的托管证书（优先）；为 0 时回退到文件路径。
func (s *RadsecPacketServer) initTLSConfig(svc *RadiusService, cfg config.RadiusConfig) (*tls.Config, error) {
	certPEM, keyPEM, err := svc.loadCertMaterial(cfg.RadsecCertID, cfg.RadsecCertFile, cfg.RadsecKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load radsec server certificate: %w", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		Time:         time.Now,
		Rand:         rand.Reader,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS12,
	}

	caPEM, err := svc.loadCAMaterial(cfg.RadsecCACertID, cfg.RadsecCAFile)
	if err != nil {
		return nil, fmt.Errorf("load radsec CA: %w", err)
	}
	if len(caPEM) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caPEM)
		tlsConfig.ClientCAs = pool
	}
	return tlsConfig, nil
}

// ListenAndServe 在 TLS 监听上接受连接，直到监听被关闭。
// 证书材料按当前配置解析（DB 托管证书优先，文件路径回退）。
func (s *RadsecPacketServer) ListenAndServe(svc *RadiusService, cfg config.RadiusConfig) error {
	tlsConfig, err := s.initTLSConfig(svc, cfg)
	if err != nil {
		return err
	}
	if s.Handler == nil {
		return errors.New("radius: nil handler")
	}
	if s.SecretSource == nil {
		return errors.New("radius: nil secret source")
	}

	addr := s.Addr
	if addr == "" {
		addr = ":2083"
	}

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.initLocked()
	tlsListener := listener
	s.mu.Unlock()
	defer func() { _ = tlsListener.Close() }() //nolint:errcheck

	s.logger.Info("radius RadSec 服务已启动", zap.String("addr", addr))
	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			// 监听关闭或临时错误：关闭流程中直接退出。
			if atomic.LoadInt32(&s.shutdownRequested) == 1 {
				return radius.ErrServerShutdown
			}
			continue
		}
		go func() { _ = s.Serve(conn) }()
	}
}

// Shutdown 优雅停止：关闭所有连接并等待在途处理完成。
func (s *RadsecPacketServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.initLocked()
	if atomic.CompareAndSwapInt32(&s.shutdownRequested, 0, 1) {
		for conn := range s.listeners {
			_ = conn.Close() //nolint:errcheck
		}
		s.ctxDone()
		s.activeDone()
	}
	s.mu.Unlock()

	select {
	case <-s.lastActive:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
