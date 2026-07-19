package radiusd

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/plugins/auth/enhancers"
	"netlab-backend/internal/radiusd/plugins/auth/guards"
	"netlab-backend/pkg/crypto"
)

// TestConfigStore 验证配置容器的读写与零值行为。
func TestConfigStore(t *testing.T) {
	store := NewConfigStore(config.RadiusConfig{AuthPort: 1812, MessageAuthMode: "warn"})
	if got := store.Get(); got.AuthPort != 1812 || got.MessageAuthMode != "warn" {
		t.Fatalf("unexpected initial config: %+v", got)
	}
	store.Set(config.RadiusConfig{AuthPort: 11812, EAPEnabled: true})
	if got := store.Get(); got.AuthPort != 11812 || !got.EAPEnabled {
		t.Fatalf("unexpected updated config: %+v", got)
	}
}

// TestEapMethodAllowed 验证启用列表匹配（"*" 与空为全部，逗号列表逐名匹配）。
func TestEapMethodAllowed(t *testing.T) {
	cases := []struct {
		list    string
		method  string
		allowed bool
	}{
		{"*", "eap-md5", true},
		{"", "eap-tls", true},
		{" eap-md5 , eap-mschapv2 ", "eap-md5", true},
		{"eap-md5,eap-mschapv2", "EAP-MSCHAPV2", true},
		{"eap-md5", "eap-tls", false},
		{"eap-tls,eap-peap", "eap-ttls", false},
	}
	for _, c := range cases {
		if got := eapMethodAllowed(c.list, c.method); got != c.allowed {
			t.Errorf("eapMethodAllowed(%q, %q) = %v, want %v", c.list, c.method, got, c.allowed)
		}
	}
}

// TestListenerConfigChanged 验证监听级字段差异判定。
func TestListenerConfigChanged(t *testing.T) {
	base := config.RadiusConfig{
		Enabled: true, BindHost: "0.0.0.0", AuthPort: 1812, AcctPort: 1813,
		RadsecEnabled: true, RadsecPort: 2083, RadsecCertID: 1,
	}
	if listenerConfigChanged(base, base) {
		t.Fatal("identical configs should not be marked changed")
	}
	// 非监听级字段变化不应触发重建
	hot := base
	hot.EAPMethod = "eap-tls"
	hot.SessionTimeout = 3600
	hot.RejectDelayMaxRejects = 99
	if listenerConfigChanged(base, hot) {
		t.Fatal("hot-level fields must not trigger listener rebuild")
	}
	// 逐字段监听级差异
	for _, mutate := range []func(*config.RadiusConfig){
		func(c *config.RadiusConfig) { c.BindHost = "127.0.0.1" },
		func(c *config.RadiusConfig) { c.AuthPort = 11812 },
		func(c *config.RadiusConfig) { c.AcctPort = 11813 },
		func(c *config.RadiusConfig) { c.RadsecEnabled = false },
		func(c *config.RadiusConfig) { c.RadsecPort = 12083 },
		func(c *config.RadiusConfig) { c.RadsecCertID = 2 },
		func(c *config.RadiusConfig) { c.RadsecCACertID = 3 },
		func(c *config.RadiusConfig) { c.RadsecCertFile = "/tmp/a.pem" },
	} {
		changed := base
		mutate(&changed)
		if !listenerConfigChanged(base, changed) {
			t.Errorf("expected listener-level change detected: %+v vs %+v", base, changed)
		}
	}
}

// TestRejectDelayGuardLiveParams 验证守卫阈值/窗口从回调动态读取。
func TestRejectDelayGuardLiveParams(t *testing.T) {
	maxRejects := int64(2)
	window := time.Hour
	guard := guards.NewRejectDelayGuard(
		func() int64 { return maxRejects },
		func() time.Duration { return window },
	)

	authCtx := &auth.AuthContext{Metadata: map[string]interface{}{"username": "alice"}}
	baseErr := context.DeadlineExceeded

	// 守卫语义为 rejects > limit 才封禁：limit=2 时前三次放行（计数 1/2/3），
	// 第四次（计数 3 > 2）触发封禁
	for i := 0; i < 3; i++ {
		if res := guard.OnAuthError(context.Background(), authCtx, "reject", baseErr); res.Action != auth.GuardActionContinue {
			t.Fatalf("attempt %d: expected continue, got %v", i+1, res.Action)
		}
	}
	if res := guard.OnAuthError(context.Background(), authCtx, "reject", baseErr); res.Action != auth.GuardActionStop {
		t.Fatalf("expected stop after threshold exceeded, got %v", res.Action)
	}

	// 动态调高阈值后立即放行
	maxRejects = 100
	if res := guard.OnAuthError(context.Background(), authCtx, "reject", baseErr); res.Action != auth.GuardActionContinue {
		t.Fatalf("expected continue after raising threshold, got %v", res.Action)
	}

	// 窗口收窄为 0（<=0 回落默认 1s）：等待默认窗口过后计数重置
	maxRejects = 0
	window = 0
	time.Sleep(1100 * time.Millisecond)
	if res := guard.OnAuthError(context.Background(), authCtx, "reject", baseErr); res.Action != auth.GuardActionContinue {
		t.Fatalf("expected continue after window reset, got %v", res.Action)
	}
}

