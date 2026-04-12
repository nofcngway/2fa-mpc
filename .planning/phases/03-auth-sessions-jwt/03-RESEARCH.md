# Phase 3: Auth Sessions & JWT - Research

**Researched:** 2026-04-12
**Domain:** JWT RS256 token management, Redis session storage, refresh token rotation with theft detection
**Confidence:** HIGH

## Summary

This phase adds login, JWT RS256 token issuance, refresh token rotation with token-family-based theft detection, logout (single + all), and token validation to the existing Auth service. The codebase already has strong foundations: `AuthService` struct with `SessionStorage` interface placeholder, `JWTConfig` in config with key paths and TTLs, `RedisStorage` skeleton, proto RPCs for Login/RefreshToken/Logout/ValidateToken, and `TokenPair` message defined. The main work is implementing the JWT helper (sign/verify with RS256), Redis session storage methods, service-layer business logic for each operation, and gRPC handlers.

The key libraries are `github.com/golang-jwt/jwt/v5` (v5.3.1) for JWT operations and the already-installed `github.com/redis/go-redis/v9` (v9.18.0) for Redis. Both are mature, well-documented, and the project's `go.mod` already pins go-redis. RSA key loading happens once at startup in bootstrap and is injected into `AuthService`.

**Primary recommendation:** Follow the existing clean architecture pattern (handler -> service -> storage), add `golang-jwt/jwt/v5`, implement `SessionStorage` interface methods in `redisstorage`, create a `jwt.go` helper file in `authService`, and implement one handler file per RPC method.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Access token claims: `sub` (user_id UUID), `email`, `jti` (unique token ID), `iat`, `exp` (iat + 15min), `iss` ("mpc-2fa-auth")
- **D-02:** Refresh token is also a JWT with same claim structure plus `token_family` (UUID) claim. Expiry: iat + 7 days
- **D-03:** Both tokens signed with RS256 using keys from `config.yaml` (`jwt.private_key_path`, `jwt.public_key_path`)
- **D-04:** Token validation MUST use `jwt.WithValidMethods([]string{"RS256"})` to prevent algorithm confusion attacks (SEC-01)
- **D-05:** Three-key Redis model: `refresh_token:{jti}`, `token_family:{family_uuid}`, `user_tokens:{user_id}`
- **D-06:** On login: generate new `token_family` UUID, store refresh token, add family to user_tokens set
- **D-07:** On refresh: delete old JTI, issue new refresh token with SAME `token_family`, add new JTI to family set
- **D-08:** Token family approach -- if refresh token has valid JWT signature but JTI is NOT in Redis, it's a reused (stolen) token
- **D-09:** On theft detection: revoke only the compromised family, not all user sessions
- **D-10:** Logout (single): delete `refresh_token:{jti}`, remove JTI from `token_family:{family}` set
- **D-11:** Logout-all: get all families from `user_tokens:{user_id}`, delete all JTIs and family sets
- **D-12:** Add `LogoutAll` RPC to proto definition
- **D-13:** Register now returns JWT tokens (auto-login after registration)
- **D-14:** SessionStorage interface with 5 methods
- **D-15:** `RefreshTokenData` struct in domain/models package
- **D-16:** Never populate `password_hash` field in proto User responses
- **D-17:** Never log JWT tokens, refresh tokens, or RSA private keys
- **D-18:** Access token validation returns user_id and email only

### Claude's Discretion
- JWT token generation/parsing helper function decomposition (jwt.go or similar)
- RSA key loading and caching strategy within AuthService
- Exact Redis command choices (SET vs HSET, JSON vs string encoding)
- Internal error types for token-related failures
- Test helper structure for JWT-related tests
- Login handler error messages (generic Unauthenticated for invalid credentials)

