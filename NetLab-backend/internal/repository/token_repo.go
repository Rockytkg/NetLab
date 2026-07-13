package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"netlab-backend/pkg/crypto"
)

// TokenRepository 管理登录会话、refresh token 重用标记和验证码。
// 无需 PostgreSQL 持久化：这些都是临时、会过期的数据。
type TokenRepository struct {
	redis *redis.Client
}

const (
	sessionKeyPrefix     = "jwt:s:"
	usedRefreshKeyPrefix = "jwt:ru:"
)

const rotateSessionScript = `
if redis.call("HGET", KEYS[1], "s") ~= ARGV[1] then
	return 0
end
if redis.call("HGET", KEYS[1], "r") ~= ARGV[2] then
	return 0
end
local absolute_exp = tonumber(redis.call("HGET", KEYS[1], "z") or "0")
if absolute_exp <= tonumber(ARGV[5]) then
	return 0
end
redis.call("PSETEX", KEYS[2], ARGV[4], "1")
redis.call("HSET", KEYS[1],
	"r", ARGV[3])
redis.call("PEXPIRE", KEYS[1], ARGV[4])
return 1
`

// NewTokenRepository 创建一个仅由 Redis 支撑的新 TokenRepository。
func NewTokenRepository(rdb *redis.Client) *TokenRepository {
	return &TokenRepository{redis: rdb}
}

// ── Refresh Token ──────────────────────────────────────────────────

// SaveSession stores the only active login session for a user.
// It revokes the previous session in O(1) through the user -> session index.
func (r *TokenRepository) SaveSession(ctx context.Context, userID, sessionID, refreshToken string, refreshExp, absoluteExp time.Time) error {
	ttl := time.Until(refreshExp)
	if ttl <= 0 {
		return nil
	}
	refreshHash := crypto.SHA256Base64URL(refreshToken)
	if err := r.RevokeAllUserTokens(ctx, userID); err != nil {
		return err
	}
	key := r.sessionKey(userID)
	_, err := r.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, map[string]interface{}{
			"s": sessionID,
			"r": refreshHash,
			"z": absoluteExp.Unix(),
		})
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

// RotateSession atomically replaces the session's current refresh hash.
func (r *TokenRepository) RotateSession(ctx context.Context, userID, sessionID, oldRefreshToken, newRefreshToken string, refreshExp time.Time) error {
	ttl := time.Until(refreshExp)
	if ttl <= 0 {
		return nil
	}
	oldHash := crypto.SHA256Base64URL(oldRefreshToken)
	newHash := crypto.SHA256Base64URL(newRefreshToken)
	status, err := r.redis.Eval(ctx, rotateSessionScript, []string{
		r.sessionKey(userID),
		r.usedRefreshTokenKey(oldHash),
	}, sessionID, oldHash, newHash, int64(ttl/time.Millisecond), time.Now().Unix()).Int()
	if err == nil && status == 0 {
		return redis.Nil
	}
	return err
}

// IsRefreshTokenActive validates that the presented refresh token is the
// current token for the user's current session.
func (r *TokenRepository) IsRefreshTokenActive(ctx context.Context, userID, sessionID, tokenValue string) (bool, time.Time, error) {
	tokenHash := crypto.SHA256Base64URL(tokenValue)
	cmds, err := r.redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		key := r.sessionKey(userID)
		pipe.HGet(ctx, key, "s")
		pipe.HGet(ctx, key, "r")
		pipe.HGet(ctx, key, "z")
		return nil
	})
	if err != nil && err != redis.Nil {
		return false, time.Time{}, err
	}
	if len(cmds) != 3 {
		return false, time.Time{}, nil
	}
	storedSID, err1 := cmds[0].(*redis.StringCmd).Result()
	storedHash, err2 := cmds[1].(*redis.StringCmd).Result()
	absoluteExpRaw, err3 := cmds[2].(*redis.StringCmd).Result()
	if err1 == redis.Nil || err2 == redis.Nil || err3 == redis.Nil {
		return false, time.Time{}, nil
	}
	if err1 != nil || err2 != nil || err3 != nil {
		return false, time.Time{}, firstRedisErr(err1, err2, err3)
	}
	absoluteExpUnix, err := strconv.ParseInt(absoluteExpRaw, 10, 64)
	if err != nil {
		return false, time.Time{}, err
	}
	absoluteExp := time.Unix(absoluteExpUnix, 0)
	if !absoluteExp.After(time.Now()) {
		return false, absoluteExp, nil
	}
	return storedSID == sessionID && storedHash == tokenHash, absoluteExp, nil
}

