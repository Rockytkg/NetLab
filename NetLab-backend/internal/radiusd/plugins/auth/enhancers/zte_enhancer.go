package enhancers

import (
	"context"
	"math"

	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/vendors"
	"netlab-backend/internal/radiusd/vendors/zte"
)

type ZTEAcceptEnhancer struct{}

func NewZTEAcceptEnhancer() *ZTEAcceptEnhancer {
	return &ZTEAcceptEnhancer{}
}

func (e *ZTEAcceptEnhancer) Name() string {
	return "accept-zte"
}

func (e *ZTEAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeZTE) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Use getter methods for bandwidth rates
	upRate := user.GetUpRate()
	downRate := user.GetDownRate()

	up := clampInt64(int64(upRate)*1024, math.MaxInt32)
	down := clampInt64(int64(downRate)*1024, math.MaxInt32)

	_ = zte.ZTERateCtrlSCRUp_Set(resp, zte.ZTERateCtrlSCRUp(up))       //nolint:errcheck,gosec // G115: clamped to MaxInt32
	_ = zte.ZTERateCtrlSCRDown_Set(resp, zte.ZTERateCtrlSCRDown(down)) //nolint:errcheck,gosec // G115: clamped to MaxInt32
	return nil
}
