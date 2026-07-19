package radiusd

import (
	"context"
	"net"
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

// newAcctPacket 构造 Accounting-Request；layeh 客户端在 Encode 时会按
// 报文 Secret 计算 RFC 2866 keyed Request Authenticator。
func newAcctPacket(t *testing.T, statusType rfc2866.AcctStatusType, username, sessionID string) *radius.Packet {
	t.Helper()
	packet := radius.New(radius.CodeAccountingRequest, []byte(testNasSecret))
	if username != "" {
		if err := rfc2865.UserName_SetString(packet, username); err != nil {
			t.Fatalf("设置 User-Name 失败: %v", err)
		}
	}
	if sessionID != "" {
		if err := rfc2866.AcctSessionID_SetString(packet, sessionID); err != nil {
			t.Fatalf("设置 Acct-Session-Id 失败: %v", err)
		}
	}
	if err := rfc2866.AcctStatusType_Set(packet, statusType); err != nil {
		t.Fatalf("设置 Acct-Status-Type 失败: %v", err)
	}
	return packet
}

// exchangeAcct 发出一个 Accounting-Request 并断言收到 Accounting-Response。
func exchangeAcct(t *testing.T, client *radius.Client, packet *radius.Packet, addr string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testExchangeTimeout)
	defer cancel()
	resp, err := client.Exchange(ctx, packet, addr)
	if err != nil {
		t.Fatalf("Accounting-Request 交换失败: %v", err)
	}
	if resp.Code != radius.CodeAccountingResponse {
		t.Fatalf("响应代码 = %v，期望 Accounting-Response", resp.Code)
	}
}

