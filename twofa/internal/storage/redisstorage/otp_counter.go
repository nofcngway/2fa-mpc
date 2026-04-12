package redisstorage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SetUsedOTPCounter stores the last used OTP counter for a user with TTL for reuse prevention.
func (rs *RedisStorage) SetUsedOTPCounter(ctx context.Context, userID string, counter int64, ttl time.Duration) error {
	key := fmt.Sprintf("otp_used:%s", userID)
	return rs.client.Set(ctx, key, counter, ttl).Err()
}

// GetUsedOTPCounter retrieves the last used OTP counter for a user. Returns 0 if not found.
func (rs *RedisStorage) GetUsedOTPCounter(ctx context.Context, userID string) (int64, error) {
	key := fmt.Sprintf("otp_used:%s", userID)
	val, err := rs.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get used otp counter: %w", err)
	}
	return val, nil
}
