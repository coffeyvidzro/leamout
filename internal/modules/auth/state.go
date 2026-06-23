package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalidOAuthState = errors.New("invalid oauth state")

type StateStore interface {
	SaveOAuthState(ctx context.Context, provider, state string, ttl time.Duration) error
	ConsumeOAuthState(ctx context.Context, provider, state string) error
}

type RedisStateStore struct {
	client *redis.Client
}

func NewRedisStateStore(client *redis.Client) *RedisStateStore {
	return &RedisStateStore{client: client}
}

func (s *RedisStateStore) SaveOAuthState(ctx context.Context, provider, state string, ttl time.Duration) error {
	if err := s.client.Set(ctx, oauthStateKey(provider, state), "1", ttl).Err(); err != nil {
		return fmt.Errorf("save oauth state: %w", err)
	}

	return nil
}

func (s *RedisStateStore) ConsumeOAuthState(ctx context.Context, provider, state string) error {
	deleted, err := s.client.Del(ctx, oauthStateKey(provider, state)).Result()
	if err != nil {
		return fmt.Errorf("consume oauth state: %w", err)
	}
	if deleted == 0 {
		return ErrInvalidOAuthState
	}

	return nil
}

func oauthStateKey(provider, state string) string {
	return "oauth_state:" + provider + ":" + state
}
