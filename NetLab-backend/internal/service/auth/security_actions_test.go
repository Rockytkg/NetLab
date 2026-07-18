package auth

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
)

func testConfigService(t *testing.T, maxAgeDays int) *sysconfig.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE nb_system_configs (
			id text PRIMARY KEY,
			key varchar(128) NOT NULL UNIQUE,
			value text NOT NULL,
			description varchar(512),
			updated_at datetime
		)
	`).Error; err != nil {
		t.Fatalf("create config table: %v", err)
	}
	cfg := sysconfig.NewService(repository.NewConfigRepository(db), nil)
	if err := cfg.SetSecurity(context.Background(), sysconfig.SecuritySettings{PasswordMaxAgeDays: maxAgeDays}); err != nil {
		t.Fatalf("set security: %v", err)
	}
	return cfg
}

func baseSecurityUser() *model.User {
	now := time.Now()
	return &model.User{
		ID:                1,
		Username:          "user",
		Email:             "user@example.com",
		PasswordHash:      "hash",
		Role:              model.UserRole("viewer"),
		PasswordChangedAt: &now,
	}
}

func TestComputeSecurityActionsForcedCases(t *testing.T) {
	ctx := context.Background()
	cfg := testConfigService(t, 0)

	resetUser := baseSecurityUser()
	resetUser.ForcePasswordChange = true
	got := computeSecurityActions(ctx, cfg, resetUser)
	if !got.RequirePasswordChange || got.RequireEmailChange || got.Reason != "password_reset" {
		t.Fatalf("password reset actions = %+v", got)
	}

	firstLoginUser := baseSecurityUser()
	firstLoginUser.ForcePasswordChange = true
	firstLoginUser.ForceEmailChange = true
	got = computeSecurityActions(ctx, cfg, firstLoginUser)
	if !got.RequirePasswordChange || !got.RequireEmailChange || got.Reason != "first_login" {
		t.Fatalf("first login actions = %+v", got)
	}

	admin := baseSecurityUser()
	admin.Username = "admin"
	admin.Role = model.RoleSuperAdmin
	admin.ForcePasswordChange = true
	admin.ForceEmailChange = true
	got = computeSecurityActions(ctx, cfg, admin)
	if !got.RequirePasswordChange || !got.RequireEmailChange || got.Reason != "default_admin_bootstrap" {
		t.Fatalf("admin bootstrap actions = %+v", got)
	}
}

func TestPasswordExpiryActions(t *testing.T) {
	ctx := context.Background()
	expiredCfg := testConfigService(t, 30)
	changedAt := time.Now().AddDate(0, 0, -31)
	user := baseSecurityUser()
	user.PasswordChangedAt = &changedAt

	got := computeSecurityActions(ctx, expiredCfg, user)
	if !got.RequirePasswordChange || got.RequireEmailChange || got.Reason != "password_expired" {
		t.Fatalf("expired password actions = %+v", got)
	}

	neverExpireCfg := testConfigService(t, 0)
	got = computeSecurityActions(ctx, neverExpireCfg, user)
	if got.RequirePasswordChange || got.RequireEmailChange || got.Reason != "" {
		t.Fatalf("max age 0 actions = %+v", got)
	}
}
