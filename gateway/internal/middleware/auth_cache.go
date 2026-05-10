package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachedIdentity is the value cached per access token: the user_id and email
// returned by Auth.ValidateToken. Keeping it tiny avoids JSON parsing on the
// hot path — a single tab-separated string round-trips through Redis with one
// allocation each way.
type CachedIdentity struct {
	UserID string
	Email  string
}

// TokenCache memoizes successful Auth.ValidateToken results in Redis so the
// Gateway does not call the Auth service on every protected request.
//
// Security note:
//
//   - Cache TTL is short (default 10s) so the staleness window after logout is
//     small. Access tokens themselves carry a 15m exp claim, so an extra 10s
//     of post-logout validity does not change the threat model meaningfully.
//   - The cache key is the SHA-256 hash of the token (truncated). Tokens
//     never appear in Redis in plaintext, so a snapshot of the cache cannot
//     be used to forge requests.
type TokenCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewTokenCache returns a TokenCache backed by rdb. ttl bounds how long a
// validated identity stays cached (10–60s is sensible). Pass nil rdb to
// disable caching transparently — Get always misses, Set is a no-op.
func NewTokenCache(rdb *redis.Client, ttl time.Duration) *TokenCache {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	return &TokenCache{rdb: rdb, ttl: ttl}
}

// Get returns the cached identity for token, or (nil, false) on miss / disabled
// cache / Redis failure.
func (c *TokenCache) Get(ctx context.Context, token string) (*CachedIdentity, bool) {
	if c == nil || c.rdb == nil {
		return nil, false
	}
	val, err := c.rdb.Get(ctx, c.key(token)).Result()
	if err != nil {
		// errors.Is(err, redis.Nil) is the cache-miss path; other errors are
		// also treated as miss so the request falls back to Auth.ValidateToken
		// rather than failing.
		if !errors.Is(err, redis.Nil) {
			return nil, false
		}
		return nil, false
	}
	parts := strings.SplitN(val, "\t", 2)
	if len(parts) != 2 {
		return nil, false
	}
	return &CachedIdentity{UserID: parts[0], Email: parts[1]}, true
}

// Set stores id in the cache with the configured TTL. Best-effort — Redis
// failures are swallowed so they do not break the request flow.
func (c *TokenCache) Set(ctx context.Context, token string, id CachedIdentity) {
	if c == nil || c.rdb == nil {
		return
	}
	val := id.UserID + "\t" + id.Email
	_ = c.rdb.Set(ctx, c.key(token), val, c.ttl).Err()
}

// key produces a stable, length-bounded Redis key from a token. Truncating the
// SHA-256 to 16 bytes (32 hex chars) keeps keys short while leaving 128 bits
// of collision resistance — vastly more than required for short-lived caches.
func (c *TokenCache) key(token string) string {
	h := sha256.Sum256([]byte(token))
	return "token_cache:" + hex.EncodeToString(h[:16])
}
