package radiusd

import (
	"sync"
	"time"

	radiuserrors "netlab-backend/internal/radiusd/errors"
)

// defaultAuthRateShards 是认证限流器的分片数量，必须为 2 的幂以便位掩码取片。
// 分片降低了全局锁竞争：不同用户仅在哈希到同一分片时才相互阻塞。
const defaultAuthRateShards = 256

// maxAuthRateShards 是分片数上限，保证 2 的幂取整不越界。
const maxAuthRateShards = 1 << 16

// authRateShard 是限流状态的一个独立加锁分区。
type authRateShard struct {
	mu    sync.Mutex
	users map[string]authRateUser
}

type authRateUser struct {
	Username  string
	Starttime time.Time
}

// authRateLimiter 实现按用户的认证频率限制（分片锁）。
type authRateLimiter struct {
	shards []*authRateShard
	mask   uint32
}

// newAuthRateLimiter 创建限流器，shardCount 向上取整为 2 的幂。
func newAuthRateLimiter(shardCount int) *authRateLimiter {
	if shardCount < 1 {
		shardCount = 1
	}
	n := 1
	for n < shardCount && n < maxAuthRateShards {
		n <<= 1
	}
	shards := make([]*authRateShard, n)
	for i := range shards {
		shards[i] = &authRateShard{users: make(map[string]authRateUser)}
	}
	return &authRateLimiter{shards: shards, mask: uint32(n - 1)} //nolint:gosec // n 有界
}

// fnv1a 计算 32 位 FNV-1a 哈希（热路径零分配）。
func fnv1a(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

func (l *authRateLimiter) shardFor(username string) *authRateShard {
	return l.shards[fnv1a(username)&l.mask]
}

// check 记录一次认证尝试；若上一次尝试仍在 interval 内则返回限流错误。
func (l *authRateLimiter) check(username string, interval time.Duration) error {
	sh := l.shardFor(username)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if val, ok := sh.users[username]; ok {
		if time.Now().Before(val.Starttime.Add(interval)) {
			return radiuserrors.NewOnlineLimitError("there is a authentication still in process")
		}
	}
	sh.users[username] = authRateUser{Username: username, Starttime: time.Now()}
	return nil
}

// release 解除用户的限流状态，允许立即再次认证。
func (l *authRateLimiter) release(username string) {
	sh := l.shardFor(username)
	sh.mu.Lock()
	delete(sh.users, username)
	sh.mu.Unlock()
}
