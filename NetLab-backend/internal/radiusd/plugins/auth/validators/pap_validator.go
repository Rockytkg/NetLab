package validators

import (
	"context"
	"strings"

	"layeh.com/radius/rfc2865"
	"netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
)

// PAPValidator handles PAP password validation
type PAPValidator struct{}

func (v *PAPValidator) Name() string {
	return "pap"
}

func (v *PAPValidator) CanHandle(authCtx *auth.AuthContext) bool {
	password := rfc2865.UserPassword_GetString(authCtx.Request.Packet)
	return strings.TrimSpace(password) != ""
}

func (v *PAPValidator) Validate(ctx context.Context, authCtx *auth.AuthContext, password string) error {
	requestPassword := rfc2865.UserPassword_GetString(authCtx.Request.Packet)

	if strings.TrimSpace(requestPassword) != password {
		return errors.NewPasswordMismatchError()
	}

	return nil
}