// TestAccountingFlow_UDP 通过真实 UDP 交换驱动 Start/Interim/Stop/On 全链路，
// 每个步骤断言 fake 存储中的会话与记账记录状态。步骤按序执行、共享状态，
// 因此用步骤表而非独立子测试。
func TestAccountingFlow_UDP(t *testing.T) {
	env := getSharedTestEnv(t)
	client := &radius.Client{Retry: 0, MaxPacketErrors: 10}

	const (
		user1  = "acctuser1"
		user2  = "acctuser2"
		sess1  = "sess-0001"
		sess2  = "sess-0002"
		sess3  = "sess-0003"
		nasIP  = "127.0.0.1"
		framed = "10.0.0.10"
	)

	steps := []struct {
		name  string
		build func(t *testing.T) *radius.Packet
		check func(t *testing.T)
	}{
		{
			name: "Start 建档在线会话与记账记录",
			build: func(t *testing.T) *radius.Packet {
				p := newAcctPacket(t, rfc2866.AcctStatusType_Value_Start, user1, sess1)
				if err := rfc2865.FramedIPAddress_Set(p, net.ParseIP(framed)); err != nil {
					t.Fatalf("设置 Framed-IP-Address 失败: %v", err)
				}
				return p
			},
			check: func(t *testing.T) {
				online, err := env.sessions.GetBySessionID(context.Background(), sess1)
				if err != nil || online == nil {
					t.Fatalf("Start 后在线会话不存在: err=%v", err)
				}
				if online.Username != user1 {
					t.Errorf("在线会话 Username = %q，期望 %q", online.Username, user1)
				}
				if online.NasAddr != nasIP {
					t.Errorf("在线会话 NasAddr = %q，期望 %q", online.NasAddr, nasIP)
				}
				if online.FramedIpaddr != framed {
					t.Errorf("在线会话 FramedIpaddr = %q，期望 %q", online.FramedIpaddr, framed)
				}
				if env.accounting.count() != 1 {
					t.Errorf("记账记录数 = %d，期望 1", env.accounting.count())
				}
				row := env.accounting.find(sess1)
				if row == nil {
					t.Fatal("Start 后记账记录不存在")
				}
				if row.AcctStopTime != nil {
					t.Error("进行中的记账记录不应有结算时间")
				}
			},
		},
		{
			name: "重复 Start 幂等不产生重复记录",
			build: func(t *testing.T) *radius.Packet {
				return newAcctPacket(t, rfc2866.AcctStatusType_Value_Start, user1, sess1)
			},
			check: func(t *testing.T) {
				if env.sessions.count() != 1 {
					t.Errorf("在线会话数 = %d，期望 1", env.sessions.count())
				}
				if env.accounting.count() != 1 {
					t.Errorf("记账记录数 = %d，期望 1（重复 Start 不应再建档）", env.accounting.count())
				}
			},
		},
		{
			name: "Interim-Update 只更新计数器",
			build: func(t *testing.T) *radius.Packet {
				p := newAcctPacket(t, rfc2866.AcctStatusType_Value_InterimUpdate, user1, sess1)
				_ = rfc2866.AcctInputOctets_Set(p, 1024)
				_ = rfc2866.AcctOutputOctets_Set(p, 2048)
				_ = rfc2866.AcctInputPackets_Set(p, 10)
				_ = rfc2866.AcctOutputPackets_Set(p, 20)
				_ = rfc2866.AcctSessionTime_Set(p, 60)
				return p
			},
			check: func(t *testing.T) {
				online, err := env.sessions.GetBySessionID(context.Background(), sess1)
				if err != nil || online == nil {
					t.Fatalf("Interim 后在线会话不存在: err=%v", err)
				}
				if online.AcctInputTotal != 1024 || online.AcctOutputTotal != 2048 {
					t.Errorf("流量计数 = (%d, %d)，期望 (1024, 2048)",
						online.AcctInputTotal, online.AcctOutputTotal)
				}
				if online.AcctInputPackets != 10 || online.AcctOutputPackets != 20 {
					t.Errorf("包计数 = (%d, %d)，期望 (10, 20)",
						online.AcctInputPackets, online.AcctOutputPackets)
				}
				if online.AcctSessionTime != 60 {
					t.Errorf("会话时长 = %d，期望 60", online.AcctSessionTime)
				}
				if env.accounting.count() != 1 {
					t.Errorf("记账记录数 = %d，期望 1（Interim 不应新增记录）", env.accounting.count())
				}
			},
		},
		{
			name: "Stop 结算记账并下线会话",
			build: func(t *testing.T) *radius.Packet {
				p := newAcctPacket(t, rfc2866.AcctStatusType_Value_Stop, user1, sess1)
				_ = rfc2866.AcctInputOctets_Set(p, 4096)
				_ = rfc2866.AcctOutputOctets_Set(p, 8192)
				_ = rfc2866.AcctSessionTime_Set(p, 120)
				return p
			},
			check: func(t *testing.T) {
				row := env.accounting.find(sess1)
				if row == nil {
					t.Fatal("Stop 后记账记录不存在")
				}
				if row.AcctStopTime == nil {
					t.Error("Stop 后记账记录应有结算时间")
				}
				if row.AcctInputTotal != 4096 || row.AcctOutputTotal != 8192 {
					t.Errorf("结算流量 = (%d, %d)，期望 (4096, 8192)",
						row.AcctInputTotal, row.AcctOutputTotal)
				}
				if row.AcctSessionTime != 120 {
					t.Errorf("结算会话时长 = %d，期望 120", row.AcctSessionTime)
				}
				online, err := env.sessions.GetBySessionID(context.Background(), sess1)
				if err != nil {
					t.Fatalf("查询在线会话失败: %v", err)
				}
				if online != nil {
					t.Error("Stop 后在线会话应被删除")
				}
			},
		},
		{
			name: "Start 会话 sess-0002",
			build: func(t *testing.T) *radius.Packet {
				return newAcctPacket(t, rfc2866.AcctStatusType_Value_Start, user1, sess2)
			},
			check: func(t *testing.T) {
				if env.sessions.count() != 1 {
					t.Errorf("在线会话数 = %d，期望 1", env.sessions.count())
				}
			},
		},
		{
			name: "Start 会话 sess-0003",
			build: func(t *testing.T) *radius.Packet {
				return newAcctPacket(t, rfc2866.AcctStatusType_Value_Start, user2, sess3)
			},
			check: func(t *testing.T) {
				if env.sessions.count() != 2 {
					t.Errorf("在线会话数 = %d，期望 2", env.sessions.count())
				}
				if env.accounting.count() != 3 {
					t.Errorf("记账记录数 = %d，期望 3", env.accounting.count())
				}
			},
		},
		{
			name: "Accounting-On 清空该 NAS 全部会话",
			build: func(t *testing.T) *radius.Packet {
				return newAcctPacket(t, rfc2866.AcctStatusType_Value_AccountingOn, "", "")
			},
			check: func(t *testing.T) {
				if env.sessions.count() != 0 {
					t.Errorf("Accounting-On 后在线会话数 = %d，期望 0", env.sessions.count())
				}
				if env.accounting.count() != 3 {
					t.Errorf("Accounting-On 不应删除记账记录，记录数 = %d，期望 3", env.accounting.count())
				}
			},
		},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			exchangeAcct(t, client, step.build(t), env.acctAddr)
			step.check(t)
		})
	}
}
