# Domain Pitfalls

**Domain:** 2FA authentication with MPC distributed secret storage (Shamir Secret Sharing)
**Researched:** 2026-04-11

## Critical Pitfalls

Mistakes that cause security breaches, data loss, or require rewrites.

### Pitfall 1: GF(256) Arithmetic Errors in Shamir Implementation

**What goes wrong:** Custom Shamir Secret Sharing implementations over GF(256) contain subtle bugs in finite field arithmetic -- incorrect multiplication/inversion tables, off-by-one errors in polynomial evaluation, or non-uniform coefficient generation. Trail of Bits disclosed vulnerabilities in multiple production Shamir implementations (Binance tss-lib and forks) in 2021, where bugs in modular arithmetic allowed attackers to steal secret keys or crash nodes.

**Why it happens:** GF(256) arithmetic (multiplication, inversion via log/antilog tables or Russian peasant multiplication) is easy to get subtly wrong. A single incorrect entry in a 256-element lookup table silently corrupts all shares. Polynomial evaluation at x=0 (the secret) works correctly but evaluation at other points produces wrong shares, making unit tests on small cases pass while real usage fails.

**Consequences:** Shares that cannot be recombined (users locked out of 2FA permanently), or worse, shares that leak information about the secret (security completely broken). Since this is a 2-of-3 scheme, even partial correctness bugs mean the system silently degrades.

**Prevention:**
- Build GF(256) multiplication table from the irreducible polynomial (0x11B for AES field) and verify ALL 256x256 = 65536 entries against a known-good reference
- Property-based tests: for random a, b, c in GF(256), verify associativity `(a*b)*c == a*(b*c)`, commutativity `a*b == b*a`, distributivity `a*(b+c) == a*b + a*c`, identity `a*1 == a`, inverse `a * inv(a) == 1` for all non-zero a
- Round-trip test: generate secret, split into n shares, combine every valid k-subset, verify all produce identical secret
- Test with ALL possible single-byte secrets (0x00 through 0xFF)
- Verify that any k-1 shares reveal zero information (information-theoretic security property)

**Detection:** Intermittent 2FA verification failures that appear as "wrong TOTP code" but are actually wrong secret reconstruction. If users report valid codes being rejected inconsistently, suspect Shamir arithmetic bugs first.

**Confidence:** HIGH -- Trail of Bits disclosures and multiple real-world incidents confirm this is the primary risk area.

**Phase:** Auth Service (Shamir implementation phase). Must be bulletproof before any integration testing.

---

### Pitfall 2: TOTP Secret Not Zeroized in Go Memory (GC Complications)

**What goes wrong:** The TOTP secret is reconstructed in memory from Shamir shares, used for TOTP validation, then "zeroized" by overwriting the byte slice. However, Go's garbage collector may have already copied the data to a new memory location (during slice growth, stack-to-heap escape, or GC compaction), leaving the original secret in freed memory pages accessible via memory dump.

**Why it happens:** Go's GC is a moving collector in some scenarios. When a `[]byte` escapes to the heap, or when the stack grows and is relocated, the runtime copies data and the old location is not zeroed. Calling `for i := range secret { secret[i] = 0 }` only zeros the current copy, not previous locations. Additionally, Go strings are immutable and cannot be zeroed at all -- if the secret ever touches a `string` type, it persists until GC collects and the OS reclaims the page.

**Consequences:** Secrets remain in process memory longer than intended. In a forensic or cold-boot attack scenario, the TOTP master secret can be extracted from memory dumps despite "zeroization."

**Prevention:**
- Go 1.26 includes experimental `runtime/secret` package (GOEXPERIMENT=runtimesecret) -- but only on linux/amd64 and linux/arm64, so it may not work in all deployment environments (not on macOS for development)
- NEVER convert `[]byte` secrets to `string` -- keep them as `[]byte` throughout their lifetime
- Pre-allocate fixed-size buffers for secrets (avoid slice append/grow which creates copies)
- Use `runtime.KeepAlive()` to prevent premature GC of the buffer before zeroization
- Minimize the window: reconstruct secret, compute TOTP, zeroize -- all in the same function scope, ideally under 10 lines
- Run escape analysis: `go build -gcflags="-m"` to verify the secret buffer does not escape to heap
- For the academic project scope, document this limitation honestly -- perfect zeroization in Go without `runtime/secret` is not fully achievable

