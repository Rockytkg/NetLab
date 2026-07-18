package rbac

import "testing"

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
