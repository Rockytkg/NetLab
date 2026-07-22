package portal

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	portalwire "netlab-backend/internal/portal"
	"netlab-backend/internal/portal/protocol"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

const secretMask = "__UNCHANGED__"

const (
	terminationQueueSize = 256
	terminationTimeout   = 5 * time.Second
)

// Service coordinates Portal NAS administration, outbound requests, and
// cache-backed notification handling. UDP workers never call repositories.
type Service struct {
	nas             *repository.PortalNasRepository
	sessions        *repository.PortalSessionRepository
	cipher          *crypto.AESCipher
	client          *portalwire.ProtocolClient
	sessionsCache   *SessionStore
	logger          *zap.Logger
	manager         *portalwire.Manager
	cfgSvc          *sysconfig.Service
	envCfg          config.PortalConfig
	cacheMu         sync.RWMutex
	lifecycleMu     sync.Mutex
	nasBySource     map[string]*model.PortalNas
	terminationJobs chan terminationJob
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type terminationJob struct {
	nasID            uint64
	clientIP, reason string
}

// NewService creates a Portal service with explicitly injected runtime dependencies.
func NewService(nas *repository.PortalNasRepository, sessions *repository.PortalSessionRepository, cipher *crypto.AESCipher, client *portalwire.ProtocolClient, sessionCache *SessionStore, cfgSvc *sysconfig.Service, envCfg config.PortalConfig, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{nas: nas, sessions: sessions, cipher: cipher, client: client, sessionsCache: sessionCache, cfgSvc: cfgSvc, envCfg: envCfg, logger: logger, nasBySource: make(map[string]*model.PortalNas), terminationJobs: make(chan terminationJob, terminationQueueSize)}
}

// SetManager attaches the runtime manager after service construction.
func (s *Service) SetManager(manager *portalwire.Manager) { s.manager = manager }

// EffectiveConfig merges environment defaults with the persisted Portal override.
func (s *Service) EffectiveConfig(ctx context.Context) config.PortalConfig {
	cfg := s.envCfg
	if s.cfgSvc != nil {
		if stored, ok, err := s.cfgSvc.PortalSystem(ctx); err == nil && ok {
			cfg.Enabled, cfg.BindHost, cfg.NotifyPort = stored.Enabled, stored.BindHost, stored.NotifyPort
		}
	}
	return cfg
}

// UpdateSettings persists and immediately applies Portal listener settings.
func (s *Service) UpdateSettings(ctx context.Context, enabled bool, bindHost string, notifyPort int) *apperrors.AppError {
	if bindHost == "" || notifyPort < 1 || notifyPort > 65535 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "Portal绑定地址或端口非法")
	}
	if s.cfgSvc == nil || s.manager == nil {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "Portal运行时未启用")
	}
	if err := s.cfgSvc.SetPortalSystem(ctx, sysconfig.PortalSystemSettings{Enabled: enabled, BindHost: bindHost, NotifyPort: notifyPort}); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "保存Portal设置失败", err)
	}
	if err := s.manager.Apply(s.EffectiveConfig(ctx)); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "应用Portal设置失败", err)
	}
	return nil
}

// Start loads NAS runtime metadata before UDP listeners start, then persists
// notification-driven termination events through one bounded background worker.
func (s *Service) Start(ctx context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.ctx != nil {
		return errors.New("portal: 运行时已启动")
	}
	rows, err := s.nas.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("加载Portal NAS运行时缓存: %w", err)
	}
	s.cacheMu.Lock()
	for i := range rows {
		s.cacheNasLocked(&rows[i])
	}
	s.cacheMu.Unlock()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg.Add(1)
	go s.terminationWorker()
	return nil
}

