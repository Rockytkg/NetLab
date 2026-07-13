package auth

import (
	"context"
	"time"

	"netlab-backend/internal/model"
	sysconfig "netlab-backend/internal/service/config"
)

// computeSecurityActions 汇总登录后需要用户立即处理的强制动作。
// 邮箱只在首次登录引导中强制更新；密码过期和密码重置只要求改密码。
// AuthService、OAuthService、
// PasskeyService 与 TwoFactorService 共用此逻辑，避免多处重复实现。
func computeSecurityActions(ctx context.Context, cs *sysconfig.Service, u *model.User) SecurityActionsResult {
	passwordExpired := passwordExpiredFor(ctx, cs, u)
	requireEmail := u.ForcePasswordChange && u.ForceEmailChange
	actions := SecurityActionsResult{
		RequirePasswordChange: u.ForcePasswordChange || passwordExpired,
		RequireEmailChange:    requireEmail,
		RequireTwoFactorSetup: !u.TwoFactorEnabled && twoFactorForced(ctx, cs),
	}
	switch {
	case requireEmail && u.Username == "admin":
		actions.Reason = "default_admin_bootstrap"
	case requireEmail:
		actions.Reason = "first_login"
	case u.ForcePasswordChange:
		actions.Reason = "password_reset"
	case passwordExpired:
		actions.Reason = "password_expired"
	}
	return actions
}

// passwordExpiredFor 判断用户密码是否已超过策略规定的有效期。
// PasswordMaxAgeDays <= 0 表示永久有效。
func passwordExpiredFor(ctx context.Context, cs *sysconfig.Service, u *model.User) bool {
	if u.PasswordHash == "" {
		return false
	}
	sec, err := cs.Security(ctx)
	if err != nil || sec.PasswordMaxAgeDays <= 0 {
		return false
	}
	if u.PasswordChangedAt == nil {
		return true
	}
	return time.Since(*u.PasswordChangedAt) > time.Duration(sec.PasswordMaxAgeDays)*24*time.Hour
}

// twoFactorForced 读取系统安全策略中的“强制两步验证”开关。
func twoFactorForced(ctx context.Context, cs *sysconfig.Service) bool {
	sec, err := cs.Security(ctx)
	if err != nil {
		return false
	}
	return sec.TwoFactorRequired
}
