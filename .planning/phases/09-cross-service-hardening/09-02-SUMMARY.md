---
phase: 09-cross-service-hardening
plan: 02
subsystem: kafka-audit-events
tags: [kafka, audit, event-producer, fire-and-forget, observability]
dependency_graph:
  requires: [09-01]
  provides: [kafka-audit-events, event-producer-interface]
  affects: [auth, twofa, mpc]
tech_stack:
  added: [segmentio/kafka-go]
  patterns: [fire-and-forget-events, NoOpProducer-fallback, per-service-EventProducer]
key_files:
  created:
    - auth/internal/services/authService/audit.go
    - auth/internal/bootstrap/kafka.go
    - auth/internal/services/authService/mocks/event_producer_mock.go
    - twofa/internal/services/twofaService/audit.go
    - twofa/internal/bootstrap/kafka.go
    - twofa/internal/services/twofaService/mocks/event_producer_mock.go
    - mpc/internal/services/mpcService/audit.go
    - mpc/internal/bootstrap/kafka.go
    - mpc/internal/services/mpcService/mocks/event_producer_mock.go
  modified:
    - auth/internal/services/authService/auth_service.go
    - auth/internal/services/authService/register.go
    - auth/internal/services/authService/login.go
    - auth/internal/services/authService/refresh_token.go
    - auth/internal/services/authService/logout.go
    - auth/internal/bootstrap/auth_service.go
    - auth/cmd/app/main.go
    - auth/internal/services/authService/register_test.go
    - auth/internal/services/authService/login_test.go
    - auth/internal/services/authService/refresh_token_test.go
    - auth/internal/services/authService/logout_test.go
    - auth/internal/services/authService/logout_all_test.go
    - auth/internal/services/authService/validate_token_test.go
    - auth/internal/services/authService/jwt_test.go
    - twofa/internal/services/twofaService/twofa_service.go
    - twofa/internal/services/twofaService/setup.go
    - twofa/internal/services/twofaService/verify.go
    - twofa/internal/services/twofaService/disable.go
    - twofa/internal/services/twofaService/status.go
    - twofa/internal/bootstrap/bootstrap.go
    - twofa/cmd/app/main.go
    - twofa/internal/services/twofaService/setup_test.go
    - twofa/internal/services/twofaService/verify_test.go
    - twofa/internal/services/twofaService/disable_test.go
    - twofa/internal/services/twofaService/status_test.go
    - mpc/internal/services/mpcService/mpc_service.go
    - mpc/internal/services/mpcService/store_share.go
    - mpc/internal/services/mpcService/retrieve_share.go
    - mpc/internal/services/mpcService/delete_share.go
    - mpc/internal/bootstrap/bootstrap.go
    - mpc/cmd/app/main.go
    - mpc/internal/services/mpcService/store_share_test.go
    - mpc/internal/services/mpcService/retrieve_share_test.go
    - mpc/internal/services/mpcService/delete_share_test.go
    - mpc/internal/services/mpcService/encrypt_test.go
    - auth/go.mod
    - auth/go.sum
    - twofa/go.mod
    - twofa/go.sum
    - mpc/go.mod
    - mpc/go.sum
decisions:
  - "Per-service EventProducer interface (not shared) since each service is a separate Go module"
  - "MPC AuditEvent includes NodeID int field for node identification in audit trail"
  - "NoOpProducer fallback when Kafka brokers not configured (empty slice check)"
  - "KafkaProducer uses Async mode with LeastBytes balancer for minimal latency impact"
  - "Kafka Close() placed as step 2 in ordered shutdown (after gRPC, before Redis/PG)"
metrics:
  tasks_completed: 2
  tasks_total: 2
  files_created: 9
  files_modified: 35
  completed: "2026-04-12"
---

# Phase 9 Plan 2: Kafka Audit Events Summary

Kafka audit event publishing across all 3 services with fire-and-forget semantics, per-service EventProducer interface, and KafkaProducer/NoOpProducer implementations

## One-Liner

Fire-and-forget Kafka audit events for 12 operations across auth/twofa/mpc via per-service EventProducer interface with NoOpProducer fallback

## What Was Done

### Task 1: EventProducer Interface + Infrastructure (34f68dc)

Created EventProducer interface and AuditEvent struct in each service's package:
- `auth/internal/services/authService/audit.go` - EventProducer with PublishEvent/Close, AuditEvent with UserID/Operation/Timestamp/Status
- `twofa/internal/services/twofaService/audit.go` - Same structure as auth
- `mpc/internal/services/mpcService/audit.go` - Same but AuditEvent adds `NodeID int` field