**Detection:** Use `go build -gcflags="-m"` during CI to detect heap escapes of secret-holding variables. Any escape of the TOTP secret buffer is a warning sign.

**Confidence:** HIGH -- well-documented Go memory model limitation. `runtime/secret` is the official answer but is experimental.

**Phase:** TwoFA Service implementation. Must be addressed when implementing Verify2FA flow.

---

### Pitfall 3: AES-256-GCM Nonce Reuse (Catastrophic)

**What goes wrong:** Two different shares encrypted with the same AES-256-GCM key and nonce allows an attacker to XOR ciphertexts to recover plaintext and forge authentication tags. This completely breaks both confidentiality and integrity of ALL shares encrypted with that key-nonce pair.

**Why it happens:** Using a counter-based nonce without persistence (counter resets on service restart), using a deterministic nonce derived from predictable input (like user_id), or generating random nonces without sufficient entropy. With 96-bit random nonces, birthday bound collision probability becomes non-negligible after ~2^32 encryptions per key.

**Consequences:** Complete compromise of share confidentiality and integrity. Attacker can recover plaintext shares and forge new ones. This is not a theoretical risk -- it is a mathematical certainty if nonce reuse occurs.

**Prevention:**
- Use `crypto/rand` for every nonce generation (already specified in project constraints -- enforce it)
- Use Go's `cipher.NewGCMWithRandomNonce()` (available in recent Go versions) which handles nonce generation and prepending automatically
- NEVER derive nonces deterministically from user_id, share_id, or timestamps
- Implement key rotation: rotate the AES-256 encryption key after 2^32 encryptions per MPC node (track encryption count in DB)
- Store nonce alongside ciphertext (standard GCM practice) -- do not try to reconstruct it
- If using a counter nonce, persist the counter to disk atomically before encrypting (survives restarts)

**Detection:** No runtime detection is possible -- nonce reuse is silent. Prevention is the only strategy. Code review must verify every `Seal()` call uses a fresh random nonce.

**Confidence:** HIGH -- mathematically proven catastrophic failure mode of GCM.

**Phase:** MPC Node implementation. The encryption layer must be correct from day one.

---

### Pitfall 4: JWT Algorithm Confusion Attack (RS256 to HS256 Downgrade)

**What goes wrong:** An attacker modifies the JWT header's `alg` field from "RS256" to "HS256" and signs the token using the public key (which is public knowledge) as the HMAC secret. If the server naively trusts the `alg` header to select verification logic, it will verify the forged token successfully.

**Why it happens:** The JWT spec allows the token itself to declare its algorithm. Libraries that implement "verify using whatever algorithm the token says" are vulnerable. The `golang-jwt/jwt/v5` library mitigates this if you use `jwt.WithValidMethods()` or pass the correct key type, but the developer must explicitly configure it.

**Consequences:** Complete authentication bypass. Any user can forge tokens for any other user, including admin accounts.

**Prevention:**
- ALWAYS specify allowed algorithms explicitly: `parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))`
- NEVER use `jwt.Parse()` with a keyfunc that returns the same key regardless of algorithm
- The keyfunc should return `*rsa.PublicKey` only -- if the library receives an HMAC token, the type mismatch will cause verification failure
- Store the private key securely (not in config.yaml -- use environment variable or mounted secret)
- Validate all standard claims: `exp`, `iat`, `iss`, `sub`

**Detection:** Monitor for tokens with unexpected `alg` values in logs (without logging the token itself). Alert on any non-RS256 algorithm appearing.

**Confidence:** HIGH -- well-documented attack vector, explicitly addressed in golang-jwt/jwt/v5 documentation.

**Phase:** Auth Service implementation. Must be correct in the initial JWT setup.

---

### Pitfall 5: Refresh Token Race Condition (Double-Use on Rotation)

