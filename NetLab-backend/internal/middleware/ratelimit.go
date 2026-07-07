package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/response"
)

// ─── 声明式限流配置 ────────────────────────────────────────────────
//
// RateLimitRule 定义了一条可应用于任意端点的限流规则。
// 使用示例：
//
//	// 登录接口：每个 IP 每分钟 5 次请求
//	POST /api/auth/login → RateLimitByIP(5, time.Minute)
//
//	// API 读取接口：每个 IP 每分钟 100 次请求
//	GET  /api/* → RateLimitByIP(100, time.Minute)
//
//	// 发送验证码接口：每个 IP 每分钟 3 次请求
//	POST /api/auth/send-code → RateLimitByIP(3, time.Minute)

// RateLimitRule 定义限流参数。
type RateLimitRule struct {
	MaxRequests int           // 时间窗口内的最大请求数
	Window      time.Duration // 时间窗口
	KeyPrefix   string        // Redis key 的前缀（例如 "auth"、"global"、"sensitive"）
	KeyFunc     func(c *gin.Context) string // 自定义 key 构建函数（默认：基于 IP）
}

// ─── 限流器 ─────────────────────────────────────────────────────────

// RateLimiter 实现了一个基于 Redis 的滑动窗口限流器。
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter 创建一个新的 RateLimiter。
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// ByRule 返回一个应用给定限流规则的 Gin 中间件。
// 这是用于声明式配置每个端点限流的主要 API：
//
//	router.POST("/api/auth/login", middleware.RateLimitByIP(5, time.Minute), handler.Login)
func (rl *RateLimiter) ByRule(rule RateLimitRule) gin.HandlerFunc {
	if rl.client == nil || rule.MaxRequests <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		// 构建限流 key
		key := rl.buildKey(c, rule)

		ctx := c.Request.Context()
		now := time.Now()
		windowStart := now.Add(-rule.Window)

		pipe := rl.client.Pipeline()

		// 移除过期条目（滑动窗口清理）
		pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))

		// 统计窗口内剩余的请求数
		countCmd := pipe.ZCard(ctx, key)

		// 添加当前请求
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(now.UnixMilli()),
			Member: fmt.Sprintf("%d", now.UnixNano()),
		})

		// 将 key 的过期时间设置为略长于窗口时长
		pipe.Expire(ctx, key, rule.Window*2)

		if _, err := pipe.Exec(ctx); err != nil {
			// 发生 Redis 错误时，放行请求（为保证可用性采用失败放行策略）
			c.Next()
			return
		}

		count, _ := countCmd.Result()
		remaining := rule.MaxRequests - int(count) - 1
		if remaining < 0 {
			remaining = 0
		}

		// 标准限流响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rule.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(rule.Window).Unix()))

		if int(count) >= rule.MaxRequests {
			response.Error(c, apperrors.ErrRateLimited)
			return
		}

		c.Next()
	}
}

// Global 返回一个适用于整个应用的全局限流器。
func (rl *RateLimiter) Global(maxRequests int, window time.Duration) gin.HandlerFunc {
	return rl.ByRule(RateLimitRule{
		MaxRequests: maxRequests,
		Window:      window,
		KeyPrefix:   "global",
	})
}

func (rl *RateLimiter) buildKey(c *gin.Context, rule RateLimitRule) string {
	if rule.KeyFunc != nil {
		return rule.KeyPrefix + ":" + rule.KeyFunc(c)
	}
	// 默认：按客户端 IP 限流
	return fmt.Sprintf("rate:%s:%s", rule.KeyPrefix, c.ClientIP())
}

// ─── 便捷构造函数 ──────────────────────────────────────────────────

// RateLimitByIP 创建一条按客户端 IP 地址限流的规则。
// 用法： r.POST("/login", RateLimitByIP(5, time.Minute), handler.Login)
func RateLimitByIP(max int, window time.Duration, prefix string) RateLimitRule {
	return RateLimitRule{
		MaxRequests: max,
		Window:      window,
		KeyPrefix:   prefix,
	}
}

// RateLimitByUser 创建一条按已认证用户 ID 限流的规则
//（若用户未认证，则回退到按 IP 限流）。
func RateLimitByUser(max int, window time.Duration, prefix string) RateLimitRule {
	return RateLimitRule{
		MaxRequests: max,
		Window:      window,
		KeyPrefix:   prefix,
		KeyFunc: func(c *gin.Context) string {
			if userID := GetUserID(c); userID != "" {
				return userID
			}
			return c.ClientIP()
		},
	}
}

// RateLimitByPathIP 创建一条按 路径 + IP 组合限流的规则。
// 这会自动为每个端点隔离限流额度。
func RateLimitByPathIP(max int, window time.Duration, prefix string) RateLimitRule {
	return RateLimitRule{
		MaxRequests: max,
		Window:      window,
		KeyPrefix:   prefix,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP() + ":" + c.FullPath()
		},
	}
}

// ─── 预配置规则 ───────────────────────────────────────────────────

// 针对不同端点类别的常用限流预设。
var (
	// StrictLimit：每分钟 5 次 —— 用于敏感操作（登录、重置密码）
	StrictLimit = func(prefix string) RateLimitRule {
		return RateLimitByIP(5, time.Minute, prefix)
	}
	// ModerateLimit：每分钟 15 次 —— 用于半敏感操作（注册、发送验证码）
	ModerateLimit = func(prefix string) RateLimitRule {
		return RateLimitByIP(15, time.Minute, prefix)
	}
	// StandardLimit：每分钟 60 次 —— 用于需认证的读取端点
	StandardLimit = func(prefix string) RateLimitRule {
		return RateLimitByIP(60, time.Minute, prefix)
	}
	// HighLimit：每分钟 300 次 —— 用于高吞吐量端点
	HighLimit = func(prefix string) RateLimitRule {
		return RateLimitByIP(300, time.Minute, prefix)
	}
)

// ─── 独立限流器（无需中间件，可从 handler 中调用）───

// TryAcquire 尝试获取一个限流令牌。返回 (是否允许, 剩余次数, 错误)。
// 适用于在 handler 或 service 内部以编程方式进行限流。
func (rl *RateLimiter) TryAcquire(ctx context.Context, key string, max int, window time.Duration) (bool, int, error) {
	if rl.client == nil {
		return true, max, nil
	}

	now := time.Now()
	windowStart := now.Add(-window)

	pipe := rl.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})
	pipe.Expire(ctx, key, window*2)

	if _, err := pipe.Exec(ctx); err != nil {
		return true, 0, err
	}

	count, _ := countCmd.Result()
	remaining := max - int(count) - 1
	if remaining < 0 {
		remaining = 0
	}

	return int(count) < max, remaining, nil
}