// Shutdown stops the persistence worker and waits until ctx expires.
func (s *Service) Shutdown(ctx context.Context) error {
	s.lifecycleMu.Lock()
	cancel := s.cancel
	s.lifecycleMu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Service) terminationWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.terminationJobs:
			ctx, cancel := context.WithTimeout(context.Background(), terminationTimeout)
			if err := s.sessions.TerminateByNasAndClientIP(ctx, job.nasID, job.clientIP, job.reason); err != nil {
				s.logger.Error("Portal会话终止持久化失败", zap.Uint64("nas_id", job.nasID), zap.String("user_ip", job.clientIP), zap.Error(err))
			}
			cancel()
		}
	}
}

type NasInput struct {
	Name, Identifier, Vendor, ProtocolProfile, SourceIP, SharedSecret string
	ACPort                                                            int
	RadiusNasID                                                       *uint64
	CoAEnabled                                                        bool
	Status, Remark                                                    string
}

// ListNas returns Portal NAS devices for management screens.
func (s *Service) ListNas(ctx context.Context, page, size int, keyword string) ([]model.PortalNas, int64, *apperrors.AppError) {
	rows, total, err := s.nas.List(ctx, page, size, keyword)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list portal nas devices", err)
	}
	return rows, total, nil
}

// CreateNas encrypts and persists a NAS, then publishes it to the UDP cache.
func (s *Service) CreateNas(ctx context.Context, in NasInput) (*model.PortalNas, *apperrors.AppError) {
	if in.SharedSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "shared secret is required")
	}
	secret, err := s.cipher.Encrypt(in.SharedSecret)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt portal shared secret", err)
	}
	nas := &model.PortalNas{Name: in.Name, Identifier: in.Identifier, Vendor: in.Vendor, ProtocolProfile: in.ProtocolProfile, SourceIP: in.SourceIP, ACPort: acPortOrDefault(in.ACPort), SharedSecret: secret, RadiusNasID: in.RadiusNasID, CoAEnabled: in.CoAEnabled, Status: statusOrDefault(in.Status), Remark: in.Remark}
	if err = s.nas.Create(ctx, nas); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create portal nas device", err)
	}
	s.cacheNas(nas)
	return nas, nil
}

// UpdateNas persists a NAS change and atomically refreshes its UDP cache entry.
func (s *Service) UpdateNas(ctx context.Context, id uint64, in NasInput) (*model.PortalNas, *apperrors.AppError) {
	nas, err := s.nas.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load portal nas device", err)
	}
	if nas == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "portal nas device not found")
	}
	previousSourceIP := nas.SourceIP
	nas.Name, nas.Identifier, nas.Vendor, nas.ProtocolProfile, nas.SourceIP, nas.ACPort = in.Name, in.Identifier, in.Vendor, in.ProtocolProfile, in.SourceIP, acPortOrDefault(in.ACPort)
	nas.RadiusNasID, nas.CoAEnabled, nas.Status, nas.Remark = in.RadiusNasID, in.CoAEnabled, statusOrDefault(in.Status), in.Remark
	if in.SharedSecret != "" && in.SharedSecret != secretMask {
		encrypted, e := s.cipher.Encrypt(in.SharedSecret)
		if e != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt portal shared secret", e)
		}
		nas.SharedSecret = encrypted
	}
	if err = s.nas.Update(ctx, nas); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update portal nas device", err)
	}
	if previousSourceIP != nas.SourceIP {
		s.removeCachedNas(previousSourceIP)
	}
	s.cacheNas(nas)
	return nas, nil
}

// DeleteNas removes a NAS from durable storage and the UDP cache.
func (s *Service) DeleteNas(ctx context.Context, id uint64) *apperrors.AppError {
	nas, err := s.nas.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load portal nas device", err)
	}
	if nas == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "portal nas device not found")
	}
	if err = s.nas.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete portal nas device", err)
	}
	s.removeCachedNas(nas.SourceIP)
	return nil
}

// ListSessions returns durable Portal session projections.
func (s *Service) ListSessions(ctx context.Context, page, size int, username, nasID string) ([]model.PortalSession, int64, *apperrors.AppError) {
	rows, total, err := s.sessions.List(ctx, page, size, username, nasID)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list portal sessions", err)
	}
	return rows, total, nil
}

