package radiusd

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/cache"
	radiuserrors "netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/repository"
	"netlab-backend/pkg/crypto"
)

// unknownNasSecret 是未知 NAS 报文的占位密钥：让报文进入 handler 后再被拒绝，
// 而不是在库层被静默丢弃，以便记录拒绝日志。
const unknownNasSecret = "__unknown_nas__" //nolint:gosec // 占位符，非真实密钥

// ConfigStore 是 RADIUS 运行时配置的原子容器：管理端修改配置后无需停止
// 监听器即可让新配置对后续报文生效（热生效）；监听地址/端口类变更由
// Manager 通过重建 Server 应用。
type ConfigStore struct {
	v atomic.Value // config.RadiusConfig
}

// NewConfigStore 构造以 cfg 为初始值的配置容器。
func NewConfigStore(cfg config.RadiusConfig) *ConfigStore {
	s := &ConfigStore{}
	s.v.Store(cfg)
	return s
}

// Get 返回当前生效的配置快照。
func (s *ConfigStore) Get() config.RadiusConfig {
	if v := s.v.Load(); v != nil {
		return v.(config.RadiusConfig) //nolint:errcheck // 写入方只存该类型
	}
	return config.RadiusConfig{}
}

// Set 原子替换当前配置。
func (s *ConfigStore) Set(cfg config.RadiusConfig) {
	s.v.Store(cfg)
}

// AuthLogger 是认证日志写入接口（由管理层的 GORM 实现满足）。
type AuthLogger interface {
	Create(ctx context.Context, entry *model.RadiusAuthLog) error
	// CreateBatch 批量插入认证日志（后台 worker 聚合落库）。
	CreateBatch(ctx context.Context, entries []*model.RadiusAuthLog) error
	// PurgeBefore 删除早于 cutoff 的日志，返回删除条数。
	PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// 认证日志异步写入管线参数：有界队列 + 固定 worker 批量落库。
const (
	// authLogQueueSize 是认证日志待写队列容量（写满即丢弃新日志）。
	authLogQueueSize = 4096
	// authLogWorkerCount 是日志落库 worker 数。
	authLogWorkerCount = 2
	// authLogBatchSize 是单次批量落库的条数阈值。
	authLogBatchSize = 100
	// authLogFlushInterval 是批量落库的时间阈值（与条数阈值先到先触发）。
	authLogFlushInterval = time.Second
	// authLogDropWarnInterval 是队列满丢弃日志的 Warn 节流间隔。
	authLogDropWarnInterval = time.Minute
)

// dbCallTimeout 是认证/记账热路径单次 DB 调用的超时上限。
const dbCallTimeout = 5 * time.Second

// RadiusService 聚合 RADIUS 运行时所需的全部依赖：存储、缓存、配置与加密组件。
// 认证（AuthService）、记账（AcctService）与 CoA（CoAService）均内嵌它。
type RadiusService struct {
	cfgStore *ConfigStore
	logger   *zap.Logger
	cipher   *crypto.AESCipher

	UserRepo       repository.UserRepository
	NasRepo        repository.NasRepository
	SessionRepo    repository.SessionRepository
	AccountingRepo repository.AccountingRepository
	CertRepo       repository.CertRepository
	BypassRepo     repository.BypassRepository
	AuthLog        AuthLogger

	authRate *authRateLimiter

	nasCache        *cache.TTLCache[*model.RadiusNas] // 正缓存 1 分钟
	nasNegCache     *cache.TTLCache[struct{}]         // 负缓存 10 秒（未授权 NAS 防穿透）
	userCache       *cache.TTLCache[*model.RadiusUser]
	lastOnlineCache *cache.TTLCache[struct{}] // 最近已写 last_online 的用户名（写节流）

	bypassCacheMu sync.Mutex
	bypassCache   []model.RadiusBypass
	bypassCacheAt time.Time

	authLogCh       chan *model.RadiusAuthLog
	authLogDropped  atomic.Int64
	authLogLastWarn atomic.Int64 // 上次丢弃 Warn 的时间（UnixNano）

	stopCh  chan struct{}
	stopped sync.Once
	wg      sync.WaitGroup
}

// NewRadiusService 构造 RADIUS 运行时服务。certRepo 可为 nil（未启用证书管理时
// EAP/RadSec 仅支持文件证书）；bypassRepo 可为 nil（免认证规则不生效）。
func NewRadiusService(
	cfg config.RadiusConfig,
	logger *zap.Logger,
	cipher *crypto.AESCipher,
	userRepo repository.UserRepository,
	nasRepo repository.NasRepository,
	sessionRepo repository.SessionRepository,
	accountingRepo repository.AccountingRepository,
	authLog AuthLogger,
	bypassRepo repository.BypassRepository,
	certRepo ...repository.CertRepository,
) *RadiusService {
	var certs repository.CertRepository
	if len(certRepo) > 0 {
		certs = certRepo[0]
	}
	return &RadiusService{
		cfgStore:        NewConfigStore(cfg),
		logger:          logger,
		cipher:          cipher,
		UserRepo:        userRepo,
		NasRepo:         nasRepo,
		SessionRepo:     sessionRepo,
		AccountingRepo:  accountingRepo,
		CertRepo:        certs,
		BypassRepo:      bypassRepo,
		AuthLog:         authLog,
		authRate:        newAuthRateLimiter(defaultAuthRateShards),
		nasCache:        cache.NewTTLCache[*model.RadiusNas](time.Minute, 1024),
		nasNegCache:     cache.NewTTLCache[struct{}](10*time.Second, 4096),
		userCache:       cache.NewTTLCache[*model.RadiusUser](10*time.Second, 2048),
		lastOnlineCache: cache.NewTTLCache[struct{}](time.Minute, 2048),
		authLogCh:       make(chan *model.RadiusAuthLog, authLogQueueSize),
		stopCh:          make(chan struct{}),
	}
}

// cfg 返回当前生效的配置快照（管理端热更新后立即反映到后续报文）。
func (s *RadiusService) cfg() config.RadiusConfig {
	return s.cfgStore.Get()
}

// UpdateConfig 原子替换运行时配置；监听地址/端口不在此列（需重启监听器）。
func (s *RadiusService) UpdateConfig(cfg config.RadiusConfig) {
	s.cfgStore.Set(cfg)
}

// Config 返回 RADIUS 配置。
func (s *RadiusService) Config() config.RadiusConfig {
	return s.cfg()
}

// StartCleanupJobs 启动后台任务：认证日志批量落库 worker，以及周期清理任务
// （僵尸在线会话与历史数据保留期清理）。
func (s *RadiusService) StartCleanupJobs() {
	for i := 0; i < authLogWorkerCount; i++ {
		s.wg.Add(1)
		go s.authLogWorker()
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				// 僵尸会话阈值：3 倍记账间隔未更新即视为僵尸（每轮读取最新配置）。
				zombieThreshold := time.Duration(s.cfg().AcctInterimInterval) * 3 * time.Second
				s.runCleanup(zombieThreshold)
			}
		}
	}()
}

