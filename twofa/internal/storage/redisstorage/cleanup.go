package redisstorage

import "context"

// DeleteKeys removes one or more keys from Redis. Used for cleanup operations.
func (rs *RedisStorage) DeleteKeys(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rs.client.Del(ctx, keys...).Err()
}
