package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

// Close implements port.CacheRepository.
func (r *Redis) Close() error {
	return r.client.Close()
}

// Delete implements port.CacheRepository.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// DeleteByPrefix implements port.CacheRepository.
func (r *Redis) DeleteByPrefix(ctx context.Context, prefix string) error {
	var cursor uint64

	for {
		keys, newCursor, err := r.client.Scan(ctx, cursor, prefix, 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning Redis keys with prefix '%s': %w", prefix, err)
		}

		for _, key := range keys {
			if err := r.client.Del(ctx, key).Err(); err != nil {
				return fmt.Errorf("error deleting key '%s': %w", key, err)
			}
		}

		if newCursor == 0 {
			break
		}

		cursor = newCursor
	}

	return nil
}

// Get implements port.CacheRepository.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	res, err := r.client.Get(ctx, key).Result()
	bytes := []byte(res)
	return bytes, err
}

// Set implements port.CacheRepository.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// New creates a new redis repository instance
func New(ctx context.Context, config *config.Redis) (port.CacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       0,
	})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &Redis{client: client}, nil
}
