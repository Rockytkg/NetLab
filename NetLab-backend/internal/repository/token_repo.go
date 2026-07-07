package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"netlab-backend/pkg/crypto"
)

// TokenRepository 管理 refresh token、JWT 黑名单、验证码
// 以及会话密钥——全部存储在 Redis 中并基于 TTL 过期。
// 无需 PostgreSQL 持久化：这些都是临时、会过期的数据。
type TokenRepository struct {
	redis *redis.Client
}

// NewTokenRepository 创建一个仅由 Redis 支撑的新 TokenRepository。
func NewTokenRepository(rdb *redis.Client) *TokenRepository {
	return &TokenRepository{redis: rdb}
}

// ── Refresh Token ──────────────────────────────────────────────────

// SaveRefreshToken 在 Redis 中存储一个有效的 refresh token。
func (r *TokenRepository) SaveRefreshToken(ctx context.Context, userID, tokenValue string, expiresAt time.Time) error {
	tokenHash := crypto.SHA256Hex(tokenValue)
	return r.redis.Set(ctx, "jwt:refresh:"+tokenHash, userID, time.Until(expiresAt)).Err()
}

// RevokeRefreshToken 从 Redis 中移除一个 refresh token（将其标记为不可用）。
func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, tokenValue string) error {
	tokenHash := crypto.SHA256Hex(tokenValue)
	return r.redis.Del(ctx, "jwt:refresh:"+tokenHash).Err()
}

// IsRefreshTokenRevoked 检查某个 refresh token 在 Redis 中是否仍然有效。
func (r *TokenRepository) IsRefreshTokenRevoked(ctx context.Context, tokenValue string) (bool, error) {
	tokenHash := crypto.SHA256Hex(tokenValue)
	exists, err := r.redis.Exists(ctx, "jwt:refresh:"+tokenHash).Result()
	if err != nil {
		return false, err
	}
	return exists == 0, nil // 不在 Redis 中 = 已撤销 / 已过期
}

// RevokeAllUserTokens 从 Redis 中移除某用户的所有 refresh token。
func (r *TokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	iter := r.redis.Scan(ctx, 0, "jwt:refresh:*", 0).Iterator()
	for iter.Next(ctx) {
		val, _ := r.redis.Get(ctx, iter.Val()).Result()
		if val == userID {
			r.redis.Del(ctx, iter.Val())
		}
	}
	return iter.Err()
}

// ── Refresh Token 重用检测 ───────────────────────────────────

// MarkRefreshTokenUsed 记录某个 refresh token 已被轮换。
// 用于重用检测：若出示一个已轮换的 token，则可能意味着令牌被盗。
func (r *TokenRepository) MarkRefreshTokenUsed(ctx context.Context, tokenValue string, ttl time.Duration) error {
	tokenHash := crypto.SHA256Hex(tokenValue)
	return r.redis.Set(ctx, "jwt:used:"+tokenHash, "1", ttl).Err()
}

// IsRefreshTokenReused 检查某个 refresh token 是否已被轮换。
func (r *TokenRepository) IsRefreshTokenReused(ctx context.Context, tokenValue string) (bool, error) {
	tokenHash := crypto.SHA256Hex(tokenValue)
	exists, err := r.redis.Exists(ctx, "jwt:used:"+tokenHash).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ── JWT 黑名单 ───────────────────────────────────────────────

// AddToBlacklist 将某个 JWT JTI 加入 Redis 中的黑名单。
func (r *TokenRepository) AddToBlacklist(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // 已过期，无需加入黑名单
	}
	return r.redis.Set(ctx, "jwt:blacklist:"+jti, "1", ttl).Err()
}

// IsBlacklisted 检查某个 JWT JTI 是否在黑名单中。
func (r *TokenRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := r.redis.Exists(ctx, "jwt:blacklist:"+jti).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// ── 验证码 ──────────────────────────────────────────────────

// StoreVerificationCode 将验证码带 TTL 存入 Redis。
func (r *TokenRepository) StoreVerificationCode(ctx context.Context, email, code, purpose string, ttl time.Duration) error {
	return r.redis.Set(ctx, "verify:code:"+email+":"+purpose, code, ttl).Err()
}

// GetVerificationCode 获取并删除验证码（一次性使用）。
func (r *TokenRepository) GetVerificationCode(ctx context.Context, email, purpose string) (string, error) {
	key := "verify:code:" + email + ":" + purpose
	code, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	r.redis.Del(ctx, key) // 一次性使用：读取后即删除
	return code, nil
}

// ── 验证码冷却 ───────────────────────────────────────────────

// SetVerificationCooldown 设置两次发送验证码之间的冷却时间。
func (r *TokenRepository) SetVerificationCooldown(ctx context.Context, email, purpose string, ttl time.Duration) error {
	return r.redis.Set(ctx, "verify:cooldown:"+email+":"+purpose, "1", ttl).Err()
}

// GetVerificationCooldown 返回剩余的冷却时长。
func (r *TokenRepository) GetVerificationCooldown(ctx context.Context, email, purpose string) (time.Duration, error) {
	return r.redis.TTL(ctx, "verify:cooldown:"+email+":"+purpose).Result()
}

// ── 会话密钥 ────────────────────────────────────────────────

// SaveSessionKeys 存储某用户会话的签名密钥。
func (r *TokenRepository) SaveSessionKeys(ctx context.Context, userID, signingKey string, ttl time.Duration) error {
	return r.redis.HSet(ctx, "session:keys:"+userID, map[string]interface{}{
		"signing_key": signingKey,
	}).Err()
}

// GetSessionKeys 获取会话签名密钥。
func (r *TokenRepository) GetSessionKeys(ctx context.Context, userID string) (signingKey string, err error) {
	result, err := r.redis.HGetAll(ctx, "session:keys:"+userID).Result()
	if err != nil {
		return "", err
	}
	return result["signing_key"], nil
}

// DeleteSessionKeys 删除某用户的会话密钥。
func (r *TokenRepository) DeleteSessionKeys(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, "session:keys:"+userID).Err()
}
