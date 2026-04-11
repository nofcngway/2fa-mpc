# Feature Landscape

**Domain:** Two-factor authentication with distributed secret storage (MPC/Shamir)
**Researched:** 2026-04-11

## Table Stakes

Features that are security requirements. Missing any of these means the system is vulnerable or non-functional.

### Authentication Core

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| User registration with email + password | Basic auth flow, entry point to system | Low | Email uniqueness constraint, bcrypt hashing |
| Password validation (strength) | Weak passwords undermine entire 2FA premise | Low | Project spec: 12+ chars, mixed classes, sequence detection. Note: NIST 800-63B Rev 4 (May 2025) actually **eliminates** composition rules in favor of length + blocklist. Project deviates from NIST intentionally for academic demonstration. |
| Login with credential verification | Core functionality | Low | bcrypt comparison, constant-time |
| JWT access tokens (short-lived) | Stateless auth for service calls | Medium | RS256, 15 min TTL. Must include user_id, 2fa_status in claims |
| JWT refresh tokens (long-lived) | Session continuity without re-login | Medium | 7 day TTL, stored in Redis with explicit TTL, single-use with rotation |
| Refresh token rotation | Prevents replay attacks on stolen refresh tokens | Medium | Issue new refresh on each use, invalidate old one. Detect reuse = revoke all user tokens |
| Logout / token revocation | User must be able to end sessions | Low | Delete refresh token from Redis, access token expires naturally |
| Token validation endpoint | Gateway/services need to verify tokens | Low | Verify RS256 signature, check expiry, return claims |

### TOTP (RFC 6238)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| TOTP secret generation | Core of 2FA setup | Low | 20-byte random secret (160 bits), crypto/rand. Use SHA-1 for compatibility -- SHA-256/512 breaks most authenticator apps |
| Provisioning URI generation | Users need to scan QR code in authenticator app | Low | Format: `otpauth://totp/Issuer:user@email?secret=BASE32&issuer=Issuer&algorithm=SHA1&digits=6&period=30`. Must include issuer parameter twice (in label prefix AND query param) for maximum compatibility |
| TOTP code validation | Core of 2FA verification | Medium | HMAC-SHA1 over time counter, 6 digits, 30-second period |
| Time window tolerance (+/-1 step) | Clock skew between server and authenticator | Low | Accept current period, previous period, and next period. Do NOT go wider than +/-1 -- it weakens security significantly |
| OTP single-use enforcement | Prevent replay attacks | Medium | Store last used time counter in Redis/DB per user. Reject any code with counter <= last_used_counter. Critical: without this, captured OTP is valid for remaining window |
| Secret zeroization after use | TOTP secret must not linger in memory | Medium | Overwrite byte slice with zeros after Shamir split/combine completes. Go GC may copy data -- use runtime.KeepAlive and explicit zeroing |

### Shamir Secret Sharing

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| GF(256) field arithmetic | Correct finite field operations for information-theoretic security | High | Must implement add (XOR), multiply (log/antilog tables), inverse correctly. Integer arithmetic is a critical vulnerability -- leaks information about the secret |
| 2-of-3 secret splitting | Core distributed storage requirement | High | Random polynomial of degree 1 (threshold - 1) over GF(256). Each byte of secret processed independently. Random coefficients from crypto/rand |
| 2-of-3 secret reconstruction | Reassemble secret from any 2 of 3 shares | High | Lagrange interpolation over GF(256). Must work correctly with any 2-share combination |
| Share validation on reconstruct | Verify reconstruction produced valid secret | Medium | After combining, validate TOTP code against user-provided OTP. This is implicit verification -- no separate integrity check needed for this use case |

