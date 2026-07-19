package radiusd

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/repository"
	"netlab-backend/pkg/crypto"
)

// 测试共享常量。
const (
	// testMasterKey 是 32 字符的 AES 主密钥（与生产 .env 同格式）。
	testMasterKey = "0123456789abcdef0123456789abcdef"
	// testNasSecret 是测试 NAS 的共享密钥明文；存储与报文均使用它。
	testNasSecret = "testing123"
	// testExchangeTimeout 是单次 UDP 交换的超时；拒绝路径服务端固定延迟 1 秒。
	testExchangeTimeout = 5 * time.Second
)

// 编译期接口断言：fake 必须满足运行时存储接口。
var (
	_ repository.UserRepository       = (*fakeUserRepo)(nil)
	_ repository.NasRepository        = (*fakeNasRepo)(nil)
	_ repository.SessionRepository    = (*fakeSessionRepo)(nil)
	_ repository.AccountingRepository = (*fakeAccountingRepo)(nil)
	_ AuthLogger                      = (*fakeAuthLogger)(nil)
)

// —— fakeUserRepo ——

// fakeUserRepo 是内存版 UserRepository。返回结构体副本，避免服务层
// （GetValidUser 解密回填）篡改库内记录。
type fakeUserRepo struct {
	mu          sync.Mutex
	users       map[string]*model.RadiusUser
	vlanUpdates map[string][2]int
	lastOnlines []string
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:       make(map[string]*model.RadiusUser),
		vlanUpdates: make(map[string][2]int),
	}
}

func (r *fakeUserRepo) add(u *model.RadiusUser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *u
	r.users[u.Username] = &cp
}

func (r *fakeUserRepo) GetByUsername(_ context.Context, username string) (*model.RadiusUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[username]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) GetByMacAddr(_ context.Context, mac string) (*model.RadiusUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.MacAddr == mac {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepo) GetProfileByID(_ context.Context, id uint64) (*model.RadiusProfile, error) {
	return nil, nil
}

func (r *fakeUserRepo) UpdateMacAddr(_ context.Context, username, mac string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[username]; ok {
		u.MacAddr = mac
	}
	return nil
}

func (r *fakeUserRepo) UpdateVlanID(_ context.Context, username string, vlanid1, vlanid2 int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[username]; ok {
		u.Vlanid1 = vlanid1
		u.Vlanid2 = vlanid2
	}
	r.vlanUpdates[username] = [2]int{vlanid1, vlanid2}
	return nil
}

func (r *fakeUserRepo) UpdateLastOnline(_ context.Context, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastOnlines = append(r.lastOnlines, username)
	return nil
}

// —— fakeNasRepo ——

// fakeNasRepo 是内存版 NasRepository，匹配语义对齐 GORM 实现：
// 仅返回启用状态的 NAS，源 IP 精确匹配优先于 identifier 匹配。
type fakeNasRepo struct {
	mu  sync.Mutex
	nas []*model.RadiusNas
}

func (r *fakeNasRepo) add(n *model.RadiusNas) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *n
	r.nas = append(r.nas, &cp)
}

func (r *fakeNasRepo) GetByIPOrIdentifier(_ context.Context, ip, identifier string) (*model.RadiusNas, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var byIdentifier *model.RadiusNas
	for _, n := range r.nas {
		if n.Status != model.RadiusNasStatusEnabled {
			continue
		}
		if ip != "" && n.Ipaddr == ip {
			cp := *n
			return &cp, nil
		}
		if identifier != "" && n.Identifier != "" && n.Identifier == identifier {
			byIdentifier = n
		}
	}
	if byIdentifier != nil {
		cp := *byIdentifier
		return &cp, nil
	}
	return nil, nil
}

// —— fakeSessionRepo ——

// fakeSessionRepo 是内存版 SessionRepository，语义对齐 GORM 实现：
// Create 幂等（重复会话返回 created=false），UpdateCounters 命中 0 行不报错。
type fakeSessionRepo struct {
	mu        sync.Mutex
	sessions  map[string]*model.RadiusOnline
	countErr  error // 非空时 CountByUsername 返回该错误（错误注入）
	createErr error
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: make(map[string]*model.RadiusOnline)}
}

func (r *fakeSessionRepo) Create(_ context.Context, session *model.RadiusOnline) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return false, r.createErr
	}
	if _, ok := r.sessions[session.AcctSessionId]; ok {
		return false, nil
	}
	cp := *session
	r.sessions[session.AcctSessionId] = &cp
	return true, nil
}

func (r *fakeSessionRepo) UpdateCounters(_ context.Context, session *model.RadiusOnline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.sessions[session.AcctSessionId]
	if !ok {
		return nil // GORM Updates 命中 0 行不返回错误
	}
	cur.AcctSessionTime = session.AcctSessionTime
	cur.AcctInputTotal = session.AcctInputTotal
	cur.AcctOutputTotal = session.AcctOutputTotal
	cur.AcctInputPackets = session.AcctInputPackets
	cur.AcctOutputPackets = session.AcctOutputPackets
	cur.LastUpdate = time.Now()
	return nil
}