// TerminateSession requests NAS logout before marking the durable session ended.
func (s *Service) TerminateSession(ctx context.Context, id string) *apperrors.AppError {
	session, err := s.sessions.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load portal session", err)
	}
	if session == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "portal session not found")
	}
	nas, err := s.nas.GetByID(ctx, session.PortalNasID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load portal nas device", err)
	}
	if nas == nil {
		return apperrors.New(apperrors.ErrCodeInternal, "portal session has no nas device")
	}
	if err = s.logout(ctx, nas, session); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "portal device logout failed", err)
	}
	if err = s.sessions.Terminate(ctx, id, "admin_logout"); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to terminate portal session", err)
	}
	if err := s.sessionsCache.Delete(ctx, session.ClientIP); err != nil {
		s.logger.Warn("删除Portal Redis会话失败", zap.String("user_ip", session.ClientIP), zap.Error(err))
	}
	return nil
}

type AuthenticateInput struct {
	NASIdentifier, Username, Password, ClientIP string
	AuthType                                    byte
}

// Authenticate completes the Portal challenge/authentication exchange with a NAS.
func (s *Service) Authenticate(ctx context.Context, in AuthenticateInput) (*model.PortalSession, *apperrors.AppError) {
	nas, err := s.nas.GetByIdentifier(ctx, in.NASIdentifier)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load portal nas device", err)
	}
	if nas == nil || nas.Status != model.PortalNasStatusEnabled {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "portal nas device is not available")
	}
	ip, err := netip.ParseAddr(in.ClientIP)
	if err != nil || !ip.Is4() {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid wlan user ip")
	}
	if in.AuthType != protocol.AuthCHAP && in.AuthType != protocol.AuthPAP {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "unsupported portal authentication type")
	}
	secret, profile, appErr := s.credentials(nas)
	if appErr != nil {
		return nil, appErr
	}
	s.logger.Info("Portal认证请求", zap.String("nas_ip", nas.SourceIP), zap.String("user_ip", ip.String()), zap.String("profile", string(profile)), zap.String("username", defaultUsername(in.Username)))
	serial, err := randomSerial()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to generate portal serial number", err)
	}
	requestID := uint16(0)
	attrs := []protocol.Attribute{{Type: protocol.AttrUsername, Value: []byte(defaultUsername(in.Username))}}
	if in.AuthType == protocol.AuthCHAP {
		challenge, e := s.client.Exchange(ctx, profile, nas.SourceIP, nas.ACPort, secret, newPacket(profile, protocol.TypeRequestChallenge, 0, serial, 0, ip, 0))
		if e != nil {
			s.logger.Warn("Portal Challenge请求失败", zap.String("nas_ip", nas.SourceIP), zap.String("user_ip", ip.String()), zap.Error(e))
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "portal challenge failed", e)
		}
		challengeValue := challenge.Attribute(protocol.AttrChallenge)
		if challenge.Type != protocol.TypeAckChallenge || challenge.ErrorCode != 0 || len(challengeValue) != 16 {
			return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "portal challenge was rejected")
		}
		requestID = challenge.RequestID
		attrs = append(attrs, protocol.Attribute{Type: protocol.AttrCHAPPassword, Value: chapResponse(requestID, in.Password, challengeValue)})
	} else {
		attrs = append(attrs, protocol.Attribute{Type: protocol.AttrPassword, Value: []byte(in.Password)})
	}
	response, err := s.client.Exchange(ctx, profile, nas.SourceIP, nas.ACPort, secret, newPacket(profile, protocol.TypeRequestAuth, in.AuthType, serial, requestID, ip, 0, attrs...))
	if err != nil {
		s.logger.Warn("Portal认证失败", zap.String("nas_ip", nas.SourceIP), zap.String("user_ip", ip.String()), zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "portal authentication failed", err)
	}
	if response.Type != protocol.TypeAckAuth || response.ErrorCode != 0 {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "portal authentication was rejected")
	}
	if err := s.client.Send(ctx, profile, nas.SourceIP, nas.ACPort, secret, newPacket(profile, protocol.TypeAffAckAuth, in.AuthType, serial, requestID, ip, 0)); err != nil {
		s.logger.Warn("Portal认证确认发送失败", zap.String("nas_ip", nas.SourceIP), zap.String("user_ip", ip.String()), zap.Error(err))
	}
	now := time.Now()
	session := &model.PortalSession{ID: uuid.NewString(), PortalNasID: nas.ID, ProtocolSessionID: fmt.Sprintf("%d-%d-%s", nas.ID, serial, ip.String()), Username: defaultUsername(in.Username), ClientIP: ip.String(), State: model.PortalSessionActive, AuthenticatedAt: now, LastSeenAt: now}
	if err = s.sessions.Create(ctx, session); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to persist portal session", err)
	}
	if err := s.sessionsCache.Set(ctx, session); err != nil {
		s.logger.Warn("写入Portal Redis会话失败", zap.String("user_ip", session.ClientIP), zap.Error(err))
	}
	s.logger.Info("Portal认证成功", zap.Uint64("nas_id", nas.ID), zap.String("user_ip", session.ClientIP), zap.String("username", session.Username))
	return session, nil
}

