package radiusd

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc3576"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/pkg/crypto"
)

// TestSessionIdentityFromOnline 验证在线会话到 RFC 5176 会话识别属性的映射，
// 重点覆盖 NAS-Port 的可选指针语义（仅 0 < port <= MaxUint32 时下发）。
func TestSessionIdentityFromOnline(t *testing.T) {
	cases := []struct {
		name     string
		online   *model.RadiusOnline
		want     SessionIdentity
		wantPort bool // 是否期望 NasPort 非空
	}{
		{
			name: "完整字段映射",
			online: &model.RadiusOnline{
				Username:      "alice",
				NasAddr:       "10.0.0.1",
				AcctSessionId: "sess-1",
				FramedIpaddr:  "10.0.0.9",
				MacAddr:       "aa:bb:cc:dd:ee:ff",
				NasPortId:     "eth0",
				NasPort:       12345,
			},
			want: SessionIdentity{
				Username:       "alice",
				NasIP:          "10.0.0.1",
				AcctSessionID:  "sess-1",
				FramedIP:       "10.0.0.9",
				CallingStation: "aa:bb:cc:dd:ee:ff",
				NasPortID:      "eth0",
			},
			wantPort: true,
		},
		{
			name:     "NasPort 为 0 不下发",
			online:   &model.RadiusOnline{Username: "bob", NasPort: 0},
			want:     SessionIdentity{Username: "bob"},
			wantPort: false,
		},
		{
			name:     "NasPort 为负不下发",
			online:   &model.RadiusOnline{Username: "bob", NasPort: -5},
			want:     SessionIdentity{Username: "bob"},
			wantPort: false,
		},
		{
			name:     "NasPort 超出 uint32 不下发",
			online:   &model.RadiusOnline{Username: "bob", NasPort: math.MaxUint32 + 1},
			want:     SessionIdentity{Username: "bob"},
			wantPort: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SessionIdentityFromOnline(tc.online)
			if got.Username != tc.want.Username ||
				got.NasIP != tc.want.NasIP ||
				got.NasIdentifier != tc.want.NasIdentifier ||
				got.AcctSessionID != tc.want.AcctSessionID ||
				got.FramedIP != tc.want.FramedIP ||
				got.CallingStation != tc.want.CallingStation ||
				got.NasPortID != tc.want.NasPortID {
				t.Errorf("SessionIdentity = %+v，期望 %+v", got, tc.want)
			}
			if tc.wantPort {
				if got.NasPort == nil {
					t.Fatal("期望 NasPort 非空")
				}
				if *got.NasPort != uint32(tc.online.NasPort) {
					t.Errorf("NasPort = %d，期望 %d", *got.NasPort, tc.online.NasPort)
				}
			} else if got.NasPort != nil {
				t.Errorf("期望 NasPort 为空，实际 = %d", *got.NasPort)
			}
		})
	}
}

// TestCoATargetFromNas 验证 NAS 记录到 CoA 目标的映射与默认端口（RFC 5176 §2.1: 3799）。
func TestCoATargetFromNas(t *testing.T) {
	cases := []struct {
		name         string
		nas          *model.RadiusNas
		wantEndpoint string
	}{
		{
			name:         "显式 CoA 端口",
			nas:          &model.RadiusNas{Ipaddr: "10.0.0.1", Secret: "s3cret", CoaPort: 4000},
			wantEndpoint: "10.0.0.1:4000",
		},
		{
			name:         "默认端口 3799",
			nas:          &model.RadiusNas{Ipaddr: "10.0.0.1", Secret: "s3cret", CoaPort: DefaultCoAPort},
			wantEndpoint: "10.0.0.1:3799",
		},
		{
			name:         "未配置端口回落 3799",
			nas:          &model.RadiusNas{Ipaddr: "10.0.0.1", Secret: "s3cret", CoaPort: 0},
			wantEndpoint: "10.0.0.1:3799",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := CoATargetFromNas(tc.nas)
			if target.Addr != tc.nas.Ipaddr {
				t.Errorf("Addr = %q，期望 %q", target.Addr, tc.nas.Ipaddr)
			}
			if target.Secret != tc.nas.Secret {
				t.Errorf("Secret = %q，期望 %q", target.Secret, tc.nas.Secret)
			}
			if got := target.endpoint(); got != tc.wantEndpoint {
				t.Errorf("endpoint() = %q，期望 %q", got, tc.wantEndpoint)
			}
		})
	}
}

