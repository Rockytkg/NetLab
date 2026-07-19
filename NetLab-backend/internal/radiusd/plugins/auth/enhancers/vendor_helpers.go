package enhancers

import (
	"strings"

	"netlab-backend/internal/radiusd/plugins/auth"
)

func matchVendor(ctx *auth.AuthContext, vendorCode string) bool {
	if ctx == nil || ctx.Nas == nil {
		return false
	}
	return ctx.Nas.VendorCode == vendorCode
}

func clampInt64(val int64, max int64) int64 {
	if val > max {
		return max
	}
	return val
}

// isNotEmptyAndNA reports whether val is non-empty and not the "N/A" sentinel
// (mirrors radiusd.NA; kept local to avoid an import cycle).
func isNotEmptyAndNA(val string) bool {
	val = strings.TrimSpace(val)
	return val != "" && val != "N/A"
}