// HandleNotification validates an NAS logout notification without repository I/O.
func (s *Service) HandleNotification(ctx context.Context, sourceIP string, raw []byte) error {
	if !s.running() {
		return errors.New("portal: 运行时未启动或已停止")
	}
	nas := s.cachedNas(sourceIP)
	if nas == nil || nas.Status != model.PortalNasStatusEnabled {
		s.logger.Warn("Portal下线通知来源未授权", zap.String("nas_ip", sourceIP))
		return errors.New("portal: unknown or disabled nas")
	}
	secret, profile, appErr := s.credentials(nas)
	if appErr != nil {
		return fmt.Errorf("读取Portal NAS凭据: %s", appErr.Message)
	}
	packet, err := s.client.DecodeAndVerify(profile, raw, secret)
	if err != nil {
		s.logger.Warn("Portal下线通知校验失败", zap.String("nas_ip", sourceIP), zap.Error(err))
		return fmt.Errorf("校验Portal下线通知: %w", err)
	}
	if packet.Type != protocol.TypeNotifyLogout || packet.ErrorCode != 0 {
		return errors.New("portal: unsupported notification")
	}
	if err := s.sessionsCache.Delete(ctx, packet.UserIP.String()); err != nil {
		s.logger.Warn("删除Portal Redis会话失败", zap.String("user_ip", packet.UserIP.String()), zap.Error(err))
	}
	select {
	case s.terminationJobs <- terminationJob{nasID: nas.ID, clientIP: packet.UserIP.String(), reason: "nas_notification"}:
		s.logger.Info("Portal下线通知", zap.Uint64("nas_id", nas.ID), zap.String("user_ip", packet.UserIP.String()))
		return nil
	default:
		return errors.New("portal: 会话终止队列已满")
	}
}
func (s *Service) logout(ctx context.Context, nas *model.PortalNas, session *model.PortalSession) error {
	secret, profile, appErr := s.credentials(nas)
	if appErr != nil {
		return fmt.Errorf("读取Portal NAS凭据: %s", appErr.Message)
	}
	ip, err := netip.ParseAddr(session.ClientIP)
	if err != nil || !ip.Is4() {
		return errors.New("portal: 会话用户IP非法")
	}
	serial, err := randomSerial()
	if err != nil {
		return fmt.Errorf("生成Portal下线流水号: %w", err)
	}
	response, err := s.client.Exchange(ctx, profile, nas.SourceIP, nas.ACPort, secret, newPacket(profile, protocol.TypeRequestLogout, 0, serial, 0, ip, 0))
	if err != nil {
		s.logger.Warn("Portal下线请求失败", zap.Uint64("nas_id", nas.ID), zap.String("user_ip", session.ClientIP), zap.Error(err))
		return fmt.Errorf("发送Portal下线请求: %w", err)
	}
	if response.Type != protocol.TypeAckLogout || response.ErrorCode != 0 {
		return errors.New("portal: NAS拒绝下线请求")
	}
	s.logger.Info("Portal下线成功", zap.Uint64("nas_id", nas.ID), zap.String("user_ip", session.ClientIP))
	return nil
}
func (s *Service) credentials(nas *model.PortalNas) (string, protocol.Profile, *apperrors.AppError) {
	if s.client == nil {
		return "", "", apperrors.New(apperrors.ErrCodeOperationDenied, "portal runtime is not enabled")
	}
	secret, err := s.cipher.Decrypt(nas.SharedSecret)
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.ErrCodeInternal, "failed to decrypt portal shared secret", err)
	}
	profile, err := profileForNas(nas.ProtocolProfile)
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.ErrCodeInvalidRequest, "unsupported portal protocol profile", err)
	}
	return secret, profile, nil
}
func profileForNas(profile string) (protocol.Profile, error) {
	switch profile {
	case "mobile-v2", string(protocol.ProfileCMCCV2):
		return protocol.ProfileCMCCV2, nil
	case string(protocol.ProfileCMCCV1):
		return protocol.ProfileCMCCV1, nil
	case string(protocol.ProfileHuaweiV1):
		return protocol.ProfileHuaweiV1, nil
	case string(protocol.ProfileHuaweiV2):
		return protocol.ProfileHuaweiV2, nil
	default:
		return "", fmt.Errorf("%s", profile)
	}
}
func newPacket(profile protocol.Profile, typ, authType byte, serial, requestID uint16, ip netip.Addr, errorCode byte, attrs ...protocol.Attribute) *protocol.Packet {
	version := protocol.VersionV1
	if profile == protocol.ProfileHuaweiV2 {
		version = protocol.VersionV2
	}
	return &protocol.Packet{Version: version, Type: typ, AuthType: authType, SerialNo: serial, RequestID: requestID, UserIP: ip, ErrorCode: errorCode, Attributes: attrs}
}
func (s *Service) cacheNas(nas *model.PortalNas) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cacheNasLocked(nas)
}
func (s *Service) cacheNasLocked(nas *model.PortalNas) {
	copy := *nas
	s.nasBySource[copy.SourceIP] = &copy
}
func (s *Service) removeCachedNas(sourceIP string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.nasBySource, sourceIP)
}
func (s *Service) cachedNas(sourceIP string) *model.PortalNas {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	nas := s.nasBySource[sourceIP]
	if nas == nil {
		return nil
	}
	copy := *nas
	return &copy
}

func (s *Service) running() bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return s.ctx != nil && s.ctx.Err() == nil
}

// chapResponse builds the MD5 input in the documented order: request ID,
// password, then the 16-byte NAS challenge. A dedicated buffer avoids append
// aliasing when the caller-owned challenge attribute has spare capacity.
func chapResponse(requestID uint16, password string, challenge []byte) []byte {
	input := make([]byte, 1+len(password)+len(challenge))
	input[0] = byte(requestID)
	copy(input[1:], password)
	copy(input[1+len(password):], challenge)
	sum := md5.Sum(input)
	return sum[:]
}
func randomSerial() (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}
func defaultUsername(username string) string {
	if username == "" {
		return "***"
	}
	return username
}
func statusOrDefault(status string) string {
	if status == "" {
		return model.PortalNasStatusEnabled
	}
	return status
}

func acPortOrDefault(port int) int {
	if port > 0 {
		return port
	}
	return portalwire.NASPort
}