// runCleanup 执行一轮清理，单点失败不影响后续轮次。
func (s *RadiusService) runCleanup(zombieThreshold time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if removed, err := s.SessionRepo.DeleteZombie(ctx, zombieThreshold); err != nil {
		s.logger.Warn("radius 僵尸会话清理失败", zap.Error(err))
	} else if removed > 0 {
		s.logger.Info("radius 清理僵尸在线会话", zap.Int64("removed", removed))
	}

	if days := s.cfg().HistoryDays; days > 0 {
		cutoff := time.Now().AddDate(0, 0, -days)
		if removed, err := s.AccountingRepo.PurgeBefore(ctx, cutoff); err != nil {
			s.logger.Warn("radius 记账历史清理失败", zap.Error(err))
		} else if removed > 0 {
			s.logger.Info("radius 清理过期记账记录", zap.Int64("removed", removed))
		}
		if removed, err := s.AuthLog.PurgeBefore(ctx, cutoff); err != nil {
			s.logger.Warn("radius 认证日志清理失败", zap.Error(err))
		} else if removed > 0 {
			s.logger.Info("radius 清理过期认证日志", zap.Int64("removed", removed))
		}
	}
}

// Shutdown 停止后台任务并等待退出。
func (s *RadiusService) Shutdown() {
	s.stopped.Do(func() { close(s.stopCh) })
	s.wg.Wait()
}

