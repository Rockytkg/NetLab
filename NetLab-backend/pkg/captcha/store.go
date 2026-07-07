package captcha

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 使用 Redis 实现 Store 接口。
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建一个基于 Redis 的验证码存储。
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Set 存储一个带 TTL 的验证码答案。
func (s *RedisStore) Set(id, answer string, ttl time.Duration) error {
	ctx := context.Background()
	return s.client.Set(ctx, "captcha:"+id, answer, ttl).Err()
}

// Incr 原子地递增某个验证码的失败尝试计数并返回新计数。
// 该计数器通过 ttl 与验证码共享生命周期，因此会被自动清理。
// 用于强制执行每个验证码的重试限制。
func (s *RedisStore) Incr(id string, ttl time.Duration) (int64, error) {
	ctx := context.Background()
	key := "captcha:attempts:" + id
	n, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// 首次递增时设置过期时间，使该计数器不会比验证码存活更久。
	if n == 1 {
		_ = s.client.Expire(ctx, key, ttl).Err()
	}
	return n, nil
}

// Get 获取一个验证码答案。
func (s *RedisStore) Get(id string) (string, error) {
	ctx := context.Background()
	val, err := s.client.Get(ctx, "captcha:"+id).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Delete 从存储中移除一个验证码及其尝试计数器。
func (s *RedisStore) Delete(id string) error {
	ctx := context.Background()
	return s.client.Del(ctx, "captcha:"+id, "captcha:attempts:"+id).Err()
}