### Deferred Ideas (OUT OF SCOPE)
- Logout-all from specific device (per-device tracking)
- Token blacklisting (access token revocation before expiry)
- Sliding window refresh (refresh token TTL is fixed at 7 days from issuance)

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-03 | User can login and receive JWT access token (RS256, 15min) and refresh token (7 days, stored in Redis) | golang-jwt RS256 signing, Redis HSET for refresh token storage, bcrypt compare for password verification |
| AUTH-04 | User can refresh access token via refresh token with rotation (old token deleted, new issued) | Token family pattern in Redis, JWT parse + re-sign, atomic Redis operations |
| AUTH-05 | Refresh token reuse detected -- revoke all tokens for user (theft detection) | Token family revocation via Redis SMEMBERS + DEL pipeline |
| AUTH-06 | User can logout (refresh token deleted from Redis, session invalidated) | Redis DEL + SREM operations for single logout, pipeline for logout-all |
| AUTH-07 | Access token can be validated by other services (returns user_id and claims) | JWT parse with `WithValidMethods([]string{"RS256"})`, public key only needed |
| SEC-01 | JWT validation uses WithValidMethods to prevent algorithm confusion | `jwt.WithValidMethods` parser option in golang-jwt/jwt/v5 |
| SEC-03 | Passwords never returned in responses or logged | Handler must omit `password_hash` from proto User, slog must never include password fields |

</phase_requirements>

## Project Constraints (from CLAUDE.md)

Key directives affecting this phase:
- **JWT**: RS256, access 15 min, refresh 7 days (stored in Redis with TTL)
- **DB**: pgx directly, NO ORM
- **Logging**: slog structured, NEVER log secrets, passwords, tokens, keys
- **Errors**: gRPC status codes (InvalidArgument, NotFound, Unauthenticated, AlreadyExists, Internal)
- **Architecture**: handler -> service -> repository, DI via bootstrap
- **HTTP**: ONLY in Gateway, Auth service is gRPC only
- **Go deps**: `github.com/golang-jwt/jwt/v5` is approved
- **Config**: config.yaml loaded in config/config.go (JWT key paths and TTLs already defined)

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/golang-jwt/jwt/v5 | v5.3.1 | JWT creation, signing (RS256), parsing, validation | Official successor to dgrijalva/jwt-go, most used Go JWT library [VERIFIED: Go proxy] |
| github.com/redis/go-redis/v9 | v9.18.0 | Redis session storage for refresh tokens | Already in go.mod, official Redis client for Go [VERIFIED: go.mod] |
| golang.org/x/crypto | v0.50.0 | bcrypt for password comparison during login | Already in go.mod [VERIFIED: go.mod] |
| github.com/google/uuid | v1.6.0 | UUID generation for JTI and token_family | Already in go.mod [VERIFIED: go.mod] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/gojuno/minimock/v3 | v3.4.7 | Mock generation for SessionStorage interface | Test time only [VERIFIED: go.mod] |
| gotest.tools/v3 | v3.5.2 | Test assertions | Test time only [VERIFIED: go.mod] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| golang-jwt/jwt/v5 | lestrrat-go/jwx | jwx is more comprehensive but heavier; golang-jwt is simpler for RS256-only use case and is the project-approved dependency |

**Installation:**
```bash
cd auth && go get github.com/golang-jwt/jwt/v5@v5.3.1
```

## Architecture Patterns

### Files to Create/Modify

```
auth/
├── internal/
│   ├── api/auth_service_api/
│   │   ├── register.go        # MODIFY: populate tokens after registration (D-13)
│   │   ├── login.go            # CREATE: Login handler
│   │   ├── refresh_token.go    # CREATE: RefreshToken handler
│   │   ├── logout.go           # CREATE: Logout handler
│   │   ├── logout_all.go       # CREATE: LogoutAll handler
│   │   └── validate_token.go   # CREATE: ValidateToken handler
│   ├── services/authService/
│   │   ├── auth_service.go     # MODIFY: add JWT config fields, update SessionStorage interface
│   │   ├── register.go         # MODIFY: return token pair after user creation
│   │   ├── jwt.go              # CREATE: RSA key loading, token generation, token parsing
│   │   ├── login.go            # CREATE: Login business logic
│   │   ├── refresh_token.go    # CREATE: Refresh with rotation + theft detection
│   │   ├── logout.go           # CREATE: Single logout
│   │   ├── logout_all.go       # CREATE: All sessions logout
│   │   ├── validate_token.go   # CREATE: Access token validation
│   │   └── mocks/
│   │       ├── storage_mock.go          # EXISTS
│   │       └── session_storage_mock.go  # CREATE: via minimock generate
│   ├── domain/
│   │   └── errors.go           # MODIFY: add JWT/auth error types
│   ├── storage/redisstorage/
│   │   ├── redisstorage.go     # EXISTS (Ping, Close)
│   │   └── session.go          # CREATE: SessionStorage implementation
│   └── bootstrap/
│       └── bootstrap.go        # MODIFY: load RSA keys, pass JWT config to AuthService
├── api/auth_api/
│   └── auth_service.proto      # MODIFY: add LogoutAll RPC
└── keys/                       # CREATE: directory for RSA key pair (dev only)
    ├── private.pem             # CREATE: dev RSA private key
    └── public.pem              # CREATE: dev RSA public key
```

