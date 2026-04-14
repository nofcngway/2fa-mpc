---
phase: 12-add-dockerfiles-update-makefiles-with-help-command-add-root-
plan: 01
subsystem: config
tags: [config, env-vars, docker, tdd]
dependency_graph:
  requires: []
  provides: [env-var-config-loading]
  affects: [auth/config, twofa/config, mpc/config]
tech_stack:
  added: []
  patterns: [env-var-override, optional-yaml-config]
key_files:
  created:
    - auth/config/config_test.go (rewritten with 7 TDD tests)
    - twofa/config/config_test.go (rewritten with 6 TDD tests)
    - mpc/config/config_test.go (rewritten with 6 TDD tests)
  modified:
    - auth/config/config.go
    - twofa/config/config.go
    - mpc/config/config.go
decisions:
  - "Helper functions (envString, envInt, envDuration, envStringSlice) duplicated per service rather than shared package -- keeps services independently deployable"
  - "Invalid env var values silently ignored (keep yaml/default) per T-12-02 mitigation"
  - "MPC Validate() method added as it was missing -- required for env-only mode to reject incomplete config"
metrics:
  duration: 299s
  completed: 2026-04-14
  tasks_completed: 2
  tasks_total: 2
---

# Phase 12 Plan 01: Config Env Var Override Support Summary

All 3 services (Auth, TwoFA, MPC) now support full environment variable configuration with AUTH_*, TWOFA_*, MPC_* prefixes, enabling Docker containers to run without config.yaml.

## One-liner

Env var override support for all config.go files with optional yaml, typed parsing (int, duration, string slice, MPC nodes), and 19 TDD tests.

## Changes Made

### Task 1: Auth config.go env var support (TDD)

- Added `envString`, `envInt`, `envDuration`, `envStringSlice` helper functions
- Added `applyEnvOverrides` with 13 AUTH_* env var mappings
- Rewrote `Load` to make yaml file optional (no error on missing file)
- 7 tests: yaml-only, env-only, env+yaml override, validation, brokers comma parsing, duration parsing, redis DB int parsing

**Commits:**
- `b41d53a` test(12-01): add failing tests for auth config env var support
- `d5448ab` feat(12-01): add env var override support to auth config

### Task 2: TwoFA and MPC config.go env var support (TDD)

**TwoFA:**
- Added same helper functions plus `envMPCNodes` for comma-separated MPC node address parsing with TrimSpace
- Added `applyEnvOverrides` with 12 TWOFA_* env var mappings
- Removed old 2-line TWOFA_SHARED_SECRET/TWOFA_DATABASE_DSN override block
- Rewrote `Load` to make yaml optional
- 6 tests: yaml-only, env-only, MPC nodes parsing, timeout duration, env override, validation

**MPC:**
- Added `envString`, `envInt`, `envStringSlice` helpers (no duration needed)
- Added `Validate()` method (was missing from original code)
- Added `applyEnvOverrides` with 9 MPC_* env var mappings
- Rewrote `Load` to make yaml optional and call Validate
- 6 tests: yaml-only, env-only, node ID parsing, env override, validation DSN, validation encryption key

**Commits:**
- `7a2b78a` test(12-01): add failing tests for twofa and mpc config env var support
- `487e497` feat(12-01): add env var override support to twofa and mpc config

## Env Var Mappings

### Auth (AUTH_* prefix)
| Env Var | Config Field |
|---------|-------------|
| AUTH_SERVER_PORT | Server.Port |
| AUTH_SERVER_METRICS_PORT | Server.MetricsPort |
| AUTH_SERVER_LOG_LEVEL | Server.LogLevel |
| AUTH_DATABASE_DSN | Database.DSN |
| AUTH_REDIS_ADDR | Redis.Addr |
| AUTH_REDIS_PASSWORD | Redis.Password |
| AUTH_REDIS_DB | Redis.DB |
| AUTH_KAFKA_BROKERS | Kafka.Brokers |
| AUTH_KAFKA_TOPIC | Kafka.Topic |
| AUTH_JWT_PRIVATE_KEY_PATH | JWT.PrivateKeyPath |
| AUTH_JWT_PUBLIC_KEY_PATH | JWT.PublicKeyPath |
| AUTH_JWT_ACCESS_TOKEN_TTL | JWT.AccessTokenTTL |
| AUTH_JWT_REFRESH_TOKEN_TTL | JWT.RefreshTokenTTL |

### TwoFA (TWOFA_* prefix)
| Env Var | Config Field |
|---------|-------------|
| TWOFA_SERVER_PORT | Server.Port |
| TWOFA_SERVER_METRICS_PORT | Server.MetricsPort |
| TWOFA_SERVER_LOG_LEVEL | Server.LogLevel |
| TWOFA_DATABASE_DSN | Database.DSN |
| TWOFA_REDIS_ADDR | Redis.Addr |
| TWOFA_REDIS_PASSWORD | Redis.Password |
| TWOFA_REDIS_DB | Redis.DB |
| TWOFA_KAFKA_BROKERS | Kafka.Brokers |
| TWOFA_KAFKA_TOPIC | Kafka.Topic |
| TWOFA_MPC_NODES | MPCNodes (comma-separated) |
| TWOFA_SHARED_SECRET | SharedSecret |
| TWOFA_MPC_TIMEOUT | MPCTimeout |

### MPC (MPC_* prefix)
| Env Var | Config Field |
|---------|-------------|
| MPC_SERVER_PORT | Server.Port |
| MPC_SERVER_METRICS_PORT | Server.MetricsPort |
| MPC_SERVER_LOG_LEVEL | Server.LogLevel |
| MPC_DATABASE_DSN | Database.DSN |
| MPC_KAFKA_BROKERS | Kafka.Brokers |
| MPC_KAFKA_TOPIC | Kafka.Topic |
| MPC_NODE_ID | Node.ID |
| MPC_NODE_ENCRYPTION_KEY | Node.EncryptionKey |
| MPC_SHARED_SECRET | SharedSecret |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing functionality] Added MPC Validate() method**
- **Found during:** Task 2
- **Issue:** MPC config.go had no Validate() method, meaning env-only mode could produce configs with empty DSN or encryption key
- **Fix:** Added Validate() checking server.port, database.dsn, node.encryption_key, shared_secret
- **Files modified:** mpc/config/config.go
- **Commit:** 487e497

## Verification

- All 3 services compile: auth, twofa, mpc `go build ./...` pass
- All config tests pass: 19 total (7 auth + 6 twofa + 6 mpc)
- All existing service tests still pass

## Self-Check: PASSED

All 6 files found. All 4 commits verified.
