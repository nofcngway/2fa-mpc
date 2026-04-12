---
phase: 06-mpc-node-service
auditor: gsd-security-auditor
asvs_level: 1
generated: "2026-04-12"
threats_total: 11
threats_closed: 11
threats_open: 0
block_on: critical
result: SECURED
---

# Security Audit — Phase 06: MPC Node Service

## Summary

All 11 registered threats verified CLOSED. No open threats. No critical gaps.

| Metric | Value |
|--------|-------|
| Phase | 06 — mpc-node-service |
| ASVS Level | 1 |
| Threats Closed | 11/11 |
| Threats Open | 0/11 |
| Unregistered Flags | 0 |
| Accepted Risks Logged | 1 (T-06-04) |

---

## Threat Verification

| Threat ID | Category | Component | Disposition | Status | Evidence |
|-----------|----------|-----------|-------------|--------|----------|
| T-06-01 | Tampering | encrypt.go | mitigate | CLOSED | `gcm.Seal` / `gcm.Open` at encrypt.go:24,41 — GCM tag authentication rejects tampered ciphertext |
| T-06-02 | Information Disclosure | retrieve_share.go | mitigate | CLOSED | `slog.Error` at retrieve_share.go:24–28 logs only `user_id`, `share_index`, `node_id`; no `share_data` or key present |
| T-06-03 | Tampering | share.go (storage) | mitigate | CLOSED | Parameterized queries `$1, $2, $3, $4, $5, $6` at share.go:23–26; `$1 AND $2` at share.go:41–42; `$1` at share.go:59 |
| T-06-04 | Repudiation | store_share.go | accept | CLOSED | Accepted: audit logging deferred to Phase 9 (Kafka events); see Accepted Risks section below |
| T-06-05 | Information Disclosure | encrypt.go | mitigate | CLOSED | `rand.Read(nonce)` at encrypt.go:21; nonce sized by `gcm.NonceSize()` at encrypt.go:20; never derived from user data |
| T-06-06 | Spoofing | interceptors.go | mitigate | CLOSED | `subtle.ConstantTimeCompare` at interceptors.go:36; missing/empty/wrong secret all return `codes.Unauthenticated` |
| T-06-07 | Information Disclosure | interceptors.go | mitigate | CLOSED | Error strings are generic: `"missing metadata"`, `"missing authorization"`, `"invalid authorization"` at interceptors.go:28,32,37 |
| T-06-08 | Elevation of Privilege | interceptors.go | mitigate | CLOSED | Exact FullMethod match `"/grpc.health.v1.Health/Check"` at interceptors.go:22; all other paths require auth |
| T-06-09 | Denial of Service | bootstrap.go | mitigate | CLOSED | `len(key) != 32` guard at bootstrap.go:34–36; service returns error and refuses to start on invalid key |
| T-06-10 | Information Disclosure | store_share.go (handler) | mitigate | CLOSED | Handler errors: `"failed to store share"` at api/store_share.go:30; `"failed to retrieve share"` at api/retrieve_share.go:27; `"failed to delete shares"` at api/delete_share.go:18 — no internal details exposed |
| T-06-11 | Tampering | handlers | mitigate | CLOSED | Input validation at api/store_share.go:15–23, api/retrieve_share.go:15–19, api/delete_share.go:13–15; `user_id` non-empty, `share_data` non-empty, `share_index >= 0` |

---

## Accepted Risks

| Threat ID | Category | Acceptance Rationale | Phase Deferred To |
|-----------|----------|----------------------|-------------------|
| T-06-04 | Repudiation | Store operations produce no audit trail in Phase 06. Kafka audit events are architecturally planned and will be implemented in Phase 9. The MPC node is an internal service not directly accessible from external clients; the TwoFA orchestrator is the accountability boundary for share operations. | Phase 09 |

---

## Unregistered Threat Flags

No unregistered threat flags were raised in 06-01-SUMMARY.md or 06-02-SUMMARY.md `## Threat Flags` sections.

The 06-01-SUMMARY.md notes one auto-fixed implementation issue (nonce length validation added to `decrypt` to prevent GCM panic on invalid input). This is a defensive improvement consistent with T-06-01 and T-06-05; it does not constitute a new attack surface.

---

## Verification Methodology

Each `mitigate` threat was verified by direct code inspection of the cited implementation file:

- **T-06-01**: `cipher.NewGCM` + `gcm.Seal`/`gcm.Open` confirm authenticated encryption. Any byte modification to ciphertext or the GCM tag causes `gcm.Open` to return an error.
- **T-06-02**: The only `slog` call in `retrieve_share.go` logs three safe fields (`user_id`, `share_index`, `node_id`). No log statement references `share.EncryptedData`, `share.Nonce`, `plaintext`, or `s.encryptionKey`.
- **T-06-03**: All three SQL statements in `share.go` use positional parameters exclusively. No string interpolation present.
- **T-06-05**: Nonce allocation uses `gcm.NonceSize()` (not a literal), and `crypto/rand.Read` fills it. No deterministic nonce derivation path exists.
- **T-06-06**: `subtle.ConstantTimeCompare` is imported from `crypto/subtle` and called on line 36 of `interceptors.go`. All three rejection branches (`!ok`, `len(values)==0 || values[0]==""`, compare != 1) return `codes.Unauthenticated`.
- **T-06-07**: All three rejection status messages in `interceptors.go` are static strings with no context from the request or internal state.
- **T-06-08**: The health-check bypass is a single exact string comparison (`info.FullMethod == "/grpc.health.v1.Health/Check"`), evaluated before any metadata access.
- **T-06-09**: `bootstrap.go:34` checks `len(key) != 32` and returns a `fmt.Errorf` before constructing the service, causing `main.go` to halt startup.
- **T-06-10**: Handler error paths in all three API files return only static `status.Error` messages. The underlying `err` value is never forwarded to the caller.
- **T-06-11**: `store_share.go` handler checks `GetUserId() == ""`, `len(GetShareData()) == 0`, and `GetShareIndex() < 0`. `retrieve_share.go` checks `GetUserId() == ""` and `GetShareIndex() < 0`. `delete_share.go` checks `GetUserId() == ""`.