### Pattern 1: JWT Helper (jwt.go)

**What:** Centralized JWT operations as methods on AuthService or standalone functions
**When to use:** All token creation and validation flows

```go
// Source: golang-jwt/jwt/v5 documentation [VERIFIED: Go proxy v5.3.1]
package authService

import (
    "crypto/rsa"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// Claims defines the JWT claims structure for both access and refresh tokens.
type Claims struct {
    jwt.RegisteredClaims
    Email       string `json:"email"`
    TokenFamily string `json:"token_family,omitempty"` // refresh tokens only
}

// GenerateAccessToken creates a signed RS256 access token.
func (s *AuthService) GenerateAccessToken(userID, email string) (string, string, error) {
    jti := uuid.New().String()
    now := time.Now()
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID,
            ID:        jti,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
            Issuer:    "mpc-2fa-auth",
        },
        Email: email,
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    signed, err := token.SignedString(s.privateKey)
    return signed, jti, err
}

// ParseToken validates a JWT token string with RS256 algorithm enforcement.
func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{},
        func(t *jwt.Token) (interface{}, error) {
            return s.publicKey, nil
        },
        jwt.WithValidMethods([]string{"RS256"}), // SEC-01: algorithm confusion prevention
    )
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok {
        return nil, ErrInvalidToken
    }
    return claims, nil
}
```

### Pattern 2: Redis Session Storage (Three-Key Model)

**What:** Redis-backed session storage implementing the decided three-key model (D-05)
**When to use:** All refresh token lifecycle operations

```go
// Source: go-redis/v9 documentation [VERIFIED: go.mod v9.18.0]
package redisstorage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// RefreshTokenData holds metadata stored alongside each refresh token JTI in Redis.
type RefreshTokenData struct {
    UserID      string `json:"user_id"`
    TokenFamily string `json:"token_family"`
    IssuedAt    string `json:"issued_at"`
}

// StoreRefreshToken stores refresh token metadata with TTL and updates family/user sets.
func (rs *RedisStorage) StoreRefreshToken(ctx context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
    data, _ := json.Marshal(RefreshTokenData{
        UserID:      userID,
        TokenFamily: tokenFamily,
        IssuedAt:    time.Now().Format(time.RFC3339),
    })

    pipe := rs.client.Pipeline()
    pipe.Set(ctx, fmt.Sprintf("refresh_token:%s", jti), data, ttl)
    pipe.SAdd(ctx, fmt.Sprintf("token_family:%s", tokenFamily), jti)
    pipe.Expire(ctx, fmt.Sprintf("token_family:%s", tokenFamily), ttl)
    pipe.SAdd(ctx, fmt.Sprintf("user_tokens:%s", userID), tokenFamily)
    _, err := pipe.Exec(ctx)
    return err
}
```

### Pattern 3: Theft Detection Flow

**What:** Token family-scoped revocation when reused refresh token is detected
**When to use:** During RefreshToken when JTI not found but JWT signature is valid

```go
// RefreshToken handles token rotation with theft detection (D-07, D-08, D-09).
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (accessToken, newRefreshToken string, err error) {
    // 1. Parse refresh JWT (validates signature + expiry + algorithm)
    claims, err := s.ParseToken(refreshTokenStr)
    if err != nil {
        return "", "", ErrInvalidToken
    }

    // 2. Look up JTI in Redis
    tokenData, err := s.sessionStorage.GetRefreshToken(ctx, claims.ID)
    if err != nil {
        // JTI not in Redis but JWT is valid => THEFT DETECTED (D-08)
        if claims.TokenFamily != "" {
            // Revoke entire token family (D-09)
            _ = s.sessionStorage.DeleteTokenFamily(ctx, claims.TokenFamily)
        }
        return "", "", ErrTokenRevoked
    }

    // 3. Delete old JTI, issue new tokens with same family (D-07)
    _ = s.sessionStorage.DeleteRefreshToken(ctx, claims.ID)

    accessToken, _, err = s.GenerateAccessToken(tokenData.UserID, claims.Email)
    if err != nil {
        return "", "", err
    }

    newRefreshToken, err = s.generateAndStoreRefreshToken(ctx, tokenData.UserID, claims.Email, tokenData.TokenFamily)
    return accessToken, newRefreshToken, err
}
```