### MPC Node Security

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| AES-256-GCM encryption at rest | Shares must be encrypted in storage | Medium | Unique nonce per encryption via crypto/rand (96-bit). Never reuse nonces. GCM provides both confidentiality and integrity |
| Per-node encryption keys | Compromise of one node's key doesn't expose other nodes' shares | Low | Each MPC node has its own AES key, loaded from config |
| Node authentication (shared secret) | Prevent unauthorized share operations | Low | gRPC metadata-based auth. Acceptable for academic project; production would use mTLS |
| Share storage with user mapping | Retrieve correct share for user | Low | PostgreSQL: user_id + node_id + encrypted_share + nonce |
| Share deletion | Complete removal when 2FA disabled | Low | Hard delete, not soft delete. All 3 nodes must delete |

### Rate Limiting

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| OTP verification rate limiting | Without this, 6-digit OTP is brute-forced in ~15 minutes at 2500 req/s | Medium | 5 attempts per 5 minutes per user_id (project spec). Use Redis INCR + EXPIRE. Rate limit on user_id, NOT just IP -- IP-based limits are trivially bypassed |
| Account lockout on excessive failures | Sustained attack detection | Low | After N failed windows, require cooldown or re-authentication. Lock for 15-30 minutes |
| Rate limit on login attempts | Credential brute force protection | Medium | Separate from OTP rate limit. Per-IP + per-email combination |

### Backup/Recovery Codes

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Backup code generation | Users locked out if phone lost without recovery path | Medium | Generate 8-10 codes during 2FA setup. Format: 8 alphanumeric chars, grouped (e.g., XXXX-XXXX) for readability |
| Backup code hashing | Codes are equivalent to passwords -- must not store plaintext | Medium | bcrypt each code individually, cost=12. Store hashes in PostgreSQL |
| Single-use backup code consumption | Each code works exactly once | Low | Delete hash after successful use. Return count of remaining codes |
| Backup code display (one-time) | User must save codes during setup -- never shown again | Low | Return codes in Setup2FA response only. Never retrievable after initial display |

### Audit Logging

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Authentication event logging | Security monitoring, incident response | Medium | Log: user_id, event_type, timestamp, IP, user_agent, success/failure |
| 2FA lifecycle event logging | Track setup, verify, disable operations | Medium | Log: user_id, operation (setup/verify/disable), timestamp, result |
| Kafka-based event bus | Decouple audit from business logic | Medium | Async publish, never block business operations on audit. Never include secrets, shares, OTP codes, or passwords in events |
| Failed attempt logging | Detect brute force and suspicious activity | Low | Log all failed OTP and login attempts with metadata |

### Infrastructure (per service)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| gRPC health check | Service discovery, load balancer integration | Low | Standard grpc.health.v1 protocol |
| Graceful shutdown | Clean connection termination | Low | Signal handling, drain in-flight requests, close DB/Redis/Kafka connections |
| Prometheus metrics | Observability baseline | Medium | Request count, latency histograms, error rates, active connections |
| Structured logging (slog) | Debuggability, log aggregation | Low | JSON format, correlation IDs, never log secrets |
| Configuration via YAML | Environment-specific settings | Low | gopkg.in/yaml.v3, no env var overrides needed for academic project |

## Differentiators

Features that provide academic novelty or competitive advantage. These are what make the project interesting beyond a standard 2FA implementation.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Distributed secret storage (no single point of compromise) | Core thesis -- TOTP secret never exists in any persistent store as a whole | High | This is the entire academic contribution. Secret only exists transiently in TwoFA service memory during split/combine |
| Custom Shamir implementation in GF(256) | Demonstrates deep cryptographic understanding for thesis | High | Not using a library forces understanding of finite field arithmetic, polynomial evaluation, Lagrange interpolation |
| Threshold reconstruction (k-of-n) | Tolerates node failure -- any 2 of 3 nodes sufficient | Medium | Built into Shamir scheme. System remains functional if 1 MPC node is down |
| Memory-safe secret handling with zeroization | Defense-in-depth -- secret doesn't linger in process memory | Medium | Explicit byte-slice zeroing after use. Demonstrates security mindset beyond basic implementation |
| Microservice isolation per security domain | Each service has minimal attack surface | Medium | Auth knows nothing about shares. MPC nodes know nothing about TOTP. TwoFA orchestrates but doesn't persist |
| Kafka-based async audit trail | Decoupled, tamper-evident event stream | Medium | Separate from business logic. Could support replay for forensics |