// authLogWorker 消费认证日志队列并批量落库：满 authLogBatchSize 条或
// authLogFlushInterval 到期的先到先写；stopCh 关闭后排空队列做最后 flush
// 再退出（由 s.wg 跟踪，Shutdown 会等待）。
func (s *RadiusService) authLogWorker() {
	defer s.wg.Done()
	batch := make([]*model.RadiusAuthLog, 0, authLogBatchSize)
	ticker := time.NewTicker(authLogFlushInterval)
	defer ticker.Stop()

	flush := func() {
		s.flushAuthLogs(batch)
		batch = batch[:0]
	}
	for {
		select {
		case entry := <-s.authLogCh:
			batch = append(batch, entry)
			if len(batch) >= authLogBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.stopCh:
			// 排空剩余日志后退出；此后到达的日志由 RecordAuthLog 的丢弃路径处理。
			for {
				select {
				case entry := <-s.authLogCh:
					batch = append(batch, entry)
					if len(batch) >= authLogBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// flushAuthLogs 以 10 秒超时批量写入认证日志；失败仅告警（日志不阻断认证）。
func (s *RadiusService) flushAuthLogs(batch []*model.RadiusAuthLog) {
	if len(batch) == 0 || s.AuthLog == nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Error("radius 认证日志写入 panic", zap.Any("recover", rec))
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.AuthLog.CreateBatch(ctx, batch); err != nil {
		s.logger.Warn("radius 认证日志批量写入失败", zap.Int("batch", len(batch)), zap.Error(err))
	}
}

// RADIUSSecret 实现 layeh/radius 的 SecretSource 接口：按报文源 IP 查询 NAS
// 并返回其共享密钥（运行时解密）。未知 NAS 返回占位密钥，让报文进入 handler
// 后以"未授权 NAS"拒绝并记录日志。
func (s *RadiusService) RADIUSSecret(ctx context.Context, remoteAddr net.Addr) ([]byte, error) {
	ip := remoteAddr.String()
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	nas, err := s.GetNas(ip, "")
	if err != nil {
		return []byte(unknownNasSecret), nil
	}
	return []byte(nas.Secret), nil
}

// GetNas 按源 IP（优先）或 NAS-Identifier 查找 NAS：命中结果正缓存 1 分钟，
// 未命中结果负缓存 10 秒（防止伪造源地址的洪水每包打穿 PG）。
// 返回的 NAS 的 Secret 已解密为明文（仅存在于内存）。
func (s *RadiusService) GetNas(ip, identifier string) (*model.RadiusNas, error) {
	cacheKey := fmt.Sprintf("%s|%s", ip, identifier)
	if nas, ok := s.nasCache.Get(cacheKey); ok {
		return nas, nil
	}
	if _, ok := s.nasNegCache.Get(cacheKey); ok {
		return nil, radiuserrors.NewUnauthorizedNasError(ip, identifier, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCallTimeout)
	defer cancel()
	nas, err := s.NasRepo.GetByIPOrIdentifier(ctx, ip, identifier)
	if err != nil {
		return nil, err
	}
	if nas == nil {
		s.nasNegCache.Set(cacheKey, struct{}{})
		return nil, radiuserrors.NewUnauthorizedNasError(ip, identifier, nil)
	}

	// 解密共享密钥后再缓存与使用；解密失败按未授权处理（配置损坏）。
	plain, err := s.cipher.Decrypt(nas.Secret)
	if err != nil {
		s.logger.Error("radius NAS 密钥解密失败", zap.String("ip", ip), zap.Error(err))
		return nil, radiuserrors.NewUnauthorizedNasError(ip, identifier, err)
	}
	nas.Secret = plain

	s.nasCache.Set(cacheKey, nas)
	return nas, nil
}

// GetValidUser 按用户名（或 MAC 认证时的 MAC）加载用户并校验状态与有效期。
// 返回的用户的 Password 已解密为明文，Profile 已回填（供取值器继承套餐）。
// 结果缓存 10 秒。
func (s *RadiusService) GetValidUser(usernameOrMac string, macauth bool) (*model.RadiusUser, error) {
	cacheKey := fmt.Sprintf("%t|%s", macauth, usernameOrMac)
	if cached, ok := s.userCache.Get(cacheKey); ok {
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCallTimeout)
	defer cancel()
	var user *model.RadiusUser
	var err error
	if macauth {
		user, err = s.UserRepo.GetByMacAddr(ctx, usernameOrMac)
	} else {
		user, err = s.UserRepo.GetByUsername(ctx, usernameOrMac)
	}
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, radiuserrors.NewUserNotExistsError()
	}
	if user.Status == model.RadiusUserStatusDisabled {
		return nil, radiuserrors.NewUserDisabledError()
	}
	if user.ExpireTime.Before(time.Now()) {
		return nil, radiuserrors.NewUserExpiredError()
	}

	plain, err := s.cipher.Decrypt(user.Password)
	if err != nil {
		s.logger.Error("radius 用户密码解密失败", zap.String("username", user.Username), zap.Error(err))
		return nil, radiuserrors.NewUserNotExistsError()
	}
	user.Password = plain

	// 回填套餐供取值器动态继承；套餐缺失不阻断认证（仅用本行配置）。
	if user.ProfileID != nil {
		if profile, perr := s.UserRepo.GetProfileByID(ctx, *user.ProfileID); perr == nil && profile != nil {
			user.Profile = profile
		}
	}

	s.userCache.Set(cacheKey, user)
	return user, nil
}

// GetBypassRules 返回启用状态的免认证规则（10 秒缓存）。
// 存储不可用时记录日志并返回空列表：免认证只是放行捷径，缓存/DB 抖动
// 不应阻断正常认证流程。
func (s *RadiusService) GetBypassRules() []model.RadiusBypass {
	s.bypassCacheMu.Lock()
	defer s.bypassCacheMu.Unlock()
	if s.bypassCache != nil && time.Since(s.bypassCacheAt) < 10*time.Second {
		return s.bypassCache
	}
	if s.BypassRepo == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rules, err := s.BypassRepo.ListEnabled(ctx)
	if err != nil {
		s.logger.Warn("radius 免认证规则加载失败", zap.Error(err))
		return nil
	}
	s.bypassCache = rules
	s.bypassCacheAt = time.Now()
	return rules
}

// InvalidateBypassRules removes the local rule snapshot after an admin change.
// A disabled or deleted terminal must not remain eligible for the cache TTL.
func (s *RadiusService) InvalidateBypassRules() {
	s.bypassCacheMu.Lock()
	defer s.bypassCacheMu.Unlock()
	s.bypassCache = nil
	s.bypassCacheAt = time.Time{}
}

// UpdateUserMac 持久化用户最近看到的 MAC（绑定学习）。
func (s *RadiusService) UpdateUserMac(username, macaddr string) {
	ctx, cancel := context.WithTimeout(context.Background(), dbCallTimeout)
	defer cancel()
	_ = s.UserRepo.UpdateMacAddr(ctx, username, macaddr)
}

// UpdateUserLastOnline 记录用户最近上线时间；60 秒内重复认证仅写一次，
// 降低认证热路径的写放大（last_online 本身是粗粒度信息）。
func (s *RadiusService) UpdateUserLastOnline(username string) {
	if username == "" {
		return
	}
	if _, ok := s.lastOnlineCache.Get(username); ok {
		return
	}
	s.lastOnlineCache.Set(username, struct{}{})
	ctx, cancel := context.WithTimeout(context.Background(), dbCallTimeout)
	defer cancel()
	_ = s.UserRepo.UpdateLastOnline(ctx, username)
}

// CheckAuthRateLimit 按用户名限制认证频率（每秒 1 次，防爆破与重放）。
func (s *RadiusService) CheckAuthRateLimit(username string) error {
	return s.authRate.check(username, time.Second)
}

// ReleaseAuthRateLimit 解除用户名的认证频率限制状态。
func (s *RadiusService) ReleaseAuthRateLimit(username string) {
	s.authRate.release(username)
}

// RecordAuthLog 异步写入认证日志：非阻塞投递到有界队列，由后台 worker
// 聚合批量落库，绝不阻塞认证路径。队列满时丢弃并计数，Warn 每分钟最多一条。
func (s *RadiusService) RecordAuthLog(entry *model.RadiusAuthLog) {
	if s.AuthLog == nil || entry == nil {
		return
	}
	select {
	case s.authLogCh <- entry:
	default:
		dropped := s.authLogDropped.Add(1)
		if now := time.Now().UnixNano(); now-s.authLogLastWarn.Load() >= int64(authLogDropWarnInterval) {
			s.authLogLastWarn.Store(now)
			s.logger.Warn("radius 认证日志队列已满，丢弃日志", zap.Int64("dropped_total", dropped))
		}
	}
}

// CheckRequestSecret 校验记账报文的 keyed Request Authenticator（RFC 2866）：
// MD5(code+id+len + 16×0 + attrs + secret) 须等于报文的 Authenticator 字段。
func (s *RadiusService) CheckRequestSecret(r *radius.Packet, secret []byte) error {
	request, err := r.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal packet: %w", err)
	}
	if len(secret) == 0 {
		return errors.New("secret is empty")
	}
	hash := md5.New()
	hash.Write(request[:4])
	var nul [16]byte
	hash.Write(nul[:])
	hash.Write(request[20:])
	hash.Write(secret)
	var sum [md5.Size]byte
	if !bytes.Equal(hash.Sum(sum[:0]), request[4:20]) {
		return errors.New("secret mismatch")
	}
	return nil
}