// IsSessionActive validates the current access token's session.
func (r *TokenRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	storedSID, err := r.redis.HGet(ctx, r.sessionKey(userID), "s").Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return storedSID == sessionID, nil
}

// RevokeAllUserTokens revokes the user's current session without scanning Redis.
func (r *TokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return r.revokeSession(ctx, userID)
}

// ── Refresh Token 重用检测 ───────────────────────────────────

// IsRefreshTokenReused 检查某个 refresh token 是否已被轮换。
func (r *TokenRepository) IsRefreshTokenReused(ctx context.Context, tokenValue string) (bool, error) {
	tokenHash := crypto.SHA256Base64URL(tokenValue)
	exists, err := r.redis.Exists(ctx, r.usedRefreshTokenKey(tokenHash)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *TokenRepository) revokeSession(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, r.sessionKey(userID)).Err()
}

func (r *TokenRepository) sessionKey(userID string) string {
	return sessionKeyPrefix + userID
}

func (r *TokenRepository) usedRefreshTokenKey(tokenHash string) string {
	return usedRefreshKeyPrefix + tokenHash
}

func firstRedisErr(errs ...error) error {
	for _, err := range errs {
		if err != nil && err != redis.Nil {
			return err
		}
	}
	return nil
}

// ── 验证码 ──────────────────────────────────────────────────

// StoreVerificationCode 将验证码带 TTL 存入 Redis。
func (r *TokenRepository) StoreVerificationCode(ctx context.Context, email, code, purpose string, ttl time.Duration) error {
	return r.redis.Set(ctx, r.verificationCodeKey(email, purpose), code, ttl).Err()
}

func (r *TokenRepository) verificationCodeKey(email, purpose string) string {
	return "verify:code:" + email + ":" + purpose
}

// PeekVerificationCode 获取验证码但不删除。
func (r *TokenRepository) PeekVerificationCode(ctx context.Context, email, purpose string) (string, error) {
	code, err := r.redis.Get(ctx, r.verificationCodeKey(email, purpose)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return code, err
}

// GetVerificationCode 获取并删除验证码（一次性使用）。
func (r *TokenRepository) GetVerificationCode(ctx context.Context, email, purpose string) (string, error) {
	key := r.verificationCodeKey(email, purpose)
	code, err := r.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return code, err
}

// ── 验证码冷却 ───────────────────────────────────────────────

// SetVerificationCooldown 设置两次发送验证码之间的冷却时间。
func (r *TokenRepository) SetVerificationCooldown(ctx context.Context, email, purpose string, ttl time.Duration) error {
	return r.redis.Set(ctx, r.verificationCooldownKey(email, purpose), "1", ttl).Err()
}

// GetVerificationCooldown 返回剩余的冷却时长。
func (r *TokenRepository) GetVerificationCooldown(ctx context.Context, email, purpose string) (time.Duration, error) {
	return r.redis.TTL(ctx, r.verificationCooldownKey(email, purpose)).Result()
}

func (r *TokenRepository) verificationCooldownKey(email, purpose string) string {
	return "verify:cooldown:" + email + ":" + purpose
}

// ── 登录失败计数 / 临时锁定 ───────────────────────────────────────

func (r *TokenRepository) loginFailKey(userID string) string {
	return "auth:login:fail:" + userID
}

func (r *TokenRepository) loginLockKey(userID string) string {
	return "auth:login:lock:" + userID
}

// IncrementLoginFailure 增加 Redis 中的失败次数；达到阈值后写入锁定键。
func (r *TokenRepository) IncrementLoginFailure(ctx context.Context, userID string, maxAttempts int, lockDuration time.Duration) (int, bool, error) {
	key := r.loginFailKey(userID)
	attempts, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, false, err
	}
	if attempts == 1 {
		_ = r.redis.Expire(ctx, key, lockDuration).Err()
	}
	if int(attempts) >= maxAttempts {
		lockKey := r.loginLockKey(userID)
		if err := r.redis.Set(ctx, lockKey, fmt.Sprint(time.Now().Add(lockDuration).Unix()), lockDuration).Err(); err != nil {
			return int(attempts), false, err
		}
		return int(attempts), true, nil
	}
	return int(attempts), false, nil
}

func (r *TokenRepository) IsLoginLocked(ctx context.Context, userID string) (bool, time.Duration, error) {
	ttl, err := r.redis.TTL(ctx, r.loginLockKey(userID)).Result()
	if err != nil {
		return false, 0, err
	}
	return ttl > 0, ttl, nil
}

func (r *TokenRepository) ClearLoginFailures(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, r.loginFailKey(userID), r.loginLockKey(userID)).Err()
}

const twoFactorChallengeNamespace = "2fa:challenge"