## Anti-Features

Features to explicitly NOT build. These are out of scope per project constraints or would add complexity without value.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| OAuth / SSO / social login | Out of scope per project spec. Adds massive complexity without advancing the MPC/2FA thesis | Email + password registration only |
| Email verification | Not in spec. Requires email service infrastructure, adds onboarding friction for academic demo | Accept email at face value. Unique constraint prevents duplicates |
| Password reset flow | Not in spec. Requires email delivery, token generation, complex state machine | Out of scope -- user manages credentials |
| SMS/push/WebAuthn as 2FA methods | Project is specifically about TOTP + Shamir. Other methods don't demonstrate distributed secret storage | TOTP only, with backup codes as recovery |
| ORM (GORM etc.) | Explicitly forbidden. pgx provides sufficient abstraction | Raw SQL via pgx. initTables for schema |
| Third-party Shamir libraries | Academic requirement: implement from scratch to demonstrate understanding | Custom GF(256) implementation |
| HTTP endpoints in services | Architecture constraint: gRPC only (except future Gateway) | gRPC for all inter-service communication |
| Frontend application | Not in current scope. Backend services are the deliverable | API contracts (protobuf) define the interface |
| API Gateway | Deferred. Focus on core Auth + TwoFA + MPC services first | Direct gRPC calls for testing |
| Verifiable Secret Sharing (VSS) | Adds complexity without clear benefit when dealer (TwoFA) is trusted | Basic Shamir is sufficient since TwoFA service is the trusted dealer |
| mTLS between services | Shared secret auth is simpler and sufficient for academic demonstration | gRPC metadata-based shared secret auth |
| TOTP with SHA-256/SHA-512 | Breaks compatibility with Google Authenticator, Authy, most hardware tokens | SHA-1 as per RFC 6238 defaults. 6 digits, 30-second period |
| Compromised password screening (NIST) | Would require external API (HaveIBeenPwned) or large dataset. Out of scope | Rely on complexity rules as specified in project requirements |
| Token blacklisting for access tokens | Adds statefulness to access tokens, defeating JWT purpose | Short TTL (15 min) makes blacklisting unnecessary. Revoke refresh token instead |

## Feature Dependencies

```
Registration → Login (must have users before auth)
Login → JWT Issuance (login produces tokens)
JWT Issuance → Token Validation (must issue before validating)
JWT Issuance → Refresh Token Rotation (refresh depends on initial issuance)

GF(256) Arithmetic → Shamir Split/Combine (field ops are foundation)
Shamir Split → MPC Share Storage (must split before storing)
MPC Share Storage → MPC Share Retrieval (must store before retrieving)
Shamir Combine → TOTP Validation (must reconstruct secret to validate OTP)

AES-256-GCM → Share Storage (encryption before persistence)
Share Storage → Share Retrieval → Share Deletion

TOTP Secret Generation → Shamir Split → Share Distribution → Provisioning URI (full setup flow)
TOTP Validation + Rate Limiting (rate limit wraps validation)
OTP Single-Use → TOTP Validation (enforce after successful validation)

Backup Code Generation → Backup Code Hashing → Storage (during setup)
Backup Code Verification → Single-Use Consumption (during recovery)

All Auth Events → Kafka Audit (async, non-blocking)
All 2FA Events → Kafka Audit (async, non-blocking)
```

## MVP Recommendation

### Phase 1: Auth Service (foundation)
Prioritize:
1. Registration with password validation and bcrypt hashing
2. Login with JWT RS256 issuance (access + refresh)
3. Refresh token rotation via Redis
4. Token validation endpoint
5. Kafka audit event publishing
6. Prometheus metrics, health check, graceful shutdown