Created KafkaProducer and NoOpProducer in each service's bootstrap:
- `auth/internal/bootstrap/kafka.go` - KafkaProducer wraps kafka.Writer (Async, LeastBytes balancer), NoOpProducer for fallback
- `twofa/internal/bootstrap/kafka.go` - Same pattern
- `mpc/internal/bootstrap/kafka.go` - Same pattern

Generated minimock mocks for all 3 services. Added `github.com/segmentio/kafka-go` v0.4.50 dependency to all go.mod files.

### Task 2: Service Wiring + Audit Events + Test Updates (8cc25c5)

Added `eventProducer EventProducer` field to all 3 service structs and updated constructors:
- `AuthService`: eventProducer as 3rd parameter
- `TwoFAService`: eventProducer as 4th parameter (after mpcClients)
- `MPCService`: eventProducer as 4th parameter (after nodeID)

Published audit events in 12 service methods (fire-and-forget with slog.Warn on error):

| Service | Operation | Event Name |
|---------|-----------|------------|
| Auth | Register | user.registered |
| Auth | Login | user.logged_in |
| Auth | RefreshToken | token.refreshed |
| Auth | RefreshToken (theft) | token.refresh_reuse_detected (status: alert) |
| Auth | Logout | user.logged_out |
| TwoFA | Setup | 2fa.setup |
| TwoFA | Verify | 2fa.verified |
| TwoFA | Disable | 2fa.disabled |
| TwoFA | GetStatus | 2fa.status_checked |
| MPC | StoreShare | share.stored |
| MPC | RetrieveShare | share.retrieved |
| MPC | DeleteShare | share.deleted |

Updated bootstrap wiring in all 3 services to accept and pass EventProducer. Updated main.go in all 3 services with KafkaProducer creation and ordered shutdown (Kafka Close as step 2, after gRPC stop, before Redis/PG close).

Updated all existing tests (17 test files) to pass mock EventProducer with `.Optional().Return(nil)` pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed jwt_test.go missing EventProducer argument**
- **Found during:** Task 2
- **Issue:** jwt_test.go was not listed in plan's read_first but calls NewAuthService directly, causing compilation failure after constructor change
- **Fix:** Added `&bootstrap.NoOpProducer{}` as EventProducer arg and imported bootstrap package
- **Files modified:** auth/internal/services/authService/jwt_test.go
- **Commit:** 8cc25c5

**2. [Rule 1 - Bug] Fixed minimock Optional().Return(nil) chain order**
- **Found during:** Task 2 test run
- **Issue:** Initial mock setup used `.Return(nil)` (mandatory expectation) instead of `.Optional().Return(nil)`, causing all error-path tests to fail when audit event was not published
- **Fix:** Changed all mock setups to `PublishEventMock.Optional().Return(nil)` and `CloseMock.Optional().Return(nil)` across all 17 test files
- **Files modified:** All test files in auth/twofa/mpc
- **Commit:** 8cc25c5

**3. [Rule 1 - Bug] Fixed encrypt_test.go missing EventProducer argument**
- **Found during:** Task 2 test run
- **Issue:** Internal encrypt_test.go (package mpcService, not mpcService_test) calls NewMPCService directly with 3 args instead of 4
- **Fix:** Added `nil` as 4th argument (EventProducer) since encrypt/decrypt methods do not use it
- **Files modified:** mpc/internal/services/mpcService/encrypt_test.go
- **Commit:** 8cc25c5

**4. [Rule 1 - Bug] Fixed shortEventProducer instances in refresh_token_test.go and validate_token_test.go**
- **Found during:** Task 2 test run
- **Issue:** Inline shortService instances (for expired token tests) created their own EventProducerMock with mandatory `.Return(nil)` instead of `.Optional().Return(nil)`
- **Fix:** Updated both shortEventProducer setups to use Optional pattern
- **Files modified:** auth/internal/services/authService/refresh_token_test.go, auth/internal/services/authService/validate_token_test.go
- **Commit:** 8cc25c5

## Verification

All tests pass across all 3 services:
- auth: 22 tests PASS (4.2s)
- mpc: 14 tests PASS (1.3s)
- twofa: 14 tests PASS (25.0s)

## Known Stubs

None. All audit events are wired to actual KafkaProducer (or NoOpProducer fallback).

## Self-Check: PASSED

All 9 created files verified. Both commits (34f68dc, 8cc25c5) verified.
