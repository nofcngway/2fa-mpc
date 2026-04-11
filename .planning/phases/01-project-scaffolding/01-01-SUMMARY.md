---
phase: 01-project-scaffolding
plan: 01
subsystem: auth
tags: [proto, grpc, tooling, go-module]
dependency_graph:
  requires: []
  provides: [auth-proto-contract, auth-go-module, auth-build-tooling]
  affects: [01-04-auth-skeleton]
tech_stack:
  added: [protobuf, grpc-go, pgx-v5, redis-go-v9, kafka-go, jwt-v5, prometheus, uuid, yaml-v3, x-crypto]
  patterns: [proto-code-generation, makefile-build-targets]
key_files:
  created:
    - auth/go.mod
    - auth/go.sum
    - auth/api/models/models.proto
    - auth/api/auth_api/auth_service.proto
    - auth/scripts/generate.sh
    - auth/internal/pb/models/models.pb.go
    - auth/internal/pb/auth_api/auth_service.pb.go
    - auth/internal/pb/auth_api/auth_service_grpc.pb.go
    - auth/Makefile
    - auth/.gitignore
  modified: []
decisions:
  - Proto go_package uses full module paths (github.com/vbncursed/vkr/auth/internal/pb/*) for protoc-gen-go compatibility
metrics:
  duration: 2m 25s
  completed: "2026-04-11T19:09:40Z"
---

# Phase 01 Plan 01: Auth Proto and Module Setup Summary

Auth Go module with proto definitions (5 RPCs, 2 models), protoc code generation via generate.sh, and Makefile with RSA key generation target.

## What Was Done

### Task 1: Create Auth Go module, proto definitions, and generate.sh
- Initialized `github.com/vbncursed/vkr/auth` Go module with all required dependencies
- Created `auth/api/models/models.proto` with User and TokenPair messages
- Created `auth/api/auth_api/auth_service.proto` with 5 RPCs: Register, Login, RefreshToken, Logout, ValidateToken
- Created `auth/scripts/generate.sh` for protoc compilation to `internal/pb/`
- Generated Go protobuf code (3 files: models.pb.go, auth_service.pb.go, auth_service_grpc.pb.go)
- **Commit:** c492d14

### Task 2: Create Auth Makefile and .gitignore
- Created Makefile with targets: generate, build, run, test, clean, generate-keys, tidy
- `generate-keys` target creates RSA-2048 keys in `auth/keys/` for JWT RS256 signing
- Created `.gitignore` to exclude `keys/` and `bin/` directories
- **Commit:** 251e576

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed proto go_package paths for protoc-gen-go compatibility**
- **Found during:** Task 1
- **Issue:** Plan specified `option go_package = "models"` and `"auth_api"` which protoc-gen-go rejects (requires at least one '.' or '/' in import path)
- **Fix:** Changed to full module paths: `github.com/vbncursed/vkr/auth/internal/pb/models` and `github.com/vbncursed/vkr/auth/internal/pb/auth_api`
- **Files modified:** auth/api/models/models.proto, auth/api/auth_api/auth_service.proto
- **Commit:** c492d14

## Verification Results

1. `bash scripts/generate.sh` -- produces valid .pb.go files in internal/pb/
2. `go.mod` has correct module path: github.com/vbncursed/vkr/auth
3. Proto defines exactly 5 RPCs (Register, Login, RefreshToken, Logout, ValidateToken)
4. Makefile has all required targets including generate-keys
5. .gitignore covers keys/ and bin/

## Self-Check: PASSED

- [x] auth/go.mod exists
- [x] auth/api/models/models.proto exists
- [x] auth/api/auth_api/auth_service.proto exists
- [x] auth/scripts/generate.sh exists
- [x] auth/internal/pb/models/models.pb.go exists
- [x] auth/internal/pb/auth_api/auth_service.pb.go exists
- [x] auth/internal/pb/auth_api/auth_service_grpc.pb.go exists
- [x] auth/Makefile exists
- [x] auth/.gitignore exists
- [x] Commit c492d14 exists
- [x] Commit 251e576 exists
