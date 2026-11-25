// Package oauthstate stores one-time OAuth "state" values used for CSRF
// protection during the authorization-code flow.
package oauthstate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "oauth_state:"

// RedisStore keeps states in Redis with a TTL.
type RedisStore struct {
	client redis.Cmdable
}

func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client}
}

// Save stores state for ttl.
func (s *RedisStore) Save(ctx context.Context, state string, ttl time.Duration) error {
	if err := s.client.Set(ctx, keyPrefix+state, "1", ttl).Err(); err != nil {
		return fmt.Errorf("oauthstate: save: %w", err)
	}

	return nil
}

// Consume atomically deletes state and reports whether it existed, so a state
// can be used at most once.
func (s *RedisStore) Consume(ctx context.Context, state string) (bool, error) {
	err := s.client.GetDel(ctx, keyPrefix+state).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("oauthstate: consume: %w", err)
	}

	return true, nil
}