**What goes wrong:** Two concurrent requests hit the refresh endpoint with the same refresh token. Both read the token from Redis, both validate it, both issue new access+refresh tokens, and both try to delete/replace the old token. Depending on timing: one request succeeds and one gets a 401 (annoying), or both succeed and two valid refresh tokens exist (security breach -- token family not invalidated).

**Why it happens:** Redis individual operations are atomic, but the read-validate-delete-write sequence is not atomic. In a microservice environment with multiple Auth Service instances, this race window is wider.

**Consequences:** At minimum, poor user experience (random 401 errors on mobile apps that retry aggressively). At worst, refresh token reuse goes undetected, enabling session hijacking.

**Prevention:**
- Use Redis WATCH/MULTI/EXEC (optimistic locking) or a Lua script to make the rotate operation atomic:
  ```
  -- Lua: atomic token rotation
  local old = redis.call('GET', KEYS[1])
  if old == ARGV[1] then
    redis.call('DEL', KEYS[1])
    redis.call('SET', KEYS[2], ARGV[2], 'EX', ARGV[3])
    return 1
  end
  return 0
  ```
- Alternatively, use Redis `GETDEL` (Redis 6.2+) to atomically read and delete the old token, then set the new one
- Implement refresh token families: if a previously-used refresh token is presented, invalidate ALL tokens in the family (detect replay attacks)
- Set a short grace period (e.g., 5 seconds) where the old token is still accepted after rotation

**Detection:** Monitor Redis for multiple DEL operations on the same refresh token key within a short window. Log refresh token rotation events to Kafka audit.

**Confidence:** HIGH -- well-documented concurrency issue in token rotation systems.

**Phase:** Auth Service implementation, specifically the RefreshToken RPC.

## Moderate Pitfalls

### Pitfall 6: TOTP Clock Skew and Time Window Edge Cases

**What goes wrong:** TOTP validation rejects valid codes because server time and authenticator app time differ by more than the allowed window. Also, codes generated at the boundary of a 30-second window may be valid on the client but have already rolled over on the server.

**Why it happens:** NTP drift on containers, VM clock issues, or authenticator apps on phones with incorrect time. The project allows +/-1 window (+-30s), but accumulated drift beyond 60 seconds causes permanent failure.

**Prevention:**
- Validate TOTP against current time step AND +/-1 adjacent steps (already planned)
- Use `time.Now().UTC()` consistently -- never local time
- Ensure Docker containers and host machines have NTP configured
- When TOTP verification fails, log the time step difference (without the code or secret) for diagnostics
- Document for users: "ensure your phone's time is set to automatic"

**Detection:** Track TOTP verification failure rates per user. A sudden spike in failures across multiple users suggests server clock drift. A single user with persistent failures suggests client clock drift.

**Confidence:** MEDIUM -- standard TOTP implementation concern, well-understood mitigation.

**Phase:** TwoFA Service, Verify2FA implementation.

---

### Pitfall 7: Shamir Share Distribution Partial Failure

**What goes wrong:** During 2FA setup, the TwoFA service splits the secret into 3 shares and sends them to 3 MPC nodes. Node 1 and 2 accept their shares, but Node 3 fails (network timeout, disk full, etc.). Now 2 shares exist but the system recorded a successful setup. Later, if Node 1 goes down, only Node 2's share is available -- not enough for 2-of-3 reconstruction.

**Why it happens:** Distributed write without transactional guarantees. Each MPC node StoreShare is an independent gRPC call. There is no distributed transaction protocol.

**Consequences:** User cannot verify 2FA, cannot disable 2FA, effectively locked out of their account.

**Prevention:**
- Implement a saga pattern: store shares sequentially, if any fails, compensate by deleting already-stored shares
- Store the share distribution status in TwoFA's PostgreSQL: `pending -> storing -> share_1_ok -> share_2_ok -> share_3_ok -> active`
- Only mark 2FA as "active" after ALL 3 shares confirmed stored
- Implement a cleanup job that detects stuck `storing` states and rolls back
- Add retry with exponential backoff for individual share stores before triggering rollback
- Consider: if exactly 2 shares stored successfully, the system still works for 2-of-3 -- decide if this is acceptable or if all-3 is required for redundancy

