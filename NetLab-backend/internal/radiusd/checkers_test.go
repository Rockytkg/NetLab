package radiusd

import (
	"context"
	"errors"
	"testing"
	"time"

	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/plugins/auth/checkers"
)

// TestStatusChecker 验证用户状态检查器：仅 disabled 状态被拒绝。
func TestStatusChecker(t *testing.T) {
	checker := &checkers.StatusChecker{}

	cases := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"启用用户通过", model.RadiusUserStatusEnabled, false},
		{"停用用户拒绝", model.RadiusUserStatusDisabled, true},
		{"空状态通过", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authCtx := &auth.AuthContext{User: &model.RadiusUser{Username: "u1", Status: tc.status}}
			err := checker.Check(context.Background(), authCtx)
			if (err != nil) != tc.wantErr {
				t.Errorf("Check() err = %v，期望出错 = %v", err, tc.wantErr)
			}
		})
	}
}

// TestExpireChecker 验证账号有效期检查器：过期时间早于当前时刻被拒绝。
func TestExpireChecker(t *testing.T) {
	checker := &checkers.ExpireChecker{}

	cases := []struct {
		name       string
		expireTime time.Time
		wantErr    bool
	}{
		{"未来过期时间通过", time.Now().Add(24 * time.Hour), false},
		{"过去过期时间拒绝", time.Now().Add(-time.Hour), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authCtx := &auth.AuthContext{User: &model.RadiusUser{Username: "u1", ExpireTime: tc.expireTime}}
			err := checker.Check(context.Background(), authCtx)
			if (err != nil) != tc.wantErr {
				t.Errorf("Check() err = %v，期望出错 = %v", err, tc.wantErr)
			}
		})
	}
}

// TestOnlineCountChecker 验证并发在线数检查器，含套餐继承与存储错误传播。
func TestOnlineCountChecker(t *testing.T) {
	errBoom := errors.New("storage boom")

	// seedSessions 向 fake 会话存储写入 n 条该用户的在线会话。
	seedSessions := func(repo *fakeSessionRepo, username string, n int) {
		t.Helper()
		for i := 0; i < n; i++ {
			session := &model.RadiusOnline{
				Username:      username,
				AcctSessionId: username + "-sess-" + string(rune('a'+i)),
				LastUpdate:    time.Now(),
			}
			if _, err := repo.Create(context.Background(), session); err != nil {
				t.Fatalf("写入在线会话失败: %v", err)
			}
		}
	}

	cases := []struct {
		name     string
		user     *model.RadiusUser
		online   int   // 预置在线会话数
		countErr error // 非空时注入存储错误
		wantErr  bool
	}{
		{
			name:   "无并发限制（ActiveNum=0）不校验",
			user:   &model.RadiusUser{Username: "u-nolimit", ActiveNum: 0, ProfileLinkMode: model.RadiusLinkModeStatic},
			online: 5,
		},
		{
			name:   "在线数未达上限通过",
			user:   &model.RadiusUser{Username: "u-under", ActiveNum: 2, ProfileLinkMode: model.RadiusLinkModeStatic},
			online: 1,
		},
		{
			name:    "在线数达到上限拒绝",
			user:    &model.RadiusUser{Username: "u-full", ActiveNum: 2, ProfileLinkMode: model.RadiusLinkModeStatic},
			online:  2,
			wantErr: true,
		},
		{
			name: "动态继承套餐上限，达到上限拒绝",
			user: &model.RadiusUser{
				Username:        "u-profile",
				ActiveNum:       0,
				ProfileLinkMode: model.RadiusLinkModeDynamic,
				Profile:         &model.RadiusProfile{ActiveNum: 1},
			},
			online:  1,
			wantErr: true,
		},
		{
			name: "动态继承套餐上限，未达上限通过",
			user: &model.RadiusUser{
				Username:        "u-profile-ok",
				ActiveNum:       0,
				ProfileLinkMode: model.RadiusLinkModeDynamic,
				Profile:         &model.RadiusProfile{ActiveNum: 2},
			},
			online: 1,
		},
		{
			name: "静态模式忽略套餐上限",
			user: &model.RadiusUser{
				Username:        "u-static",
				ActiveNum:       0,
				ProfileLinkMode: model.RadiusLinkModeStatic,
				Profile:         &model.RadiusProfile{ActiveNum: 1},
			},
			online: 3,
		},
		{
			name:     "存储错误向上传播",
			user:     &model.RadiusUser{Username: "u-err", ActiveNum: 1, ProfileLinkMode: model.RadiusLinkModeStatic},
			countErr: errBoom,
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeSessionRepo()
			repo.countErr = tc.countErr
			seedSessions(repo, tc.user.Username, tc.online)

			checker := checkers.NewOnlineCountChecker(repo)
			authCtx := &auth.AuthContext{User: tc.user}
			err := checker.Check(context.Background(), authCtx)
			if (err != nil) != tc.wantErr {
				t.Errorf("Check() err = %v，期望出错 = %v", err, tc.wantErr)
			}
			if tc.countErr != nil && !errors.Is(err, tc.countErr) {
				t.Errorf("Check() err = %v，期望包装 %v", err, tc.countErr)
			}
		})
	}
}
