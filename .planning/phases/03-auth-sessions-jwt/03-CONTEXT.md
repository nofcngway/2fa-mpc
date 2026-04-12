# Phase 3: Auth Sessions & JWT - Context

**Gathered:** 2026-04-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement login, JWT RS256 token issuance (access + refresh), refresh token rotation with theft detection via token families, logout (single + all sessions), and token validation for other services. Extends the Auth service from Phase 2 with session management backed by Redis.

</domain>

<decisions>
## Implementation Decisions

### JWT Token Structure
- **D-01:** Access token claims: `sub` (user_id UUID), `email`, `jti` (unique token ID), `iat`, `exp` (iat + 15min), `iss` ("mpc-2fa-auth")
- **D-02:** Refresh token is also a JWT with same claim structure plus `token_family` (UUID) claim. Expiry: iat + 7 days
- **D-03:** Both tokens signed with RS256 using keys from `config.yaml` (`jwt.private_key_path`, `jwt.public_key_path`)
- **D-04:** Token validation MUST use `jwt.WithValidMethods([]string{"RS256"})` to prevent algorithm confusion attacks (SEC-01)

### Refresh Token Storage (Redis)
- **D-05:** Three-key Redis model:
  - `refresh_token:{jti}` → Hash/JSON `{user_id, token_family, issued_at}` with 7d TTL
  - `token_family:{family_uuid}` → Set of JTIs with 7d TTL
  - `user_tokens:{user_id}` → Set of token_family UUIDs (no TTL, cleaned on last logout)
- **D-06:** On login: generate new `token_family` UUID, store refresh token, add family to user_tokens set
- **D-07:** On refresh: delete old JTI, issue new refresh token with SAME `token_family`, add new JTI to family set

### Theft Detection
- **D-08:** Token family approach — if a refresh token has valid JWT signature but JTI is NOT in Redis, it's a reused (stolen) token
- **D-09:** On theft detection: look up `token_family:{family}` → delete all JTIs in that family → remove family from `user_tokens:{user_id}`. Only the compromised family is revoked, not all user sessions.

### Logout
- **D-10:** Logout (single): delete `refresh_token:{jti}`, remove JTI from `token_family:{family}` set. If family set becomes empty, remove from `user_tokens:{user_id}`
- **D-11:** Logout-all: get all families from `user_tokens:{user_id}`, for each family delete all JTIs and the family set, then delete `user_tokens:{user_id}` entry
- **D-12:** Add `LogoutAll` RPC to proto definition alongside existing `Logout`

### Register Update
- **D-13:** Register now returns JWT tokens (auto-login) — update Register handler to issue access + refresh tokens after successful registration. RegisterResponse.tokens will be populated.

### SessionStorage Interface
- **D-14:** SessionStorage interface methods (implemented by redisstorage):
  - `StoreRefreshToken(ctx, jti, userID, tokenFamily string, ttl time.Duration) error`
  - `GetRefreshToken(ctx, jti string) (*RefreshTokenData, error)`
  - `DeleteRefreshToken(ctx, jti string) error`
  - `DeleteTokenFamily(ctx, family string) error`
  - `DeleteAllUserTokens(ctx, userID string) error`
- **D-15:** `RefreshTokenData` struct lives in `domain` or `models` package: `{UserID, TokenFamily, IssuedAt}`

### Security (SEC-01, SEC-03)
- **D-16:** Never populate `password_hash` field in proto User responses — handlers must omit it
- **D-17:** Never log JWT tokens, refresh tokens, or RSA private keys
- **D-18:** Access token validation returns user_id and email only (no sensitive data)

