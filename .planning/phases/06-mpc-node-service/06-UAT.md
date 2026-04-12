---
status: complete
phase: 06-mpc-node-service
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md]
started: 2026-04-12T14:00:00Z
updated: 2026-04-12T14:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Unit tests pass
expected: Run `cd mpc && go test ./...` — all 25 tests pass with 0 failures
result: pass

### 2. AES-256-GCM encryption roundtrip
expected: Encrypt then decrypt returns original plaintext. Each encrypt produces a unique 12-byte nonce via crypto/rand. Wrong key fails decryption. Corrupted ciphertext fails decryption. Invalid nonce length returns error (not panic).
result: pass

### 3. StoreShare service method
expected: Encrypts share data, generates UUID share ID, persists via storage. Duplicate (user_id, share_index) returns ErrDuplicateShare. Empty share data is accepted (GCM tag still produced).
result: pass

### 4. RetrieveShare service method
expected: Fetches encrypted share from storage, decrypts, returns plaintext bytes. Non-existent share returns ErrShareNotFound. Corrupted encrypted data returns decryption error.
result: pass

### 5. DeleteShare service method
expected: Deletes all shares for user_id from node. Returns count of deleted rows. Returns 0 with no error if none exist (idempotent).
result: pass

### 6. Auth interceptor (shared-secret)
expected: Valid shared secret in `authorization` metadata passes through. Wrong secret returns Unauthenticated. Missing metadata returns Unauthenticated. Empty authorization value returns Unauthenticated. Health check (`/grpc.health.v1.Health/Check`) bypasses auth. Uses constant-time comparison (subtle.ConstantTimeCompare).
result: pass

### 7. gRPC handler input validation
expected: StoreShare rejects empty user_id, empty share_data, negative share_index (InvalidArgument). RetrieveShare rejects empty user_id, negative share_index. DeleteShare rejects empty user_id. Error messages are generic (no internal details leaked).
result: pass

### 8. Bootstrap encryption key validation
expected: NewMPCService returns error if encryption key is not exactly 32 bytes. Service refuses to start with invalid key length.
result: pass

### 9. Clean Architecture: interfaces aligned with auth service pattern
expected: API layer defines Service interface (not concrete *MPCService). Service layer defines Storage interface (not concrete *PGStorage). Domain errors (ErrDuplicateShare, ErrShareNotFound) live in models package, not pgstorage. No pgstorage imports in API or service layers.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]