func (r *fakeSessionRepo) Delete(_ context.Context, acctSessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, acctSessionID)
	return nil
}

func (r *fakeSessionRepo) GetBySessionID(_ context.Context, acctSessionID string) (*model.RadiusOnline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[acctSessionID]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSessionRepo) CountByUsername(_ context.Context, username string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.countErr != nil {
		return 0, r.countErr
	}
	n := 0
	for _, s := range r.sessions {
		if s.Username == username {
			n++
		}
	}
	return n, nil
}

func (r *fakeSessionRepo) BatchDeleteByNas(_ context.Context, nasAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if nasAddr == "" {
		return nil
	}
	for id, s := range r.sessions {
		if s.NasAddr == nasAddr {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *fakeSessionRepo) DeleteZombie(_ context.Context, threshold time.Duration) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-threshold)
	var removed int64
	for id, s := range r.sessions {
		if s.LastUpdate.Before(cutoff) {
			delete(r.sessions, id)
			removed++
		}
	}
	return removed, nil
}

// count 返回当前在线会话总数（测试断言辅助）。
func (r *fakeSessionRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sessions)
}

// —— fakeAccountingRepo ——

// fakeAccountingRepo 是内存版 AccountingRepository，语义对齐 GORM 实现：
// UpdateStop 由存储侧写入结算时间，未命中返回错误。
type fakeAccountingRepo struct {
	mu   sync.Mutex
	rows []*model.RadiusAccounting
}

func (r *fakeAccountingRepo) Create(_ context.Context, accounting *model.RadiusAccounting) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *accounting
	r.rows = append(r.rows, &cp)
	return nil
}

func (r *fakeAccountingRepo) UpdateStop(_ context.Context, acctSessionID string, accounting *model.RadiusAccounting) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.AcctSessionId == acctSessionID {
			now := time.Now()
			row.AcctStopTime = &now
			row.AcctInputTotal = accounting.AcctInputTotal
			row.AcctOutputTotal = accounting.AcctOutputTotal
			row.AcctInputPackets = accounting.AcctInputPackets
			row.AcctOutputPackets = accounting.AcctOutputPackets
			row.AcctSessionTime = accounting.AcctSessionTime
			row.LastUpdate = now
			return nil
		}
	}
	return fmt.Errorf("no accounting record with acct_session_id = %s", acctSessionID)
}

func (r *fakeAccountingRepo) PurgeBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var kept []*model.RadiusAccounting
	var removed int64
	for _, row := range r.rows {
		if row.AcctStopTime != nil && row.AcctStopTime.Before(cutoff) {
			removed++
			continue
		}
		kept = append(kept, row)
	}
	r.rows = kept
	return removed, nil
}

// count 返回记账记录总数（测试断言辅助）。
func (r *fakeAccountingRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.rows)
}

// find 按会话 ID 返回记账记录副本（测试断言辅助）。
func (r *fakeAccountingRepo) find(acctSessionID string) *model.RadiusAccounting {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.AcctSessionId == acctSessionID {
			cp := *row
			return &cp
		}
	}
	return nil
}

// —— fakeAuthLogger ——

// fakeAuthLogger 记录全部认证日志（RecordAuthLog 异步写入，断言需轮询等待）。
type fakeAuthLogger struct {
	mu      sync.Mutex
	entries []model.RadiusAuthLog
}

func (l *fakeAuthLogger) Create(_ context.Context, entry *model.RadiusAuthLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, *entry)
	return nil
}

func (l *fakeAuthLogger) CreateBatch(_ context.Context, entries []*model.RadiusAuthLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range entries {
		l.entries = append(l.entries, *e)
	}
	return nil
}

func (l *fakeAuthLogger) PurgeBefore(_ context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

// waitFor 轮询直到出现满足 pred 的日志条目或超时。
func (l *fakeAuthLogger) waitFor(t *testing.T, timeout time.Duration, pred func(model.RadiusAuthLog) bool) model.RadiusAuthLog {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		for _, e := range l.entries {
			if pred(e) {
				l.mu.Unlock()
				return e
			}
		}
		l.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("等待认证日志超时（%s 内未出现匹配条目）", timeout)
	return model.RadiusAuthLog{}
}

// —— 共享 UDP 测试服务 ——

// radiusTestEnv 聚合一套运行中的 RADIUS UDP 服务与其 fake 存储。
// 整个包共享一份：NewServer 会向全局 registry 注册插件，多次构建会重复注册。
type radiusTestEnv struct {
	cipher     *crypto.AESCipher
	users      *fakeUserRepo
	nas        *fakeNasRepo
	sessions   *fakeSessionRepo
	accounting *fakeAccountingRepo
	authLog    *fakeAuthLogger
	service    *RadiusService
	server     *Server
	authAddr   string
	acctAddr   string
}

var (
	sharedEnvOnce sync.Once
	sharedEnv     *radiusTestEnv
	sharedEnvErr  error
)

// TestMain 负责在全部测试结束后关闭共享 UDP 服务。
func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil && sharedEnv.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		sharedEnv.server.Shutdown(ctx)
		cancel()
	}
	os.Exit(code)
}