### Claude's Discretion
- JWT token generation/parsing helper function decomposition (jwt.go or similar)
- RSA key loading and caching strategy within AuthService
- Exact Redis command choices (SET vs HSET, JSON vs string encoding)
- Internal error types for token-related failures (ErrInvalidToken, ErrTokenExpired, ErrTokenRevoked, etc.)
- Test helper structure for JWT-related tests (key generation for tests, etc.)
- Login handler error messages (invalid credentials → Unauthenticated, not specifying which field is wrong)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` — Full project spec, security rules, JWT RS256 requirements, Redis usage patterns
- `workspace/02 - Services/Auth Service.md` — Auth service API and responsibilities

### Requirements
- `.planning/REQUIREMENTS.md` — AUTH-03, AUTH-04, AUTH-05, AUTH-06, AUTH-07, SEC-01, SEC-03

### Phase 2 Context (prior decisions)
- `.planning/phases/02-auth-registration/02-CONTEXT.md` — Registration decisions (D-06 about deferred tokens, now superseded by D-13)

### Existing Code
- `auth/internal/services/authService/auth_service.go` — AuthService struct, Storage + SessionStorage interfaces
- `auth/internal/services/authService/register.go` — Register method (will be updated to issue tokens)
- `auth/internal/api/auth_service_api/register.go` — Register handler (will populate tokens)
- `auth/internal/storage/redisstorage/redisstorage.go` — RedisStorage skeleton (Ping, Close only)
- `auth/internal/storage/pgstorage/user.go` — GetUserByEmail (needed for Login)
- `auth/internal/domain/errors.go` — Domain error definitions (will add auth/token errors)
- `auth/internal/models/models.go` — User model
- `auth/internal/bootstrap/bootstrap.go` — DI wiring (NewRedisStorage, NewAuthService)
- `auth/config/config.go` — JWTConfig with key paths and TTLs already defined
- `auth/config.yaml` — JWT config: private/public key paths, access_token_ttl=15m, refresh_token_ttl=168h

### Proto Definitions
- `auth/api/auth_api/auth_service.proto` — Login, RefreshToken, Logout, ValidateToken RPCs already defined. Need to add LogoutAll.
- `auth/api/models/models.proto` — TokenPair and User messages already defined

### Test Patterns (from Phase 2)
- `auth/internal/services/authService/register_test.go` — minimock + gotest.tools/v3/assert + suite pattern
- `auth/internal/services/authService/mocks/` — Generated mocks from Storage interface

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `config.yaml` already has JWT config (key paths, TTLs) — no config changes needed
- `config.go` already has `JWTConfig` struct with `PrivateKeyPath`, `PublicKeyPath`, `AccessTokenTTL`, `RefreshTokenTTL`
- `RedisStorage` skeleton exists with `New()`, `Ping()`, `Close()` — needs SessionStorage methods added
- Proto RPCs for Login, RefreshToken, Logout, ValidateToken already defined — just need handler implementations + LogoutAll RPC
- `TokenPair` proto message already defined with `access_token` and `refresh_token` fields
- Makefile has `generate-mocks` target — extend for new interfaces
- Test patterns from Phase 2 (minimock, gotest.tools, suite) ready to reuse

### Established Patterns
- Domain errors in `auth/internal/domain/errors.go` — add ErrInvalidCredentials, ErrTokenExpired, ErrTokenRevoked, etc.
- Handler → service → storage layering with interface-based DI
- gRPC error code mapping in handlers (InvalidArgument, Unauthenticated, Internal)
- One file per method in service and handler directories
- minimock for mock generation from interfaces

### Integration Points
- `AuthService` needs access to `SessionStorage` (already has the field, currently unused)
- `AuthService` needs JWT signing config (private key, public key, TTLs) — inject via constructor or config
- Register handler needs to call token generation after user creation
- `Makefile generate-mocks` needs updating to also generate SessionStorage mock

</code_context>

<specifics>
## Specific Ideas

- Login should return Unauthenticated for both "user not found" and "wrong password" — do not reveal which field is incorrect
- RSA keys should be loaded once at service startup (in bootstrap), not on every request
- Consider a `TokenService` or keeping JWT logic as methods on `AuthService` — Claude's discretion on decomposition

</specifics>

<deferred>
## Deferred Ideas

- **Logout-all from specific device** — only logout-all (all sessions) for now, per-device tracking not in scope
- **Token blacklisting** (access token revocation before expiry) — 15min TTL is short enough, no blacklist needed for academic project
- **Sliding window refresh** — refresh token TTL is fixed at 7 days from issuance, not sliding

</deferred>

---

*Phase: 03-auth-sessions-jwt*
*Context gathered: 2026-04-12*