### Phase 2: TwoFA + MPC Core (the thesis)
Prioritize:
1. GF(256) field arithmetic with comprehensive tests
2. Shamir 2-of-3 split and combine
3. MPC node service: store/retrieve/delete shares with AES-256-GCM
4. TOTP secret generation and provisioning URI
5. TOTP code validation with +/-1 window tolerance
6. Full Setup2FA flow (generate -> split -> distribute -> return URI)
7. Full Verify2FA flow (retrieve shares -> combine -> validate OTP -> zeroize)

### Phase 3: Hardening and Completion
Prioritize:
1. OTP single-use enforcement
2. Rate limiting on OTP verification (5/5min per user)
3. Backup code generation, hashing, and single-use consumption
4. Disable2FA flow
5. Get2FAStatus endpoint
6. Login rate limiting
7. Complete audit event coverage

Defer:
- API Gateway: not needed for core functionality demonstration
- Frontend: out of scope
- Monitoring dashboards (Grafana): config only, not application code

## Confidence Assessment

| Finding | Confidence | Source |
|---------|------------|--------|
| TOTP RFC 6238 parameters (SHA-1, 6 digits, 30s) | HIGH | RFC 6238 specification, Google Authenticator Key URI format wiki |
| Provisioning URI format (otpauth://) | HIGH | Google Authenticator wiki, IETF draft |
| +/-1 time step tolerance | HIGH | RFC 6238, multiple implementation guides |
| Shamir GF(256) security requirements | HIGH | Academic literature, multiple implementations (codahale/shamir, HashiCorp Vault) |
| Backup codes: 8-10 count, bcrypt hashed | MEDIUM | Industry practice (GitHub uses 16, others 8-10), no formal standard |
| Rate limiting: per-user-id not per-IP | HIGH | Multiple HackerOne reports demonstrating IP-bypass attacks |
| OTP brute force: ~15 min without rate limiting | HIGH | Documented CVEs and security advisories |
| Refresh token rotation with reuse detection | HIGH | Auth0, Okta, OWASP recommendations |
| NIST 800-63B Rev 4 eliminates composition rules | HIGH | NIST SP 800-63B-4 (May 2025) |
| Secret zeroization effectiveness in Go (GC concerns) | MEDIUM | Known Go GC behavior, but explicit zeroing is still best practice |

## Sources

- [RFC 6238 - TOTP Specification](https://datatracker.ietf.org/doc/html/rfc6238)
- [Google Authenticator Key URI Format](https://github.com/google/google-authenticator/wiki/Key-Uri-Format)
- [NIST SP 800-63B Revision 4](https://pages.nist.gov/800-63-4/sp800-63b.html)
- [Shamir Secret Sharing Best Practices (RWoT)](https://github.com/WebOfTrustInfo/rwot8-barcelona/blob/master/draft-documents/shamir-secret-sharing-best-practices.md)
- [5 Common TOTP Mistakes (Authgear 2026)](https://www.authgear.com/post/5-common-totp-mistakes)
- [Auth0 - Refresh Tokens](https://auth0.com/blog/refresh-tokens-what-are-they-and-when-to-use-them/)
- [Shamir Security Considerations (Ethereum Research)](https://ethresear.ch/t/security-considerations-for-shamirs-secret-sharing/4294)
- [2FA Bypass Techniques (HackTricks)](https://book.hacktricks.wiki/en/pentesting-web/2fa-bypass.html)
- [Audit Logs for MFA (Hoop.dev)](https://hoop.dev/blog/audit-logs-for-multi-factor-authentication-mfa-tracking-security-events-made-simple)
- [PortSwigger - MFA Vulnerabilities](https://portswigger.net/web-security/authentication/multi-factor)
- [IETF otpauth URI draft](https://www.ietf.org/archive/id/draft-linuxgemini-otpauth-uri-00.html)
