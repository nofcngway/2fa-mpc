# Phase 8: TwoFA Verification & Management - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 08-twofa-verification-management
**Areas discussed:** Share retrieval strategy, Rate limiting design, OTP reuse prevention, Disable 2FA cleanup

---

## Share Retrieval Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| All 3 in parallel, use first 2 | Query all 3 MPC nodes via errgroup. As soon as 2 succeed, cancel remaining. Maximizes availability. | ✓ |
| Fixed pair with fallback | Try nodes 1+2 first. If either fails, try node 3 as replacement. Simpler but slower on failure. | |
| All 3, require all 3 | Retrieve all 3 shares, combine any 2. Most redundant but slowest. | |

**User's choice:** All 3 in parallel, use first 2
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| defer zeroize() for shares + secret | Same pattern as Phase 7 setup. Consistent approach. | ✓ |
| Zeroize only the combined secret | Shares are encrypted in transit and short-lived. | |
| You decide | Claude's discretion | |

**User's choice:** defer zeroize() for shares + secret
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Proceed silently | 2-of-3 is the design. Log node failure internally. | ✓ |
| Proceed with warning field | Add warnings field to response. | |
| You decide | Claude's discretion | |

**User's choice:** Proceed silently
**Notes:** None

---

## Rate Limiting Design

| Option | Description | Selected |
|--------|-------------|----------|
| INCR + EXPIRE | Redis INCR on key, EXPIRE 300s. Simple, atomic, standard. | ✓ |
| Lua script (atomic) | Single Lua script for INCR + conditional EXPIRE. Fully atomic. | |
| Sliding window (sorted set) | ZADD per attempt, ZREMRANGEBYSCORE + ZCARD. Precise but heavier. | |

**User's choice:** INCR + EXPIRE
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Allow verification | Log warning, proceed without rate check. Availability over enforcement. | ✓ |
| Deny verification | Return codes.Unavailable. Strict security. | |
| You decide | Claude's discretion | |

**User's choice:** Allow verification when Redis unavailable
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| All attempts | Count every Verify2FA call regardless of outcome. | ✓ |
| Failed only | Only increment on invalid OTP. | |
| You decide | Claude's discretion | |

**User's choice:** All attempts
**Notes:** None

---

## OTP Reuse Prevention

| Option | Description | Selected |
|--------|-------------|----------|
| Redis with TTL | Key 'otp_used:{user_id}', value: time counter, TTL: 90s. | ✓ |
| PostgreSQL column | Add 'last_used_counter' column to twofa_records. | |
| You decide | Claude's discretion | |

**User's choice:** Redis with TTL
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Same time counter value | Store exact TOTP counter. Reject if new == stored. | ✓ |
| Any overlapping window | Store all 3 counters. Reject if any match. | |
| You decide | Claude's discretion | |

**User's choice:** Same time counter value
**Notes:** None

---

## Disable 2FA Cleanup

| Option | Description | Selected |
|--------|-------------|----------|
| Delete shares first, then metadata | 1) Verify OTP, 2) Delete shares parallel, 3) Delete backup codes, 4) Delete record. | ✓ |
| Delete metadata first, then shares | 1) Verify OTP, 2) Delete record + codes, 3) Delete shares. | |
| You decide | Claude's discretion | |

**User's choice:** Delete shares first, then metadata
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Return error, retry later | Return codes.Internal. Record stays. User retries. | ✓ |
| Best-effort delete, proceed | Delete from responding nodes, remove metadata. Accept orphans. | |
| You decide | Claude's discretion | |

**User's choice:** Return error, retry later
**Notes:** None

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, delete Redis keys | DEL rate_limit and otp_used keys. Clean slate. | ✓ |
| Let them expire naturally | Short TTLs, auto-expire. Less code. | |
| You decide | Claude's discretion | |

**User's choice:** Yes, delete Redis keys
**Notes:** None

---

## Claude's Discretion

- Exact errgroup cancellation pattern for "first 2 wins" retrieval
- Whether to wrap rate limit check + OTP validation in single or separate service methods
- Prometheus metric labels for verify/disable/status operations
- Kafka audit event structure
- Internal helper decomposition
- Error message wording (no internal state leakage)

## Deferred Ideas

- Kafka audit events — Phase 9
- Prometheus metrics — Phase 9
- Backup code verification as alternative to OTP — potential future enhancement
