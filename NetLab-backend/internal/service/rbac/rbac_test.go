package rbac

import (
	"testing"

	"github.com/casbin/casbin/v3"
	casbinmodel "github.com/casbin/casbin/v3/model"
)

func TestWildcardRoleCanReadUsers(t *testing.T) {
	modelDef, err := casbinmodel.NewModelFromString(modelText)
	if err != nil {
		t.Fatalf("create casbin model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(modelDef)
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("1", "*", "*"); err != nil {
		t.Fatalf("add admin wildcard policy: %v", err)
	}
	if _, err := enforcer.AddPolicy("2", "*", "*"); err != nil {
		t.Fatalf("add super_admin wildcard policy: %v", err)
	}

	for _, role := range []string{"1", "2"} {
		allowed, err := enforcer.Enforce(role, "user", "read")
		if err != nil {
			t.Fatalf("enforce %s user:read: %v", role, err)
		}
		if !allowed {
			t.Fatalf("expected %s to be allowed user:read", role)
		}
	}
}