### Anti-Patterns to Avoid
- **Loading RSA keys per request:** Load once in bootstrap, inject into AuthService. File I/O on every token operation is wasteful and error-prone.
- **Storing full JWT string in Redis:** Only store metadata (user_id, family, issued_at). The JTI is the key. Storing full tokens wastes memory and is a security risk.
- **Returning specific login errors:** "user not found" vs "wrong password" leaks information. Always return generic `Unauthenticated`.
- **Using `jwt.Parse` without `WithValidMethods`:** Algorithm confusion attack vector. Always enforce RS256.
- **Non-atomic Redis operations:** Use pipelines for multi-key operations (store token + update family + update user set).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JWT signing/verification | Custom RSA signing code | golang-jwt/jwt/v5 | Handles claims validation, expiry checks, algorithm enforcement, key format parsing |
| UUID generation | Custom random ID | github.com/google/uuid (v4) | Cryptographically random, RFC 4122 compliant, already in project |
| Password comparison | Manual bcrypt | golang.org/x/crypto/bcrypt | Timing-safe comparison built in |
| Redis pipelining | Manual multi-command sequences | go-redis Pipeline() | Atomic execution, single round trip |

## Common Pitfalls

### Pitfall 1: RSA Key Format Mismatch
**What goes wrong:** `jwt.ParseRSAPrivateKeyFromPEM` fails silently or with cryptic errors when key format is wrong (PKCS#1 vs PKCS#8).
**Why it happens:** OpenSSL generates different formats depending on command used.
**How to avoid:** Use `openssl genrsa -out private.pem 2048` for PKCS#1 (what golang-jwt expects by default). Extract public key with `openssl rsa -in private.pem -pubout -out public.pem`.
**Warning signs:** "key is not a valid RSA private key" at startup.

### Pitfall 2: Token Family Cleanup Race
**What goes wrong:** During theft detection, concurrent requests with the same stolen token may both trigger family revocation.
**Why it happens:** No distributed lock on the theft detection path.
**How to avoid:** Make deletion idempotent. `DeleteTokenFamily` should succeed even if keys are already deleted. Use Redis pipeline (DEL is idempotent). Log the theft detection event but do not error on already-deleted keys.
**Warning signs:** Error logs during concurrent refresh attempts.

### Pitfall 3: Redis TTL Drift on Family Sets
**What goes wrong:** `token_family:{uuid}` set outlives all its member JTIs, or expires before the last JTI.
**Why it happens:** Family set TTL is set once but members are added later with their own TTL.
**How to avoid:** Reset family set TTL on every `StoreRefreshToken` call to the refresh token TTL (7 days). This is what the pipeline pattern above does.
**Warning signs:** `SMEMBERS token_family:{uuid}` returns JTIs that no longer exist as keys.

### Pitfall 4: Not Clearing password_hash in Proto Responses
**What goes wrong:** `password_hash` field from `models.proto` User message gets populated in responses (SEC-03 violation).
**Why it happens:** Directly mapping domain model to proto without filtering.
**How to avoid:** In every handler that returns User, explicitly set only `id`, `email`, `created_at`, `updated_at`. Never set `password_hash`. The existing Register handler already does this correctly -- follow same pattern.
**Warning signs:** `password_hash` visible in gRPC response inspection.

### Pitfall 5: Logging JWT Tokens
**What goes wrong:** Token strings appear in slog output (SEC-03/D-17 violation).
**Why it happens:** Logging request/response payloads, or logging error context that includes the token.
**How to avoid:** Never pass token strings to slog. Log only JTI, user_id, operation. Review LoggingInterceptor to ensure it does not log request/response bodies.
**Warning signs:** `grep -r "access_token\|refresh_token" *.go` shows log calls.

## Code Examples

### RSA Key Loading at Startup

```go
// Source: golang-jwt/jwt/v5 key parsing [VERIFIED: Go proxy v5.3.1]
import (
    "crypto/rsa"
    "os"
    "github.com/golang-jwt/jwt/v5"
)

func loadRSAKeys(privatePath, publicPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
    privData, err := os.ReadFile(privatePath)
    if err != nil {
        return nil, nil, fmt.Errorf("read private key: %w", err)
    }
    privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privData)
    if err != nil {
        return nil, nil, fmt.Errorf("parse private key: %w", err)
    }

    pubData, err := os.ReadFile(publicPath)
    if err != nil {
        return nil, nil, fmt.Errorf("read public key: %w", err)
    }
    publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pubData)
    if err != nil {
        return nil, nil, fmt.Errorf("parse public key: %w", err)
    }

    return privateKey, publicKey, nil
}
```

### Dev Key Generation (Makefile target)

```bash
# Generate RSA-2048 key pair for local development
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### Login Service Method

```go
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, string, error) {
    // 1. Find user
    user, err := s.storage.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
    if err != nil {
        return nil, "", "", fmt.Errorf("find user: %w", err)
    }
    if user == nil {
        return nil, "", "", domain.ErrInvalidCredentials
    }

    // 2. Compare password
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return nil, "", "", domain.ErrInvalidCredentials
    }

    // 3. Generate tokens
    accessToken, _, err := s.GenerateAccessToken(user.ID, user.Email)
    if err != nil {
        return nil, "", "", fmt.Errorf("generate access token: %w", err)
    }

    refreshToken, err := s.generateAndStoreRefreshToken(ctx, user.ID, user.Email, uuid.New().String())
    if err != nil {
        return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
    }

    return user, accessToken, refreshToken, nil
}
```

### Test RSA Key Helper

```go
// Source: standard crypto/rsa test pattern [ASSUMED]
package authService_test