// TestDefaultAcceptEnhancerSessionTimeoutCap 验证 Session-Timeout 上限收敛。
func TestDefaultAcceptEnhancerSessionTimeoutCap(t *testing.T) {
	enhancer := enhancers.NewDefaultAcceptEnhancer()
	user := &model.RadiusUser{Username: "bob", ExpireTime: time.Now().Add(24 * time.Hour)}

	build := func(capSec int) *radius.Packet {
		resp := radius.New(radius.CodeAccessAccept, []byte(testNasSecret))
		authCtx := &auth.AuthContext{
			User:     user,
			Response: resp,
			Metadata: map[string]interface{}{
				"acct_interim_interval": 120,
				"session_timeout_cap":   capSec,
			},
		}
		if err := enhancer.Enhance(context.Background(), authCtx); err != nil {
			t.Fatalf("enhance: %v", err)
		}
		return resp
	}

	// 无上限（0）：按过期倒计时下发（约 24h）
	noCap := build(0)
	if got := int(rfc2865.SessionTimeout_Get(noCap)); got < 86000 || got > 86400 {
		t.Fatalf("uncapped session timeout = %d, want ~86400", got)
	}
	// 上限 3600：收敛到 3600
	capped := build(3600)
	if got := int(rfc2865.SessionTimeout_Get(capped)); got != 3600 {
		t.Fatalf("capped session timeout = %d, want 3600", got)
	}
}

// TestCoAAttributeSetters 验证 CoA 授权变更属性写入。
func TestCoAAttributeSetters(t *testing.T) {
	packet := radius.New(radius.CodeCoARequest, []byte(testNasSecret))
	if err := WithSessionTimeout(3600)(packet); err != nil {
		t.Fatalf("WithSessionTimeout: %v", err)
	}
	if err := WithFilterID("throttled")(packet); err != nil {
		t.Fatalf("WithFilterID: %v", err)
	}
	if got := int(rfc2865.SessionTimeout_Get(packet)); got != 3600 {
		t.Fatalf("session timeout = %d, want 3600", got)
	}
	if got := rfc2865.FilterID_GetString(packet); got != "throttled" {
		t.Fatalf("filter id = %q, want throttled", got)
	}
}

// TestManagerApplyLifecycle 验证 Manager 的启动/热更新/重建/停止流转。
func TestManagerApplyLifecycle(t *testing.T) {
	cipher, err := crypto.NewAESCipher(testMasterKey)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	m := NewManager(
		zap.NewNop(), cipher,
		newFakeUserRepo(), &fakeNasRepo{}, newFakeSessionRepo(), &fakeAccountingRepo{}, &fakeAuthLogger{},
		nil, nil,
	)
	defer m.Shutdown(context.Background())

	if m.Running() {
		t.Fatal("manager should not be running initially")
	}
	if m.CoA() != nil {
		t.Fatal("CoA should be nil before start")
	}

	base := config.RadiusConfig{
		Enabled: true, BindHost: "127.0.0.1", AuthPort: 0, AcctPort: 0,
		AcctInterimInterval: 120, MessageAuthMode: "warn",
	}
	m.Apply(base)
	if !m.Running() {
		t.Fatal("manager should be running after Apply(enabled)")
	}
	if m.CoA() == nil {
		t.Fatal("CoA should be available while running")
	}
	server1 := m.server

	// 非监听级变更：热更新，Server 实例不变
	hot := base
	hot.SessionTimeout = 1800
	hot.EAPMethod = "eap-mschapv2"
	m.Apply(hot)
	if m.server != server1 {
		t.Fatal("hot update must not rebuild the server")
	}
	if got := m.server.service.cfg(); got.SessionTimeout != 1800 || got.EAPMethod != "eap-mschapv2" {
		t.Fatalf("hot update not reflected: %+v", got)
	}

	// 监听级变更：重建 Server 实例
	restart := hot
	restart.BindHost = "127.0.0.2"
	if _, err := net.ResolveIPAddr("ip", "127.0.0.2"); err == nil {
		m.Apply(restart)
		if m.server == server1 {
			t.Fatal("listener-level change must rebuild the server")
		}
		if !m.Running() {
			t.Fatal("manager should still be running after rebuild")
		}
	}

	// 禁用：停止
	m.Apply(config.RadiusConfig{Enabled: false})
	if m.Running() {
		t.Fatal("manager should stop after Apply(disabled)")
	}
	if m.CoA() != nil {
		t.Fatal("CoA should be nil after stop")
	}
}
