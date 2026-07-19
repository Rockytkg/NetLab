package checkers

import (
	"context"

	"netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/repository"
)

// OnlineCountChecker enforces online count limits
type OnlineCountChecker struct {
	sessionRepo repository.SessionRepository
}

// NewOnlineCountChecker creates an online count checker
func NewOnlineCountChecker(sessionRepo repository.SessionRepository) *OnlineCountChecker {
	return &OnlineCountChecker{sessionRepo: sessionRepo}
}

func (c *OnlineCountChecker) Name() string {
	return "online_count"
}

func (c *OnlineCountChecker) Order() int {
	return 30 // Execute after the bind check
}

func (c *OnlineCountChecker) Check(ctx context.Context, authCtx *auth.AuthContext) error {
	user := authCtx.User

	// An activeNum of 0 indicates no limit
	activeNum := user.GetActiveNum()
	if activeNum == 0 {
		return nil
	}

	count, err := c.sessionRepo.CountByUsername(ctx, user.Username)
	if err != nil {
		return err
	}

	if count >= activeNum {
		return errors.NewOnlineLimitError("user online count exceeded")
	}

	return nil
}
