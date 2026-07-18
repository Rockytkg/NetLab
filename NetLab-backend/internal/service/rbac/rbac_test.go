package rbac

import (
	"testing"

	"netlab-backend/internal/permission"
)

func TestCatalogContainsOnlyRegisteredPermissions(t *testing.T) {
	seen := map[string]struct{}{}
	for _, p := range permission.Catalog {
		key := permissionKey(p.Resource, p.Action)
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate permission %q", key)
		}
		seen[key] = struct{}{}
	}
	for _, stale := range []string{"device.create", "device.read", "alert.create", "group.create", "dashboard.read", "audit_log.read", "auth.update"} {
		if _, ok := seen[stale]; ok {
			t.Fatalf("stale permission %q is still registered", stale)
		}
	}
}

func TestPermissionKey(t *testing.T) {
	if got := permissionKey("device", "read"); got != "device.read" {
		t.Fatalf("permission key = %q", got)
	}
}

func TestUniqueCodes(t *testing.T) {
	got := uniqueCodes([]string{"user.read", "user.read", "device.read"})
	if len(got) != 2 {
		t.Fatalf("unique codes length = %d", len(got))
	}
}
