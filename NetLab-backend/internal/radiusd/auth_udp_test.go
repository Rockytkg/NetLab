package radiusd

import (
	"context"
	"crypto/md5"
	"net"
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"netlab-backend/internal/model"
)

// exchangeAuth 向共享认证端口发出一个 Access-Request 并返回响应。
func exchangeAuth(t *testing.T, client *radius.Client, packet *radius.Packet, addr string) *radius.Packet {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testExchangeTimeout)
	defer cancel()
	resp, err := client.Exchange(ctx, packet, addr)
	if err != nil {
		t.Fatalf("Access-Request 交换失败: %v", err)
	}
	return resp
}

// newPAPRequest 构造 PAP Access-Request（User-Password 由库按共享密钥加密）。
func newPAPRequest(t *testing.T, username, password string) *radius.Packet {
	t.Helper()
	packet := radius.New(radius.CodeAccessRequest, []byte(testNasSecret))
	if err := rfc2865.UserName_SetString(packet, username); err != nil {
		t.Fatalf("设置 User-Name 失败: %v", err)
	}
	if err := rfc2865.UserPassword_SetString(packet, password); err != nil {
		t.Fatalf("设置 User-Password 失败: %v", err)
	}
	return packet
}

// newCHAPRequest 构造 CHAP Access-Request（RFC 2865：md5(id+password+challenge)）。
func newCHAPRequest(t *testing.T, username, password string) *radius.Packet {
	t.Helper()
	challenge := []byte("0123456789abcdef") // 16 字节
	chapID := byte(0x2a)

	sum := md5.Sum(append(append([]byte{chapID}, password...), challenge...))
	chapPassword := append([]byte{chapID}, sum[:]...)

	packet := radius.New(radius.CodeAccessRequest, []byte(testNasSecret))
	if err := rfc2865.UserName_SetString(packet, username); err != nil {
		t.Fatalf("设置 User-Name 失败: %v", err)
	}
	if err := rfc2865.CHAPPassword_Set(packet, chapPassword); err != nil {
		t.Fatalf("设置 CHAP-Password 失败: %v", err)
	}
	if err := rfc2865.CHAPChallenge_Set(packet, challenge); err != nil {
		t.Fatalf("设置 CHAP-Challenge 失败: %v", err)
	}
	return packet
}

// TestAuthPAP_UDP 通过真实 UDP 交换验证 PAP 认证路径的接受/拒绝分支。
func TestAuthPAP_UDP(t *testing.T) {
	env := getSharedTestEnv(t)
	client := &radius.Client{Retry: 0, MaxPacketErrors: 10}

	cases := []struct {
		name     string
		username string
		password string
		wantCode radius.Code
	}{
		{"正确密码返回 Access-Accept", "papuser", "pap-pass-1", radius.CodeAccessAccept},
		{"错误密码返回 Access-Reject", "papuser", "wrong-pass", radius.CodeAccessReject},
		{"未知用户返回 Access-Reject", "nosuchuser", "whatever", radius.CodeAccessReject},
		{"停用用户返回 Access-Reject", "disableduser", "disabled-pass", radius.CodeAccessReject},
		{"过期用户返回 Access-Reject", "expireduser", "expired-pass", radius.CodeAccessReject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := exchangeAuth(t, client, newPAPRequest(t, tc.username, tc.password), env.authAddr)
			if resp.Code != tc.wantCode {
				t.Errorf("响应代码 = %v，期望 %v", resp.Code, tc.wantCode)
			}
		})
	}
}

// TestAuthCHAP_UDP 通过真实 UDP 交换验证 CHAP 认证路径。
func TestAuthCHAP_UDP(t *testing.T) {
	env := getSharedTestEnv(t)
	client := &radius.Client{Retry: 0, MaxPacketErrors: 10}

	cases := []struct {
		name     string
		username string
		password string
		wantCode radius.Code
	}{
		{"正确密码返回 Access-Accept", "chapuser", "chap-pass-9", radius.CodeAccessAccept},
		{"错误密码返回 Access-Reject", "chapuser", "wrong-pass", radius.CodeAccessReject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := exchangeAuth(t, client, newCHAPRequest(t, tc.username, tc.password), env.authAddr)
			if resp.Code != tc.wantCode {
				t.Errorf("响应代码 = %v，期望 %v", resp.Code, tc.wantCode)
			}
		})
	}
}

// TestAuthUnknownNAS_UDP 从未注册的源 IP（127.0.0.2）发起请求，
// 服务端应以 Access-Reject 拒绝（响应以占位密钥签名，客户端跳过验签）。
func TestAuthUnknownNAS_UDP(t *testing.T) {
	env := getSharedTestEnv(t)
	client := &radius.Client{
		Retry:              0,
		MaxPacketErrors:    10,
		InsecureSkipVerify: true, // 未知 NAS 的响应用占位密钥签名，无法按真实密钥验签
	}
	client.Dialer.LocalAddr = &net.UDPAddr{IP: net.ParseIP("127.0.0.2"), Port: 0}

	resp := exchangeAuth(t, client, newPAPRequest(t, "papuser", "pap-pass-1"), env.authAddr)
	if resp.Code != radius.CodeAccessReject {
		t.Errorf("响应代码 = %v，期望 Access-Reject", resp.Code)
	}
}

// TestAuthLogRecorded_UDP 验证认证结果（通过/拒绝）均异步写入认证日志。
func TestAuthLogRecorded_UDP(t *testing.T) {
	env := getSharedTestEnv(t)
	client := &radius.Client{Retry: 0, MaxPacketErrors: 10}

	// 通过路径：papuser 正确密码。
	resp := exchangeAuth(t, client, newPAPRequest(t, "papuser", "pap-pass-1"), env.authAddr)
	if resp.Code != radius.CodeAccessAccept {
		t.Fatalf("前置认证失败，响应代码 = %v", resp.Code)
	}
	acceptEntry := env.authLog.waitFor(t, testExchangeTimeout, func(e model.RadiusAuthLog) bool {
		return e.Username == "papuser" && e.Result == model.RadiusAuthResultAccept
	})
	if acceptEntry.AuthType != "pap" {
		t.Errorf("accept 日志 AuthType = %q，期望 %q", acceptEntry.AuthType, "pap")
	}
	if acceptEntry.NasAddr != "127.0.0.1" {
		t.Errorf("accept 日志 NasAddr = %q，期望 %q", acceptEntry.NasAddr, "127.0.0.1")
	}

	// 拒绝路径：不存在的用户。
	resp = exchangeAuth(t, client, newPAPRequest(t, "ghostuser", "whatever"), env.authAddr)
	if resp.Code != radius.CodeAccessReject {
		t.Fatalf("前置认证应为拒绝，响应代码 = %v", resp.Code)
	}
	rejectEntry := env.authLog.waitFor(t, testExchangeTimeout, func(e model.RadiusAuthLog) bool {
		return e.Username == "ghostuser" && e.Result == model.RadiusAuthResultReject
	})
	if rejectEntry.Reason == "" {
		t.Error("reject 日志 Reason 不应为空")
	}
	if rejectEntry.AuthType != "pap" {
		t.Errorf("reject 日志 AuthType = %q，期望 %q", rejectEntry.AuthType, "pap")
	}
}