// getSharedTestEnv 惰性启动共享 RADIUS 测试服务。
func getSharedTestEnv(t *testing.T) *radiusTestEnv {
	t.Helper()
	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startRadiusTestEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("启动测试 RADIUS 服务失败: %v", sharedEnvErr)
	}
	return sharedEnv
}

// freeUDPPort 选取一个空闲 UDP 端口：先绑定 :0 取端口，关闭后按原端口
// 重新绑定验证确实可用，降低与 Server.Start 之间的竞争窗口。
func freeUDPPort() (int, error) {
	for i := 0; i < 20; i++ {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			continue
		}
		port := conn.LocalAddr().(*net.UDPAddr).Port
		_ = conn.Close()

		probe, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
		if err != nil {
			continue // 端口在关闭瞬间被抢占，重新选择
		}
		_ = probe.Close()
		return port, nil
	}
	return 0, fmt.Errorf("无法分配空闲 UDP 端口")
}

// startRadiusTestEnv 构建 fake 存储、种子数据并启动共享 UDP 服务。
// layeh PacketServer 需要具体端口，端口冲突时换端口重试。
func startRadiusTestEnv() (*radiusTestEnv, error) {
	cipher, err := crypto.NewAESCipher(testMasterKey)
	if err != nil {
		return nil, err
	}
	env := &radiusTestEnv{
		cipher:     cipher,
		users:      newFakeUserRepo(),
		nas:        &fakeNasRepo{},
		sessions:   newFakeSessionRepo(),
		accounting: &fakeAccountingRepo{},
		authLog:    &fakeAuthLogger{},
	}
	if err := env.seed(); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		authPort, err := freeUDPPort()
		if err != nil {
			lastErr = err
			continue
		}
		acctPort, err := freeUDPPort()
		if err != nil {
			lastErr = err
			continue
		}

		cfg := config.RadiusConfig{
			Enabled:             true,
			BindHost:            "127.0.0.1",
			AuthPort:            authPort,
			AcctPort:            acctPort,
			AcctInterimInterval: 600,
			HistoryDays:         0,
			MessageAuthMode:     MsgAuthModeWarn,
			EAPEnabled:          false,
			RadsecEnabled:       false,
		}
		service := NewRadiusService(cfg, zap.NewNop(), cipher,
			env.users, env.nas, env.sessions, env.accounting, env.authLog, nil)
		server := NewServer(service)
		server.Start()

		// 绑定失败时监听 goroutine 会尽快把错误写入 Errors()。
		time.Sleep(150 * time.Millisecond)
		if errs := server.Errors(); len(errs) > 0 {
			lastErr = errs[0]
			ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
			server.Shutdown(ctx)
			cancel()
			continue
		}

		env.service = service
		env.server = server
		env.authAddr = fmt.Sprintf("127.0.0.1:%d", authPort)
		env.acctAddr = fmt.Sprintf("127.0.0.1:%d", acctPort)
		return env, nil
	}
	return nil, fmt.Errorf("多次尝试后仍无法启动 RADIUS UDP 服务: %w", lastErr)
}

// seed 写入测试 NAS 与测试用户（密钥/密码均为密文存储）。
func (e *radiusTestEnv) seed() error {
	nasSecret, err := e.cipher.Encrypt(testNasSecret)
	if err != nil {
		return err
	}
	e.nas.add(&model.RadiusNas{
		ID:         1,
		Name:       "test-nas",
		Identifier: "test-nas",
		Ipaddr:     "127.0.0.1",
		Secret:     nasSecret,
		CoaPort:    DefaultCoAPort,
		Status:     model.RadiusNasStatusEnabled,
	})

	users := []struct {
		username string
		password string
		status   string
		expire   time.Time
	}{
		{"papuser", "pap-pass-1", model.RadiusUserStatusEnabled, time.Now().Add(24 * time.Hour)},
		{"chapuser", "chap-pass-9", model.RadiusUserStatusEnabled, time.Now().Add(24 * time.Hour)},
		{"disableduser", "disabled-pass", model.RadiusUserStatusDisabled, time.Now().Add(24 * time.Hour)},
		{"expireduser", "expired-pass", model.RadiusUserStatusEnabled, time.Now().Add(-time.Hour)},
	}
	for i, u := range users {
		encPassword, err := e.cipher.Encrypt(u.password)
		if err != nil {
			return err
		}
		e.users.add(&model.RadiusUser{
			ID:         uint64(i + 1),
			Username:   u.username,
			Password:   encPassword,
			Status:     u.status,
			ExpireTime: u.expire,
		})
	}
	return nil
}
