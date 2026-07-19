package guards

import (
	"context"
	"sync"
	"time"

	"layeh.com/radius/rfc2865"

	"netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/auth"
)

const (
	defaultRejectLimit   = 7
	defaultResetWindow   = 1 * time.Second
	maxCachedRejectItems = 65535
)

// rejectItem stores rejection info for a single username
type rejectItem struct {
	mu         sync.Mutex
	rejects    int64
	lastReject time.Time
}

func (ri *rejectItem) exceeded(limit int64, window time.Duration) bool {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	if time.Since(ri.lastReject) > window {
		ri.rejects = 0
	}

	if ri.rejects > limit {
		return true
	}

	ri.rejects++
	ri.lastReject = time.Now()
	return false
}

// RejectDelayGuard blocks requests when consecutive rejection counts exceed the threshold
type RejectDelayGuard struct {
	maxRejectsFn func() int64
	windowFn     func() time.Duration

	mu    sync.RWMutex
	items map[string]*rejectItem
}

// NewRejectDelayGuard 创建 RejectDelayGuard。两个回调在每次错误处理时求值，
// 使管理端的配置修改无需重启即可生效；传 nil 使用默认值（7 次 / 1 秒窗口）。
func NewRejectDelayGuard(maxRejectsFn func() int64, windowFn func() time.Duration) *RejectDelayGuard {
	if maxRejectsFn == nil {
		maxRejectsFn = func() int64 { return defaultRejectLimit }
	}
	if windowFn == nil {
		windowFn = func() time.Duration { return defaultResetWindow }
	}
	return &RejectDelayGuard{
		maxRejectsFn: maxRejectsFn,
		windowFn:     windowFn,
		items:        make(map[string]*rejectItem),
	}
}

func (g *RejectDelayGuard) Name() string {
	return "reject-delay"
}

// OnError tracks rejection counts and returns a rate-limit error when the threshold is exceeded
func (g *RejectDelayGuard) OnError(ctx context.Context, authCtx *auth.AuthContext, stage string, err error) error {
	if err == nil {
		return nil
	}

	username := g.resolveUsername(authCtx)
	if username == "" {
		username = "anonymous"
	}

	item := g.getItem(username)
	if item.exceeded(g.currentMaxRejects(), g.currentResetWindow()) {
		return errors.NewAuthError(errors.MetricsRadiusRejectLimit, err.Error())
	}

	return nil
}

// OnAuthError implements the new Guard interface with GuardResult.
// It provides the same behavior as OnError but with more explicit control flow.
func (g *RejectDelayGuard) OnAuthError(ctx context.Context, authCtx *auth.AuthContext, stage string, err error) *auth.GuardResult {
	if err == nil {
		return &auth.GuardResult{Action: auth.GuardActionContinue}
	}

	username := g.resolveUsername(authCtx)
	if username == "" {
		username = "anonymous"
	}

	item := g.getItem(username)
	if item.exceeded(g.currentMaxRejects(), g.currentResetWindow()) {
		// Return rate limit error and stop processing other guards
		return &auth.GuardResult{
			Action: auth.GuardActionStop,
			Err:    errors.NewAuthError(errors.MetricsRadiusRejectLimit, err.Error()),
		}
	}

	// Continue to next guard, keeping original error
	return &auth.GuardResult{Action: auth.GuardActionContinue, Err: err}
}

func (g *RejectDelayGuard) currentMaxRejects() int64 {
	if n := g.maxRejectsFn(); n > 0 {
		return n
	}
	return defaultRejectLimit
}

func (g *RejectDelayGuard) currentResetWindow() time.Duration {
	if d := g.windowFn(); d > 0 {
		return d
	}
	return defaultResetWindow
}

func (g *RejectDelayGuard) resolveUsername(ctx *auth.AuthContext) string {
	if ctx == nil {
		return ""
	}
	if ctx.Metadata != nil {
		if v, ok := ctx.Metadata["username"].(string); ok && v != "" {
			return v
		}
	}
	if ctx.User != nil && ctx.User.Username != "" {
		return ctx.User.Username
	}
	if ctx.Request != nil {
		if v := rfc2865.UserName_GetString(ctx.Request.Packet); v != "" {
			return v
		}
	}
	return ""
}

func (g *RejectDelayGuard) getItem(username string) *rejectItem {
	g.mu.RLock()
	item, ok := g.items[username]
	g.mu.RUnlock()
	if ok {
		return item
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if item, ok = g.items[username]; ok {
		return item
	}

	if len(g.items) >= maxCachedRejectItems {
		g.items = make(map[string]*rejectItem)
	}

	item = &rejectItem{}
	g.items[username] = item
	return item
}
