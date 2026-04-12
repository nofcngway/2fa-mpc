package redisstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// Key prefix constants for Redis session storage.
const (
	prefixRefreshToken = "refresh_token:"
	prefixTokenFamily  = "token_family:"
	prefixUserTokens   = "user_tokens:"
)

// StoreRefreshToken stores a refresh token with its metadata in Redis using a pipeline.
func (rs *RedisStorage) StoreRefreshToken(ctx context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
	data := &domain.RefreshTokenData{
		UserID:      userID,
		TokenFamily: tokenFamily,
		IssuedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal refresh token data: %w", err)
	}

	pipe := rs.client.Pipeline()

	// Store token data with TTL
	pipe.Set(ctx, prefixRefreshToken+jti, string(jsonData), ttl)

	// Add JTI to token family set
	pipe.SAdd(ctx, prefixTokenFamily+tokenFamily, jti)
	pipe.Expire(ctx, prefixTokenFamily+tokenFamily, ttl)

	// Add family to user tokens set and refresh TTL to cap stale entry growth (WR-02).
	// The TTL matches the refresh token TTL so orphaned families expire naturally.
	pipe.SAdd(ctx, prefixUserTokens+userID, tokenFamily)
	pipe.Expire(ctx, prefixUserTokens+userID, ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves refresh token data by JTI.
func (rs *RedisStorage) GetRefreshToken(ctx context.Context, jti string) (*domain.RefreshTokenData, error) {
	val, err := rs.client.Get(ctx, prefixRefreshToken+jti).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	var data domain.RefreshTokenData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal refresh token data: %w", err)
	}

	return &data, nil
}

// DeleteRefreshToken removes a single refresh token and cleans up family/user sets.
func (rs *RedisStorage) DeleteRefreshToken(ctx context.Context, jti string) error {
	// First get the token data to find its family and user
	data, err := rs.GetRefreshToken(ctx, jti)
	if err != nil {
		return fmt.Errorf("get token for deletion: %w", err)
	}
	if data == nil {
		// Token already deleted, idempotent
		return nil
	}

	pipe := rs.client.Pipeline()

	// Delete the token itself
	pipe.Del(ctx, prefixRefreshToken+jti)

	// Remove JTI from family set
	pipe.SRem(ctx, prefixTokenFamily+data.TokenFamily, jti)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	// Check if family set is now empty; if so, clean up
	count, err := rs.client.SCard(ctx, prefixTokenFamily+data.TokenFamily).Result()
	if err != nil {
		return fmt.Errorf("check family set size: %w", err)
	}

	if count == 0 {
		pipe2 := rs.client.Pipeline()
		pipe2.Del(ctx, prefixTokenFamily+data.TokenFamily)
		pipe2.SRem(ctx, prefixUserTokens+data.UserID, data.TokenFamily)
		_, err = pipe2.Exec(ctx)
		if err != nil {
			return fmt.Errorf("clean up empty family: %w", err)
		}
	}

	return nil
}

// DeleteTokenFamily removes all tokens in a token family and cleans up the user-tokens set.
func (rs *RedisStorage) DeleteTokenFamily(ctx context.Context, family, userID string) error {
	// Get all JTIs in the family
	jtis, err := rs.client.SMembers(ctx, prefixTokenFamily+family).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("get family members: %w", err)
	}

	pipe := rs.client.Pipeline()

	if len(jtis) == 0 {
		// Idempotent: family already empty, just delete the key
		pipe.Del(ctx, prefixTokenFamily+family)
	} else {
		// Delete each refresh token
		for _, jti := range jtis {
			pipe.Del(ctx, prefixRefreshToken+jti)
		}

		// Delete the family set itself
		pipe.Del(ctx, prefixTokenFamily+family)
	}

	// Remove the family from the user-tokens set to prevent stale references
	if userID != "" {
		pipe.SRem(ctx, prefixUserTokens+userID, family)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete token family: %w", err)
	}

	return nil
}

// deleteAllScript is a Lua script that atomically reads and deletes all tokens,
// families, and the user-tokens set for a given user. Running on the Redis server
// avoids TOCTOU races with concurrent token rotation (WR-01).
var deleteAllScript = redis.NewScript(`
	local families = redis.call('SMEMBERS', KEYS[1])
	for _, family in ipairs(families) do
		local jtis = redis.call('SMEMBERS', 'token_family:' .. family)
		for _, jti in ipairs(jtis) do
			redis.call('DEL', 'refresh_token:' .. jti)
		end
		redis.call('DEL', 'token_family:' .. family)
	end
	redis.call('DEL', KEYS[1])
	return 1
`)

// DeleteAllUserTokens removes all tokens and families for a user.
// Uses a Lua script for true atomicity — reads and deletes happen in a single
// server-side operation, preventing TOCTOU races with concurrent StoreRefreshToken.
func (rs *RedisStorage) DeleteAllUserTokens(ctx context.Context, userID string) error {
	err := deleteAllScript.Run(ctx, rs.client, []string{prefixUserTokens + userID}).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("delete all user tokens: %w", err)
	}
	return nil
}