import (
    "crypto/rand"
    "crypto/rsa"
    "github.com/golang-jwt/jwt/v5"
)

func generateTestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
    key, _ := rsa.GenerateKey(rand.Reader, 2048)
    return key, &key.PublicKey
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| dgrijalva/jwt-go | golang-jwt/jwt/v5 | 2021 (fork), v5 stable 2023 | Must use golang-jwt, not the unmaintained original |
| jwt.Parse without method check | jwt.WithValidMethods option | jwt/v5 | Prevents algorithm confusion (SEC-01) |
| go-redis/v8 | go-redis/v9 | 2022 | Project already uses v9, context-first API |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `rsa.GenerateKey` with 2048 bits is sufficient for test key generation | Code Examples (test helper) | LOW -- standard practice, only affects tests |
| A2 | `json.Marshal` for Redis value encoding is adequate (vs msgpack or protobuf) | Architecture Patterns (Redis) | LOW -- simple struct, json is readable for debugging |

## Open Questions (RESOLVED)

1. **RSA key generation for CI/tests**
   - What we know: Dev keys can be generated via `openssl genrsa`. Tests use in-memory generated keys.
   - What's unclear: Whether CI pipeline needs pre-generated keys or can generate at test time.
   - RESOLVED: Generate in-memory keys in test helpers via `rsa.GenerateKey(rand.Reader, 2048)`; add Makefile target `generate-keys` for local dev. No CI-specific key management needed.

2. **user_tokens set cleanup**
   - What we know: D-05 says `user_tokens:{user_id}` has no TTL, cleaned on last logout.
   - What's unclear: If user never logs out, this set grows indefinitely.
   - RESOLVED: Acceptable for academic project scope. The set grows only by one entry per login session, and LogoutAll cleans it entirely. Could add periodic cleanup later if needed.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All code | Yes | 1.26.2 | -- |
| Redis | Session storage | Not running locally | -- | docker-compose in auth/ brings up Redis on port 6380 |
| PostgreSQL | User lookup (login) | Not running locally | -- | docker-compose in auth/ brings up PG on port 5433 |
| openssl | Dev key generation | Yes (macOS built-in) | -- | -- |

