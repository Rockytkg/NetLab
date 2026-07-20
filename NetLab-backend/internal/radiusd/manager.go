package radiusd

import (
	"context"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"netlab-backend/config"
	"netlab-backend/internal/radiusd/repository"
	"netlab-backend/pkg/crypto"
)

// Manager 持有 RADIUS 服务器的生命周期，并负责把管理端的配置变更应用到
// 运行时：监听地址/端口/RadSec 证书等"监听级"变更通过进程内重建 Server
// 生效（UDP 重绑在毫秒级完成）；其余配置（EAP 方法、证书引用、安全策略等）
// 通过对运行中 Server 的原子配置替换热生效。
type Manager struct {
	logger *zap.Logger
	cipher *crypto.AESCipher

	userRepo       repository.UserRepository
	nasRepo        repository.NasRepository
	sessionRepo    repository.SessionRepository
	accountingRepo repository.AccountingRepository
	authLog        AuthLogger
	bypassRepo     repository.BypassRepository
	certRepo       repository.CertRepository

	mu      sync.Mutex
	server  *Server
	coa     *CoAService
	cfg     config.RadiusConfig
	running bool
}

// NewManager 构造 RADIUS 运行时管理器；certRepo 可为 nil（仅文件证书可用），
// bypassRepo 可为 nil（免认证规则不生效）。
func NewManager(
	logger *zap.Logger,
	cipher *crypto.AESCipher,
	userRepo repository.UserRepository,
	nasRepo repository.NasRepository,
	sessionRepo repository.SessionRepository,
	accountingRepo repository.AccountingRepository,
	authLog AuthLogger,
	bypassRepo repository.BypassRepository,
	certRepo repository.CertRepository,
) *Manager {
	return &Manager{
		logger:         logger,
		cipher:         cipher,
		userRepo:       userRepo,
		nasRepo:        nasRepo,
		sessionRepo:    sessionRepo,
		accountingRepo: accountingRepo,
		authLog:        authLog,
		bypassRepo:     bypassRepo,
		certRepo:       certRepo,
	}
}

// Apply 应用一份新的生效配置。调用方负责把 env 与 DB 配置合并成本函数
// 入参；本函数决定热替换还是重启监听器。
func (m *Manager) Apply(cfg config.RadiusConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case m.running && !cfg.Enabled:
		m.logger.Info("radius 服务停止（配置已禁用）")
		m.shutdownLocked()
	case m.running && cfg.Enabled && listenerConfigChanged(m.cfg, cfg):
		m.logger.Info("radius 监听级配置变更，重建监听器")
		m.shutdownLocked()
		m.startLocked(cfg)
	case m.running:
		// 非监听级变更：原子替换配置，后续报文立即按新配置处理。
		m.server.UpdateConfig(cfg)
		m.logger.Info("radius 运行时配置已热更新")
	case cfg.Enabled:
		m.startLocked(cfg)
	}
	m.cfg = cfg
}

// CoA 返回当前运行中的 CoA 服务；服务未运行时返回 nil。
func (m *Manager) CoA() *CoAService {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.coa
}

// Running 报告 RADIUS 监听器是否在运行。
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// CurrentConfig 返回最近一次应用的配置。
func (m *Manager) CurrentConfig() config.RadiusConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// InvalidateBypassRules publishes terminal-access changes to the running server.
func (m *Manager) InvalidateBypassRules() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server != nil {
		m.server.service.InvalidateBypassRules()
	}
}

// Shutdown 停止全部监听器与后台任务（进程退出时调用）。
func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server != nil {
		m.server.Shutdown(ctx)
		m.server = nil
		m.coa = nil
	}
	m.running = false
}

// startLocked 构建并启动新的 Server（调用方须持锁且保证当前无运行实例）。
func (m *Manager) startLocked(cfg config.RadiusConfig) {
	core := NewRadiusService(
		cfg, m.logger, m.cipher,
		m.userRepo, m.nasRepo, m.sessionRepo, m.accountingRepo, m.authLog, m.bypassRepo, m.certRepo,
	)
	m.server = NewServer(core)
	m.coa = NewCoAService(core)
	m.server.Start()
	m.running = true
	m.logger.Info("radius 服务已启动",
		zap.String("authAddr", addrOf(cfg.BindHost, cfg.AuthPort)),
		zap.String("acctAddr", addrOf(cfg.BindHost, cfg.AcctPort)),
		zap.Bool("eap", cfg.EAPEnabled),
		zap.Bool("radsec", cfg.RadsecEnabled),
	)
}

// shutdownLocked 停止当前 Server（调用方须持锁）。
func (m *Manager) shutdownLocked() {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	if m.server != nil {
		m.server.Shutdown(ctx)
		m.server = nil
		m.coa = nil
	}
	m.running = false
}

// listenerConfigChanged 判定两份配置在"监听级"字段上是否有差异：
// 有差异时必须重建监听器（UDP 重绑 / RadSec TLS 重建），无差异可热替换。
func listenerConfigChanged(oldCfg, newCfg config.RadiusConfig) bool {
	return oldCfg.BindHost != newCfg.BindHost ||
		oldCfg.AuthPort != newCfg.AuthPort ||
		oldCfg.AcctPort != newCfg.AcctPort ||
		oldCfg.RadsecEnabled != newCfg.RadsecEnabled ||
		oldCfg.RadsecPort != newCfg.RadsecPort ||
		oldCfg.RadsecCertID != newCfg.RadsecCertID ||
		oldCfg.RadsecCACertID != newCfg.RadsecCACertID ||
		oldCfg.RadsecCertFile != newCfg.RadsecCertFile ||
		oldCfg.RadsecKeyFile != newCfg.RadsecKeyFile ||
		oldCfg.RadsecCAFile != newCfg.RadsecCAFile
}

// addrOf 格式化 host:port。
func addrOf(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
