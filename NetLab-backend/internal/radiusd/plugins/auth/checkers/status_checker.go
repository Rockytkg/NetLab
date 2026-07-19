package checkers

import (
	"context"

	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
)

// StatusChecker verifies user status
type StatusChecker struct{}

func (c *StatusChecker) Name() string {
	return "status"
}

func (c *StatusChecker) Order() int {
	return 5 // Execute early
}

func (c *StatusChecker) Check(ctx context.Context, authCtx *auth.AuthContext) error {
	user := authCtx.User

	if user.Status == model.RadiusUserStatusDisabled {
		return errors.NewUserDisabledError()
	}

	return nil
}
