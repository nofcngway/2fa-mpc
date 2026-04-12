---
phase: 07
slug: twofa-setup-flow
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-12
---

# Phase 07 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| TwoFA Service -> MPC Nodes (gRPC) | Shares sent over network to remote MPC nodes | Shamir shares (sensitive) |
| Client -> TwoFA Service (gRPC) | user_id and email from untrusted caller | User identifiers |
| TwoFA Service -> PostgreSQL | Storage of 2FA records and backup code hashes | Hashed backup codes, 2FA metadata |
| Test seam (GenerateSecretFunc) | Exported package variable allows test-time replacement of secret generation | N/A (test-only) |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-07-01 (P01) | Spoofing | MPC gRPC calls | mitigate | `bootstrap.go:65-76` — `authMetadataInterceptor` attaches shared secret in "authorization" metadata via `grpc.WithUnaryInterceptor` | closed |
| T-07-02 (P01) | Information Disclosure | Memory | mitigate | `setup.go:50` — `defer crypto.Zeroize(raw)`; `setup.go:58-62` — deferred loop zeroing `shares[i].Data` | closed |
| T-07-03 (P01) | Tampering | SQL injection | mitigate | `twofa_record.go:14,24` — parameterized `$1`; `backup_code.go:13` — parameterized `$1,$2,$3` | closed |
| T-07-04 (P01) | Information Disclosure | Logging | mitigate | `setup.go:89` — slog.Info logs only "user_id"; `setup.go:139` — slog.Error logs only "node", "user_id", "error" | closed |
| T-07-05 (P01) | Denial of Service | Partial share storage | mitigate | `setup.go:119-122` — `deleteSharesFromAllNodes` called on errgroup failure across all 3 nodes | closed |
| T-07-06 (P01) | Tampering | Duplicate setup | mitigate | `setup.go:41-43` — is_enabled check returns `ErrAlreadyEnabled` | closed |
| T-07-01 (P02) | Information Disclosure | TOTP secret in memory | mitigate | `setup.go:50` — `defer crypto.Zeroize(raw)` immediately after `GenerateSecretFunc()` call | closed |
| T-07-02 (P02) | Information Disclosure | Share data in memory | mitigate | `setup.go:58-62` — deferred loop `crypto.Zeroize(shares[i].Data)` for all 3 shares | closed |
| T-07-03 (P02) | Information Disclosure | Logging | mitigate | slog calls log only user_id and node index, no secret or share bytes | closed |
| T-07-04 (P02) | Tampering | Duplicate 2FA setup | mitigate | `setup.go:41-43` — GetTwoFARecord check for is_enabled=true returns ErrAlreadyEnabled | closed |
| T-07-05 (P02) | Denial of Service | Orphaned shares | mitigate | `setup.go:131-142` — `deleteSharesFromAllNodes` uses `context.Background()` with mpcTimeout | closed |
| T-07-06 (P02) | Spoofing | Weak backup codes | mitigate | `backup_codes.go:21,25` — `crypto/rand` with `big.NewInt(10000)`, 10^8 keyspace | closed |
| T-07-07 (P02) | Information Disclosure | Backup code plaintext persistence | mitigate | `backup_codes.go:45` — bcrypt cost=12; only hashed codes stored | closed |
| T-07-08 (P02) | Spoofing | Input validation | mitigate | `api/twofa_service_api/setup.go:16` — empty user_id or email returns `codes.InvalidArgument` | closed |
| T-07-09 (P02) | Information Disclosure | gRPC error messages | mitigate | `api/twofa_service_api/setup.go:25` — `codes.Internal, "internal error"` for unknown errors | closed |
| T-07-10 (P03) | Tampering | GenerateSecretFunc exported variable | accept | See Accepted Risks Log | closed |
| T-07-11 (P03) | Information Disclosure | Secret zeroization verification | mitigate | `setup_test.go:TestSetup_SecretZeroized` captures raw slice via injected `GenerateSecretFunc`, asserts all 20 bytes are zero after Setup returns | closed |

*Status: open / closed*
*Disposition: mitigate (implementation required) / accept (documented risk) / transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-07-01 | T-07-10 | `GenerateSecretFunc` is an exported package-level variable for test-time injection only. Production binary uses default `totp.GenerateSecret`. Go test binaries compile separately — no runtime substitution risk. Alternative (unexported + internal test package) would require migrating all 18 existing tests. | Plan 07-03 / security audit | 2026-04-12 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-12 | 17 | 17 | 0 | gsd-security-auditor + manual fix |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-12
