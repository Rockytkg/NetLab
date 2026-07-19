package checkers

import (
	"context"
	"strings"

	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
	vendorparsers "netlab-backend/internal/radiusd/plugins/vendorparsers"
)

// MacBindChecker enforces MAC binding
type MacBindChecker struct{}

func (c *MacBindChecker) Name() string {
	return "mac_bind"
}

func (c *MacBindChecker) Order() int {
	return 20 // Execute after status and expiration checks
}

func (c *MacBindChecker) Check(ctx context.Context, authCtx *auth.AuthContext) error {
	user := authCtx.User

	// Skip MAC bind check
	if !user.GetBindMac() {
		return nil
	}

	// Get MAC addresses from the vendor request
	vendorReq, ok := authCtx.VendorRequest.(*vendorparsers.VendorRequest)
	if !ok || vendorReq == nil {
		return nil
	}

	// e.g., if both the user MAC and request MAC are present, ensure they match;
	// user.MacAddr 可能是逗号分隔的多 MAC 列表，命中任一即视为绑定匹配。
	if isNotEmptyAndNA(user.MacAddr) && vendorReq.MacAddr != "" && !model.MacListContains(user.MacAddr, vendorReq.MacAddr) {
		return errors.NewMacBindError()
	}

	return nil
}

// isNotEmptyAndNA reports whether val is non-empty and not the "N/A" sentinel
// (mirrors radiusd.NA; kept local to avoid an import cycle).
func isNotEmptyAndNA(val string) bool {
	val = strings.TrimSpace(val)
	return val != "" && val != "N/A"
}