**Detection:** Monitor the share distribution status table for entries stuck in intermediate states. Alert if any setup takes longer than 30 seconds.

**Confidence:** HIGH -- fundamental distributed systems problem, directly applicable to 2-of-3 architecture.

**Phase:** TwoFA Service, Setup2FA implementation.

---

### Pitfall 8: pgx Connection Pool Exhaustion Under Load

**What goes wrong:** Each service uses pgxpool to manage PostgreSQL connections. Under burst load (especially during 2FA verification which hits TwoFA DB + 2 MPC node DBs), connection pools exhaust, requests queue up, timeouts cascade, and the entire service becomes unresponsive.

**Why it happens:** Default pgxpool MaxConns is `max(4, runtime.NumCPU())`. In a container with 2 CPUs, that is 4 connections. If Verify2FA takes 200ms (2 MPC calls + DB lookups) and 20 concurrent users verify simultaneously, the pool is saturated.

**Consequences:** Request timeouts, cascading failures across services, 2FA verification unavailable.

**Prevention:**
- Set explicit `MaxConns` based on expected concurrency: start with `(CPU * 2) + 1`, tune from there
- Set `MaxConnIdleTime` (e.g., 30 minutes) to reclaim idle connections
- Set `MaxConnLifetime` with jitter to prevent thundering herd on connection refresh
- Set acquire timeout on the pool (context with deadline) so callers fail fast instead of blocking indefinitely
- For MPC nodes: they each have their own DB, so pool sizing is per-node
- Monitor pool statistics via Prometheus: `pgxpool.Stat()` exposes `AcquireCount`, `AcquiredConns`, `IdleConns`, `TotalConns`

**Detection:** Prometheus metrics on pool wait time and acquired connections. Alert when acquire latency exceeds 100ms.

**Confidence:** HIGH -- standard production issue with pgx, well-documented.

**Phase:** Infrastructure setup in each service. Configure in bootstrap phase.

---

### Pitfall 9: gRPC Error Messages Leaking Internal State

**What goes wrong:** A gRPC handler catches a PostgreSQL error (e.g., `pq: duplicate key value violates unique constraint "users_email_key"`) and wraps it directly in a gRPC status error: `status.Errorf(codes.Internal, "database error: %v", err)`. This exposes table names, constraint names, and database schema to the client. A recent vulnerability (CVE in grpc-go 1.64.x) showed that logging contexts with gRPC metadata can leak tokens.

**Why it happens:** Developers wrap errors for debugging convenience. Go's `fmt.Errorf("...: %w", err)` chains propagate raw errors across service boundaries. In gRPC, the error message is sent to the caller.

**Consequences:** Information disclosure (database schema, internal error details, potentially secrets in metadata). Attackers use this to map internal architecture for targeted attacks.

**Prevention:**
- At every gRPC handler boundary, translate errors into domain-specific messages: `status.Error(codes.AlreadyExists, "email already registered")` -- never include the raw error
- Log the full error server-side with slog (for debugging), but send only the sanitized status to the client
- Create an error translation interceptor (gRPC middleware) that catches any `codes.Internal` errors and strips details before sending
- NEVER log `context.Context` directly -- it may contain gRPC metadata with tokens
- NEVER include `err.Error()` from downstream services in gRPC responses

**Detection:** Code review: search for `status.Errorf(codes.Internal, ".*%v"` patterns -- any format string that includes `%v` or `%w` with an error variable is suspect.

**Confidence:** HIGH -- documented gRPC vulnerability and common Go anti-pattern.

**Phase:** Every service, from the first gRPC handler implementation. Establish the error handling pattern in Auth Service (first service built) and replicate.

---

### Pitfall 10: Kafka Audit Events Lost on Service Crash

**What goes wrong:** A service performs an operation (e.g., successful login), then attempts to publish an audit event to Kafka. If the service crashes between the operation and the publish, or if Kafka is temporarily unavailable, the audit event is silently lost. For a security-critical audit trail, missing events undermine the entire audit purpose.