// startFakeCoANAS 启动一个假 NAS 的动态授权 UDP 服务：
// 对 Acct-Session-Id 为 "sess-nak" 的请求回 Disconnect-NAK（含 Error-Cause），
// 其余回 Disconnect-ACK。返回监听端口。
func startFakeCoANAS(t *testing.T, secret string) int {
	t.Helper()
	port, err := freeUDPPort()
	if err != nil {
		t.Fatalf("分配 CoA 端口失败: %v", err)
	}
	srv := &radius.PacketServer{
		Addr: net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)),
		Handler: radius.HandlerFunc(func(w radius.ResponseWriter, r *radius.Request) {
			if rfc2866.AcctSessionID_GetString(r.Packet) == "sess-nak" {
				resp := r.Response(radius.CodeDisconnectNAK)
				_ = rfc3576.ErrorCause_Set(resp, rfc3576.ErrorCause_Value_SessionContextNotFound)
				_ = w.Write(resp)
				return
			}
			_ = w.Write(r.Response(radius.CodeDisconnectACK))
		}),
		SecretSource: radius.StaticSecretSource([]byte(secret)),
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	// 等待监听就绪（UDP 无连接语义，短暂等待后由客户端重传兜底）。
	time.Sleep(150 * time.Millisecond)
	return port
}

// TestDisconnectSessionRoundTrip 端到端验证 DisconnectSession：
// 会话解析 → NAS 解析 → Disconnect-Request → ACK/NAK 分类。
func TestDisconnectSessionRoundTrip(t *testing.T) {
	const secret = "coa-secret-1"
	port := startFakeCoANAS(t, secret)

	cipher, err := crypto.NewAESCipher(testMasterKey)
	if err != nil {
		t.Fatalf("创建加密组件失败: %v", err)
	}
	encSecret, err := cipher.Encrypt(secret)
	if err != nil {
		t.Fatalf("加密 NAS 密钥失败: %v", err)
	}

	nasRepo := &fakeNasRepo{}
	nasRepo.add(&model.RadiusNas{
		ID:      1,
		Name:    "coa-nas",
		Ipaddr:  "127.0.0.1",
		Secret:  encSecret,
		CoaPort: port,
		Status:  model.RadiusNasStatusEnabled,
	})

	sessionRepo := newFakeSessionRepo()
	for _, sid := range []string{"sess-ack", "sess-nak"} {
		if _, err := sessionRepo.Create(context.Background(), &model.RadiusOnline{
			Username:      "coauser",
			NasAddr:       "127.0.0.1",
			AcctSessionId: sid,
			LastUpdate:    time.Now(),
		}); err != nil {
			t.Fatalf("写入在线会话失败: %v", err)
		}
	}

	svc := NewRadiusService(config.RadiusConfig{}, zap.NewNop(), cipher,
		newFakeUserRepo(), nasRepo, sessionRepo, &fakeAccountingRepo{}, &fakeAuthLogger{}, nil)
	coa := NewCoAService(svc)
	coa.timeout = 500 * time.Millisecond // 缩短单次交换超时，加快失败路径

	cases := []struct {
		name         string
		sessionID    string
		wantSuccess  bool
		wantCode     string
		wantErrCause int
		wantErr      error
	}{
		{
			name:        "ACK 下线成功",
			sessionID:   "sess-ack",
			wantSuccess: true,
			wantCode:    "Disconnect-ACK",
		},
		{
			name:         "NAK 携带 Error-Cause",
			sessionID:    "sess-nak",
			wantSuccess:  false,
			wantCode:     "Disconnect-NAK",
			wantErrCause: int(rfc3576.ErrorCause_Value_SessionContextNotFound),
		},
		{
			name:      "会话不存在返回 ErrSessionNotFound",
			sessionID: "sess-missing",
			wantErr:   ErrSessionNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testExchangeTimeout)
			defer cancel()
			result, err := coa.DisconnectSession(ctx, tc.sessionID)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("DisconnectSession() err = %v，期望 %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DisconnectSession() err = %v", err)
			}
			if result.Success != tc.wantSuccess {
				t.Errorf("Success = %v，期望 %v", result.Success, tc.wantSuccess)
			}
			if result.ResponseCode != tc.wantCode {
				t.Errorf("ResponseCode = %q，期望 %q", result.ResponseCode, tc.wantCode)
			}
			if tc.wantErrCause != 0 && result.ErrorCause != tc.wantErrCause {
				t.Errorf("ErrorCause = %d，期望 %d", result.ErrorCause, tc.wantErrCause)
			}
			if result.TimedOut {
				t.Error("不应标记为超时")
			}
			if result.Attempts != 1 {
				t.Errorf("Attempts = %d，期望 1", result.Attempts)
			}
		})
	}
}
