package radiusd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"

	"netlab-backend/config"
)

// Server 管理 RADIUS 全部监听器（UDP 认证/记账 + 可选 RadSec）的生命周期。
type Server struct {
	service     *RadiusService
	authService *AuthService
	acctService *AcctService

	authServer   *radius.PacketServer
	acctServer   *radius.PacketServer
	radsecServer *RadsecPacketServer

	mu   sync.Mutex
	errs []error
}

// NewServer 构造 RADIUS 服务器（构建认证/记账服务并注册插件）。
func NewServer(service *RadiusService) *Server {
	registerPlugins(service)
	return &Server{
		service:     service,
		authService: NewAuthService(service),
		acctService: NewAcctService(service),
	}
}

// AuthService 返回认证服务（CoA 等管理操作需要）。
func (s *Server) AuthService() *AuthService { return s.authService }

// Start 启动全部启用的监听器（非阻塞，各自在独立 goroutine 中运行）。
func (s *Server) Start() {
	cfg := s.service.cfg()
	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.AuthPort)
	s.authServer = &radius.PacketServer{
		Addr:               addr,
		Handler:            s.authService,
		SecretSource:       s.service,
		InsecureSkipVerify: true, // 验签在 pipeline 各 stage 内完成
	}
	s.serve("auth", addr, func() error { return s.authServer.ListenAndServe() })

	acctAddr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.AcctPort)
	s.acctServer = &radius.PacketServer{
		Addr:               acctAddr,
		Handler:            s.acctService,
		SecretSource:       s.service,
		InsecureSkipVerify: true,
	}
	s.serve("acct", acctAddr, func() error { return s.acctServer.ListenAndServe() })

	if cfg.RadsecEnabled {
		radsecAddr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.RadsecPort)
		s.radsecServer = &RadsecPacketServer{
			Addr:         radsecAddr,
			SecretSource: s.service,
			Handler:      s.authService,
			logger:       s.service.logger,
		}
		s.serve("radsec", radsecAddr, func() error {
			return s.radsecServer.ListenAndServe(s.service, cfg)
		})
	}

	s.service.StartCleanupJobs()
}

// UpdateConfig 热更新运行时配置（不影响监听地址/端口；后者需重建 Server）。
func (s *Server) UpdateConfig(cfg config.RadiusConfig) {
	s.service.UpdateConfig(cfg)
}

// serve 在独立 goroutine 中运行一个监听器并记录致命错误。
func (s *Server) serve(name, addr string, listen func() error) {
	s.service.wg.Add(1)
	go func() {
		defer s.service.wg.Done()
		s.service.logger.Info("radius 监听器启动",
			zap.String("listener", name),
			zap.String("addr", addr),
		)
		if err := listen(); err != nil && !errors.Is(err, radius.ErrServerShutdown) && !errors.Is(err, net.ErrClosed) {
			s.mu.Lock()
			s.errs = append(s.errs, fmt.Errorf("radius %s listener: %w", name, err))
			s.mu.Unlock()
			s.service.logger.Error("radius 监听器异常退出",
				zap.String("listener", name),
				zap.Error(err),
			)
		}
	}()
}

// Shutdown 优雅停止全部监听器与后台任务。
func (s *Server) Shutdown(ctx context.Context) {
	if s.authServer != nil {
		_ = s.authServer.Shutdown(ctx)
	}
	if s.acctServer != nil {
		_ = s.acctServer.Shutdown(ctx)
	}
	if s.radsecServer != nil {
		_ = s.radsecServer.Shutdown(ctx)
	}
	s.service.Shutdown()
}

// Errors 返回运行期间监听器产生的异常（用于健康诊断）。
func (s *Server) Errors() []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]error(nil), s.errs...)
}

// ShutdownTimeout 是 RADIUS 服务优雅停止的建议超时。
const ShutdownTimeout = 5 * time.Second
