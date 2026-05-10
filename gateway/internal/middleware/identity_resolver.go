package middleware

import (
	"context"

	pb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
)

// IdentityResolver returns the user identity associated with a JWT access
// token. Auth middleware depends on this interface — not on the gRPC client
// nor the cache directly — so resolution strategy (cached / direct / mocked
// for tests) can be swapped at the composition root.
type IdentityResolver interface {
	Resolve(ctx context.Context, token string) (CachedIdentity, error)
}

// directResolver always calls Auth.ValidateToken — no cache. Useful when the
// gateway runs without Redis, in tests, or as the inner stage of a cached
// resolver.
type directResolver struct {
	client pb.AuthServiceClient
}

// NewDirectResolver returns a resolver that hits Auth.ValidateToken on every
// call. Suitable for setups without Redis or as a building block for
// cachedResolver.
func NewDirectResolver(client pb.AuthServiceClient) IdentityResolver {
	return &directResolver{client: client}
}

func (r *directResolver) Resolve(ctx context.Context, token string) (CachedIdentity, error) {
	resp, err := r.client.ValidateToken(ctx, &pb.ValidateTokenRequest{AccessToken: token})
	if err != nil {
		return CachedIdentity{}, err
	}
	return CachedIdentity{UserID: resp.UserId, Email: resp.Email}, nil
}

// cachedResolver tries TokenCache first and falls back to an inner resolver
// (typically a directResolver) on cache miss. Successful results are written
// back to the cache.
type cachedResolver struct {
	cache *TokenCache
	inner IdentityResolver
}

// NewCachedResolver wraps inner with a Redis-backed TokenCache. Pass cache=nil
// to disable caching transparently — Resolve degrades to inner.Resolve.
func NewCachedResolver(cache *TokenCache, inner IdentityResolver) IdentityResolver {
	return &cachedResolver{cache: cache, inner: inner}
}

func (r *cachedResolver) Resolve(ctx context.Context, token string) (CachedIdentity, error) {
	if cached, ok := r.cache.Get(ctx, token); ok {
		return *cached, nil
	}
	id, err := r.inner.Resolve(ctx, token)
	if err != nil {
		return CachedIdentity{}, err
	}
	r.cache.Set(ctx, token, id)
	return id, nil
}