// StoreTwoFactorChallenge 为用户生成一个短期有效的 2FA 登录挑战令牌，
// 并在 Redis 中保存 tokenHash→userID 映射（TTL 后自动过期）。
func (r *TokenRepository) StoreTwoFactorChallenge(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	return r.StoreOneTimeToken(ctx, twoFactorChallengeNamespace, []byte(userID), ttl)
}

// ResolveTwoFactorChallenge 校验挑战令牌并返回对应用户 ID（不删除，允许重试）。
// 令牌无效或过期时返回空字符串。
func (r *TokenRepository) ResolveTwoFactorChallenge(ctx context.Context, token string) (string, error) {
	raw, err := r.PeekOneTimeToken(ctx, twoFactorChallengeNamespace, token)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", nil
	}
	return string(raw), nil
}

// ConsumeTwoFactorChallenge 在 2FA 验证成功后删除挑战令牌，使其一次性有效。
func (r *TokenRepository) ConsumeTwoFactorChallenge(ctx context.Context, token string) {
	_ = r.DeleteOneTimeToken(ctx, twoFactorChallengeNamespace, token)
}

// ConsumeTwoFactorChallenge atomically consumes a valid challenge token.
func (r *TokenRepository) ConsumeTwoFactorChallengeValue(ctx context.Context, token string) (string, error) {
	raw, err := r.ConsumeOneTimeToken(ctx, twoFactorChallengeNamespace, token)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", nil
	}
	return string(raw), nil
}

// 2FA 绑定过程中的待确认密钥
// BeginSetup 生成的新密钥先暂存于 Redis，ConfirmSetup 校验动态码通过后才落库。

func (r *TokenRepository) twoFactorSetupKey(userID string) string {
	return "2fa:setup:" + userID
}

// StoreTwoFactorSetupSecret 暂存绑定流程中的 TOTP 密钥（明文，短期有效）。
func (r *TokenRepository) StoreTwoFactorSetupSecret(ctx context.Context, userID, secret string, ttl time.Duration) error {
	return r.redis.Set(ctx, r.twoFactorSetupKey(userID), secret, ttl).Err()
}

// GetTwoFactorSetupSecret 读取暂存的 TOTP 密钥（不删除，允许重试输入）。
func (r *TokenRepository) GetTwoFactorSetupSecret(ctx context.Context, userID string) (string, error) {
	secret, err := r.redis.Get(ctx, r.twoFactorSetupKey(userID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return secret, err
}

// DeleteTwoFactorSetupSecret 清除暂存的 TOTP 密钥。
func (r *TokenRepository) DeleteTwoFactorSetupSecret(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, r.twoFactorSetupKey(userID)).Err()
}

// 2FA 动态码尝试频率限制
// 与登录失败计数同理：达到阈值后临时锁定，防止暴力枚举 6 位动态码。

func (r *TokenRepository) twoFactorFailKey(userID string) string {
	return "auth:2fa:fail:" + userID
}

func (r *TokenRepository) twoFactorLockKey(userID string) string {
	return "auth:2fa:lock:" + userID
}

// IncrementTwoFactorFailure 增加 2FA 动态码失败次数；达到阈值后写入锁定键。
func (r *TokenRepository) IncrementTwoFactorFailure(ctx context.Context, userID string, maxAttempts int, lockDuration time.Duration) (int, bool, error) {
	key := r.twoFactorFailKey(userID)
	attempts, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, false, err
	}
	if attempts == 1 {
		_ = r.redis.Expire(ctx, key, lockDuration).Err()
	}
	if int(attempts) >= maxAttempts {
		lockKey := r.twoFactorLockKey(userID)
		if err := r.redis.Set(ctx, lockKey, fmt.Sprint(time.Now().Add(lockDuration).Unix()), lockDuration).Err(); err != nil {
			return int(attempts), false, err
		}
		return int(attempts), true, nil
	}
	return int(attempts), false, nil
}

// IsTwoFactorLocked 检查用户是否因 2FA 动态码失败过多而被临时锁定。
func (r *TokenRepository) IsTwoFactorLocked(ctx context.Context, userID string) (bool, time.Duration, error) {
	ttl, err := r.redis.TTL(ctx, r.twoFactorLockKey(userID)).Result()
	if err != nil {
		return false, 0, err
	}
	return ttl > 0, ttl, nil
}

// ClearTwoFactorFailures 清除 2FA 失败计数与锁定。
func (r *TokenRepository) ClearTwoFactorFailures(ctx context.Context, userID string) error {
	return r.redis.Del(ctx, r.twoFactorFailKey(userID), r.twoFactorLockKey(userID)).Err()
}
