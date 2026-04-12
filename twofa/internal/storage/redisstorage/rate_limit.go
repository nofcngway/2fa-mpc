package redisstorage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IncrementRateLimit atomically increments a rate limit counter and sets TTL on first increment.
func (rs *RedisStorage) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := rs.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incr rate limit: %w", err)
	}
	if count == 1 {
		rs.client.Expire(ctx, key, ttl)
	}
	return count, nil
}

// GetRateLimit returns the current rate limit counter value. Returns 0 if key does not exist.
func (rs *RedisStorage) GetRateLimit(ctx context.Context, key string) (int64, error) {
	count, err := rs.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get rate limit: %w", err)
	}
	return count, nil
}
