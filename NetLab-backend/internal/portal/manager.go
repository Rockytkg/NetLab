package portal

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"netlab-backend/config"
)

// Manager hot-applies Portal notification listener configuration.
type Manager struct {
	logger  *zap.Logger
	handler func(context.Context, string, []byte) error
	mu      sync.Mutex
	server  *NotificationServer
	cfg     config.PortalConfig
}

// NewManager creates a Portal runtime manager with its notification callback.
func NewManager(logger *zap.Logger, handler func(context.Context, string, []byte) error) *Manager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Manager{logger: logger, handler: handler}
}

// Apply starts, stops, or rebuilds the UDP listener when configuration changes.
func (m *Manager) Apply(cfg config.PortalConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !cfg.Enabled {
		if m.server != nil {
			if err := m.server.Shutdown(); err != nil {
				return fmt.Errorf("停止Portal通知监听器: %w", err)
			}
			m.server = nil
			m.logger.Info("Portal通知监听器已停止")
		}
		m.cfg = cfg
		return nil
	}
	if m.server != nil && m.cfg.BindHost == cfg.BindHost && m.cfg.NotifyPort == cfg.NotifyPort {
		m.cfg = cfg
		return nil
	}
	if m.server != nil {
		if err := m.server.Shutdown(); err != nil {
			return fmt.Errorf("重建前停止Portal监听器: %w", err)
		}
		m.server = nil
	}
	server, err := StartNotificationServer(cfg.BindHost, cfg.NotifyPort, m.logger, m.handler)
	if err != nil {
		return fmt.Errorf("启动Portal通知监听器: %w", err)
	}
	m.server, m.cfg = server, cfg
	m.logger.Info("Portal通知监听器已启动", zap.String("addr", fmt.Sprintf("%s:%d", cfg.BindHost, cfg.NotifyPort)))
	return nil
}

// Shutdown stops the currently active Portal listener.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server == nil {
		return nil
	}
	err := m.server.Shutdown()
	m.server = nil
	return err
}
