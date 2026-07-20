package radiusd

import (
	"testing"
	"time"

	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/vendors"
)

func TestMatchBypassRuleMACScopeAndExpiry(t *testing.T) {
	now := time.Now()
	nasID := uint64(7)
	expired := now.Add(-time.Minute)

	ctx := &AuthPipelineContext{
		NAS:           &model.RadiusNas{ID: nasID},
		VendorRequest: &vendors.VendorRequest{MacAddr: "aa:bb:cc:dd:ee:ff"},
	}
	rules := []model.RadiusBypass{
		{Type: "ip", Value: "10.0.0.0/8", ProfileID: 1}, // Legacy IP rules never grant access.
		{Type: model.RadiusBypassTypeMac, Value: "aa:bb:cc:dd:ee:ff", ProfileID: 0},
		{Type: model.RadiusBypassTypeMac, Value: "aa:bb:cc:dd:ee:ff", ProfileID: 1, NasID: ptrUint64(8)},
		{Type: model.RadiusBypassTypeMac, Value: "aa:bb:cc:dd:ee:ff", ProfileID: 1, ExpireTime: &expired},
		{Type: model.RadiusBypassTypeMac, Value: "aa:bb:cc:dd:ee:ff", ProfileID: 1, NasID: &nasID},
	}

	rule, ok := matchBypassRule(rules, ctx)
	if !ok {
		t.Fatal("expected scoped, active MAC rule to match")
	}
	if rule.NasID == nil || *rule.NasID != nasID {
		t.Fatalf("matched unexpected rule: %+v", rule)
	}
}

func TestNormalizeAuthMAC(t *testing.T) {
	for _, raw := range []string{"aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF", "AABBCCDDEEFF"} {
		if got := normalizeAuthMAC(raw); got != "aa:bb:cc:dd:ee:ff" {
			t.Fatalf("normalizeAuthMAC(%q) = %q", raw, got)
		}
	}
	if got := normalizeAuthMAC("not-a-mac"); got != "" {
		t.Fatalf("invalid MAC normalized to %q", got)
	}
}

func TestBypassIPv4Match(t *testing.T) {
	if !bypassIPv4Match("192.0.2.10", []byte{192, 0, 2, 10}) {
		t.Fatal("expected exact IPv4 match")
	}
	if bypassIPv4Match("192.0.2.10", []byte{192, 0, 2, 11}) {
		t.Fatal("unexpected IPv4 match")
	}
	if bypassIPv4Match("192.0.2.0/24", []byte{192, 0, 2, 10}) {
		t.Fatal("CIDR must not be accepted as an IP bypass rule")
	}
}

func ptrUint64(v uint64) *uint64 { return &v }