**Why it happens:** Publishing to Kafka after committing to PostgreSQL is a dual-write problem. There is no atomic transaction spanning PostgreSQL and Kafka.

**Consequences:** Incomplete audit trail. Security incidents may go undetected because the corresponding audit events were lost.

**Prevention:**
- Use the transactional outbox pattern: write audit events to a PostgreSQL `audit_outbox` table in the same transaction as the business operation, then have a separate goroutine/process poll the outbox and publish to Kafka
- For the academic project scope, the simpler approach is acceptable: publish to Kafka with `RequireAll` acks and retry on failure, accepting that crash-window losses are possible -- document this limitation
- Configure `segmentio/kafka-go` writer with `RequiredAcks: kafka.RequireAll` for durability
- Set `WriteTimeout` and implement retry with backoff
- Never block the main request path on Kafka publish -- use async publish with a buffered channel

**Detection:** Compare audit event counts in Kafka vs. operation counts in PostgreSQL. Discrepancies indicate lost events.

**Confidence:** MEDIUM -- the full outbox pattern may be over-engineering for an academic project, but the limitation should be documented.

**Phase:** Infrastructure setup. Decide on outbox vs. fire-and-forget in the architecture phase.

## Minor Pitfalls

### Pitfall 11: Protobuf Schema Drift Between TwoFA and MPC

**What goes wrong:** The MPC node proto is defined in TwoFA and copied to MPC. After initial setup, a field is added to the TwoFA-side proto but not copied to MPC, or field numbers are reused after deletion. gRPC calls start failing with deserialization errors or silently drop new fields.

**Prevention:**
- Use a shared proto directory or git submodule for proto files shared between services
- Never reuse protobuf field numbers -- mark deleted fields as `reserved`
- Run `buf lint` or `buf breaking` in CI to detect breaking changes
- Generate Go code from the same proto source for both services

**Confidence:** MEDIUM -- operational risk, standard in multi-service protobuf projects.

**Phase:** TwoFA + MPC integration phase.

---

### Pitfall 12: Graceful Shutdown Order (Wrong Sequence = Data Loss)

**What goes wrong:** On SIGTERM, all connections close simultaneously. In-flight gRPC requests get dropped. Kafka messages in the buffer are lost. Database transactions are aborted mid-write.

**Prevention:**
- Shutdown in reverse initialization order:
  1. Stop accepting new gRPC connections (`grpcServer.GracefulStop()`)
  2. Wait for in-flight requests to complete (with timeout)
  3. Flush Kafka writer (`writer.Close()` flushes buffer)
  4. Close Redis connections
  5. Close PostgreSQL pool (`pool.Close()`)
- Use `signal.NotifyContext` for clean signal handling
- Set a hard deadline (e.g., 30 seconds) after which force-kill

**Confidence:** HIGH -- standard microservice concern, well-documented patterns.

**Phase:** Every service, cmd/app/main.go. Establish pattern in Auth Service.

---

### Pitfall 13: bcrypt Timing Side Channel on User Enumeration

**What goes wrong:** Login with a non-existent email returns immediately (no bcrypt hash to compare), while login with an existing email takes ~250ms (bcrypt cost=12 comparison). An attacker can enumerate valid emails by measuring response time.

**Prevention:**
- When a user is not found, still perform a bcrypt comparison against a dummy hash: `bcrypt.CompareHashAndPassword(dummyHash, password)`. This ensures constant-time response regardless of whether the email exists.
- Pre-compute the dummy hash at startup: `dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), 12)`

**Confidence:** HIGH -- well-known timing attack, simple fix.

**Phase:** Auth Service, Login implementation.

---

### Pitfall 14: MPC Shared Secret in gRPC Metadata Logged Accidentally

**What goes wrong:** MPC nodes authenticate TwoFA service via a shared secret in gRPC metadata. If any logging middleware logs the full metadata map (common in debug logging), the shared secret appears in log files.

**Prevention:**
- The gRPC interceptor for MPC authentication must extract only the auth token and never log it
- Implement a metadata sanitizer that strips sensitive keys before logging
- Use slog with structured fields -- never `slog.Any("metadata", md)` on raw gRPC metadata
- The shared secret should be compared using `subtle.ConstantTimeCompare` to prevent timing attacks

