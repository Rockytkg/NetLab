package enhancers

import (
	"context"
	"fmt"

	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/vendors"
	"netlab-backend/internal/radiusd/vendors/mikrotik"
)

type MikrotikAcceptEnhancer struct{}

func NewMikrotikAcceptEnhancer() *MikrotikAcceptEnhancer {
	return &MikrotikAcceptEnhancer{}
}

func (e *MikrotikAcceptEnhancer) Name() string {
	return "accept-mikrotik"
}

func (e *MikrotikAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeMikrotik) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Use getter methods for bandwidth rates
	upRate := user.GetUpRate()
	downRate := user.GetDownRate()

	_ = mikrotik.MikrotikRateLimit_SetString(resp, fmt.Sprintf("%dk/%dk", upRate, downRate)) //nolint:errcheck
	return nil
}
