package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"netlab-backend/internal/model"
)

const portalSessionTTL = 24 * time.Hour

// SessionStore is the Redis runtime authority for Portal sessions.
type SessionStore struct{ redis *redis.Client }

// NewSessionStore creates a Redis-backed Portal session projection store.
func NewSessionStore(client *redis.Client) *SessionStore { return &SessionStore{redis: client} }

// Set stores a session for its fixed runtime TTL.
func (s *SessionStore) Set(ctx context.Context, session *model.PortalSession) error {
	if s == nil || s.redis == nil {
		return nil
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化Portal会话: %w", err)
	}
	return s.redis.Set(ctx, sessionKey(session.ClientIP), raw, portalSessionTTL).Err()
}

// Delete removes the runtime session projection for a client IP.
func (s *SessionStore) Delete(ctx context.Context, userIP string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, sessionKey(userIP)).Err()
}
func sessionKey(userIP string) string { return "portal:session:" + userIP }
