package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"netlab-backend/pkg/crypto"
)

const oneTimeTokenPrefix = "ott:"

// StoreOneTimeToken 存储一个短时效的不透明 token 载荷，并返回应发送给
// 客户端的明文 token。Redis 键只包含 token 的哈希，因此键转储不会泄露
// 可用凭据。
func (r *TokenRepository) StoreOneTimeToken(ctx context.Context, namespace string, payload []byte, ttl time.Duration) (string, error) {
	token, err := crypto.GenerateRandomBase64URL(32)
	if err != nil {
		return "", err
	}
	if err := r.redis.Set(ctx, r.oneTimeTokenKey(namespace, token), payload, ttl).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// ConsumeOneTimeToken 原子地读取并删除 token 载荷。
// token 不存在或已过期时返回空切片。
func (r *TokenRepository) ConsumeOneTimeToken(ctx context.Context, namespace, token string) ([]byte, error) {
	if token == "" {
		return nil, nil
	}
	raw, err := r.redis.GetDel(ctx, r.oneTimeTokenKey(namespace, token)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return raw, err
}

// PeekOneTimeToken 读取 token 载荷但不消费。仅在多步流程必须允许重试时
// 使用；流程成功完成后应显式消费该 token。
func (r *TokenRepository) PeekOneTimeToken(ctx context.Context, namespace, token string) ([]byte, error) {
	if token == "" {
		return nil, nil
	}
	raw, err := r.redis.Get(ctx, r.oneTimeTokenKey(namespace, token)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return raw, err
}

// DeleteOneTimeToken 删除 token 而不读取其载荷。
func (r *TokenRepository) DeleteOneTimeToken(ctx context.Context, namespace, token string) error {
	if token == "" {
		return nil
	}
	return r.redis.Del(ctx, r.oneTimeTokenKey(namespace, token)).Err()
}

func (r *TokenRepository) oneTimeTokenKey(namespace, token string) string {
	return oneTimeTokenPrefix + namespace + ":" + crypto.SHA256Base64URL(token)
}
