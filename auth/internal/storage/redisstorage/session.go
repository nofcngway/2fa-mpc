package redisstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

// Key prefix constants for Redis session storage.
const (
	prefixRefreshToken = "refresh_token:"
	prefixTokenFamily  = "token_family:"
	prefixUserTokens   = "user_tokens:"
)

// StoreRefreshToken stores a refresh token with its metadata in Redis using a pipeline.
func (rs *RedisStorage) StoreRefreshToken(ctx context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
	data := &authService.RefreshTokenData{
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

	// Add family to user tokens set (no TTL per D-05)
	pipe.SAdd(ctx, prefixUserTokens+userID, tokenFamily)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves refresh token data by JTI.
func (rs *RedisStorage) GetRefreshToken(ctx context.Context, jti string) (*authService.RefreshTokenData, error) {
	val, err := rs.client.Get(ctx, prefixRefreshToken+jti).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	var data authService.RefreshTokenData
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

// DeleteAllUserTokens removes all tokens and families for a user.
// Uses MULTI/EXEC (TxPipelined) to avoid TOCTOU races with concurrent token rotation.
func (rs *RedisStorage) DeleteAllUserTokens(ctx context.Context, userID string) error {
	// Get all families for the user
	families, err := rs.client.SMembers(ctx, prefixUserTokens+userID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("get user families: %w", err)
	}

	_, err = rs.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, family := range families {
			jtis, err := rs.client.SMembers(ctx, prefixTokenFamily+family).Result()
			if err != nil && err != redis.Nil {
				return fmt.Errorf("get family %s members: %w", family, err)
			}

			for _, jti := range jtis {
				pipe.Del(ctx, prefixRefreshToken+jti)
			}

			pipe.Del(ctx, prefixTokenFamily+family)
		}

		// Delete the user tokens set
		pipe.Del(ctx, prefixUserTokens+userID)
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete all user tokens: %w", err)
	}

	return nil
}
