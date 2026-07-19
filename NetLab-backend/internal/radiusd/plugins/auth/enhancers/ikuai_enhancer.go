package enhancers

import (
	"context"
	"math"

	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/vendors"
	"netlab-backend/internal/radiusd/vendors/ikuai"
)

type IkuaiAcceptEnhancer struct{}

func NewIkuaiAcceptEnhancer() *IkuaiAcceptEnhancer {
	return &IkuaiAcceptEnhancer{}
}

func (e *IkuaiAcceptEnhancer) Name() string {
	return "accept-ikuai"
}

func (e *IkuaiAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeIkuai) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Use getter methods for bandwidth rates
	upRate := user.GetUpRate()
	downRate := user.GetDownRate()

	up := clampInt64(int64(upRate)*1024*8, math.MaxInt32)
	down := clampInt64(int64(downRate)*1024*8, math.MaxInt32)

	_ = ikuai.RPUpstreamSpeedLimit_Set(resp, ikuai.RPUpstreamSpeedLimit(up))       //nolint:errcheck,gosec // G115: clamped to MaxInt32
	_ = ikuai.RPDownstreamSpeedLimit_Set(resp, ikuai.RPDownstreamSpeedLimit(down)) //nolint:errcheck,gosec // G115: clamped to MaxInt32
	return nil
}
