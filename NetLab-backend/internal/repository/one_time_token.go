package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"netlab-backend/pkg/crypto"
)

const oneTimeTokenPrefix = "ott:"

// StoreOneTimeToken stores a short-lived opaque token payload and returns the
// plaintext token that should be sent to the client. Redis keys contain only a
// hash of the token, so a key dump does not reveal usable credentials.
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

// ConsumeOneTimeToken atomically reads and deletes a token payload. It returns
// an empty slice when the token is missing or expired.
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

// PeekOneTimeToken reads a token payload without consuming it. Use only when a
// multi-step flow must allow retries; successful completion should consume it.
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

// DeleteOneTimeToken removes a token without reading its payload.
func (r *TokenRepository) DeleteOneTimeToken(ctx context.Context, namespace, token string) error {
	if token == "" {
		return nil
	}
	return r.redis.Del(ctx, r.oneTimeTokenKey(namespace, token)).Err()
}

func (r *TokenRepository) oneTimeTokenKey(namespace, token string) string {
	return oneTimeTokenPrefix + namespace + ":" + crypto.SHA256Base64URL(token)
}