**Missing dependencies with no fallback:** None -- all infrastructure available via docker-compose.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + gotest.tools/v3 + minimock v3.4.7 |
| Config file | None needed (Go standard testing) |
| Quick run command | `cd auth && go test ./internal/services/authService/ -v -count=1` |
| Full suite command | `cd auth && go test ./... -v -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-03 | Login returns access + refresh tokens | unit | `go test ./internal/services/authService/ -run TestLogin -v` | No -- Wave 0 |
| AUTH-04 | Refresh rotates tokens (old deleted, new issued) | unit | `go test ./internal/services/authService/ -run TestRefreshToken -v` | No -- Wave 0 |
| AUTH-05 | Reused refresh token triggers family revocation | unit | `go test ./internal/services/authService/ -run TestRefreshToken_TheftDetection -v` | No -- Wave 0 |
| AUTH-06 | Logout deletes refresh token from Redis | unit | `go test ./internal/services/authService/ -run TestLogout -v` | No -- Wave 0 |
| AUTH-07 | ValidateToken returns user_id + email | unit | `go test ./internal/services/authService/ -run TestValidateToken -v` | No -- Wave 0 |
| SEC-01 | Algorithm confusion rejected (non-RS256) | unit | `go test ./internal/services/authService/ -run TestParseToken_AlgorithmConfusion -v` | No -- Wave 0 |
| SEC-03 | password_hash never in proto response | unit | `go test ./internal/api/auth_service_api/ -run TestRegister_NoPasswordHash -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `cd auth && go test ./internal/services/authService/ -v -count=1`
- **Per wave merge:** `cd auth && go test ./... -v -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `auth/internal/services/authService/login_test.go` -- covers AUTH-03
- [ ] `auth/internal/services/authService/refresh_token_test.go` -- covers AUTH-04, AUTH-05
- [ ] `auth/internal/services/authService/logout_test.go` -- covers AUTH-06
- [ ] `auth/internal/services/authService/validate_token_test.go` -- covers AUTH-07, SEC-01
- [ ] `auth/internal/services/authService/mocks/session_storage_mock.go` -- generated via minimock
- [ ] `go get github.com/golang-jwt/jwt/v5@v5.3.1` -- add dependency

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | bcrypt cost=12 for password verification, generic error on invalid credentials |
| V3 Session Management | yes | Refresh token rotation with family-based theft detection, Redis TTL enforcement |
| V4 Access Control | no | Not in this phase (gateway concern) |
| V5 Input Validation | yes | Email format validation (existing), token format validation via JWT parser |
| V6 Cryptography | yes | RS256 (RSA-2048), algorithm confusion prevention via WithValidMethods |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Algorithm confusion (alg:none, alg:HS256 with public key) | Spoofing | `jwt.WithValidMethods([]string{"RS256"})` -- D-04 |
| Refresh token theft/replay | Spoofing | Token family rotation + revocation on reuse -- D-08, D-09 |
| Credential enumeration | Information Disclosure | Generic "invalid credentials" error for both wrong email and wrong password |
| Password hash leakage | Information Disclosure | Never populate password_hash in proto responses -- D-16 |
| Token logging | Information Disclosure | Never log JWT strings -- D-17, slog discipline |
| Brute force login | Tampering | Rate limiting (deferred to Gateway phase, 15min access TTL limits damage) |

## Sources

### Primary (HIGH confidence)
- Go module proxy (`proxy.golang.org`) -- verified golang-jwt/jwt v5.3.1 latest, go-redis v9.18.0
- Existing codebase (`auth/go.mod`, `auth/config.yaml`, `auth/internal/`) -- verified all existing code patterns and dependencies
- CONTEXT.md decisions D-01 through D-18 -- locked implementation constraints

### Secondary (MEDIUM confidence)
- golang-jwt/jwt/v5 API patterns (ParseWithClaims, WithValidMethods, SigningMethodRS256) -- from training knowledge, consistent with v5 API [ASSUMED but high confidence given library stability]

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries verified in go.mod or Go proxy, versions confirmed
- Architecture: HIGH -- follows existing codebase patterns exactly, decisions are locked
- Pitfalls: HIGH -- well-known JWT/Redis patterns, verified against codebase structure

**Research date:** 2026-04-12
**Valid until:** 2026-05-12 (stable libraries, locked decisions)