**Confidence:** HIGH -- directly related to grpc-go metadata logging vulnerability (CVE in v1.64.x).

**Phase:** MPC Node, interceptor implementation.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Shamir GF(256) implementation | Arithmetic table errors, off-by-one in polynomial eval | Exhaustive property-based tests, reference table verification |
| TOTP implementation | Clock skew, algorithm mismatch (SHA1 vs SHA256) | Use UTC consistently, test with +-1 window, match Google Authenticator defaults (SHA1, 6 digits, 30s) |
| MPC Node encryption | AES-GCM nonce reuse | Use `NewGCMWithRandomNonce()` or `crypto/rand` for every nonce, never derive deterministically |
| Auth JWT | Algorithm confusion, key management | `jwt.WithValidMethods(["RS256"])`, store private key outside config.yaml |
| Auth Login | Timing side channel on user enumeration | bcrypt dummy hash comparison for non-existent users |
| TwoFA Setup | Partial share distribution failure | Saga pattern with rollback, status tracking in DB |
| Token Refresh | Concurrent rotation race condition | Atomic Redis operation via Lua script or GETDEL |
| Secret Zeroization | Go GC copies secrets in memory | Avoid heap escape, pre-allocate buffers, minimize secret lifetime, document limitation |
| All gRPC handlers | Error message information leakage | Error translation at handler boundary, never forward raw errors |
| All services | Connection pool exhaustion | Explicit pgxpool MaxConns, monitoring, acquire timeouts |
| All services | Graceful shutdown ordering | Reverse-order shutdown: gRPC -> Kafka -> Redis -> PostgreSQL |
| Audit trail | Kafka event loss on crash | Transactional outbox or documented fire-and-forget limitation |
| Proto contracts | Schema drift between TwoFA and MPC | Shared proto source, buf lint, reserved field numbers |

## Sources

- [Trail of Bits: Disclosing Shamir's Secret Sharing vulnerabilities](https://blog.trailofbits.com/2021/12/21/disclosing-shamirs-secret-sharing-vulnerabilities-and-announcing-zkdocs/) -- HIGH confidence
- [WebOfTrust: Shamir Secret Sharing Best Practices](https://github.com/WebOfTrustInfo/rwot8-barcelona/blob/master/draft-documents/shamir-secret-sharing-best-practices.md) -- HIGH confidence
- [Go runtime/secret package documentation](https://pkg.go.dev/runtime/secret) -- HIGH confidence
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26) -- HIGH confidence
- [Go & AES-GCM: A Security Deep Dive](https://dev.to/js402/go-aes-gcm-a-security-deep-dive-3ec8) -- MEDIUM confidence
- [RFC 8452: AES-GCM-SIV Nonce Misuse-Resistant Encryption](https://www.rfc-editor.org/rfc/rfc8452.html) -- HIGH confidence
- [5 Common TOTP Mistakes Developers Make](https://www.authgear.com/post/5-common-totp-mistakes) -- MEDIUM confidence
- [RFC 6238: TOTP](https://www.rfc-editor.org/rfc/rfc6238) -- HIGH confidence
- [JetBrains: Secure Error Handling in Go](https://blog.jetbrains.com/go/2026/03/02/secure-go-error-handling-best-practices/) -- MEDIUM confidence
- [gRPC Security Best Practices](https://www.stackhawk.com/blog/best-practices-for-grpc-security/) -- MEDIUM confidence
- [JWT Security Pitfalls](https://mojoauth.com/ciam-qna/jwt-security-pitfalls-implementation) -- MEDIUM confidence
- [pgx Connection Pooling Best Practices](https://hexacluster.ai/blog/postgresql-client-side-connection-pooling-in-golang-using-pgxpool) -- MEDIUM confidence
- [Redis Race Conditions in Go](https://hackernoon.com/fixing-race-conditions-in-go-with-redis-based-distributed-locks) -- MEDIUM confidence
- [Kafka Delivery Semantics](https://docs.confluent.io/kafka/design/delivery-semantics.html) -- HIGH confidence
