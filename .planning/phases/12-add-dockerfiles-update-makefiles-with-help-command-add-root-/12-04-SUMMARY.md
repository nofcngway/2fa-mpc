---
phase: 12-add-dockerfiles-update-makefiles-with-help-command-add-root-
plan: 04
subsystem: build-tooling
tags: [makefile, help, docker, lint, mock]
dependency_graph:
  requires: [12-02, 12-03]
  provides: [per-service-makefiles, root-makefile]
  affects: [auth/Makefile, twofa/Makefile, mpc/Makefile, Makefile]
tech_stack:
  added: []
  patterns: [auto-help-makefile, self-documenting-targets]
key_files:
  created:
    - Makefile
  modified:
    - auth/Makefile
    - twofa/Makefile
    - mpc/Makefile
decisions:
  - "Auth generate-mocks renamed to mock for consistency across services"
  - "Added lint-all to root Makefile as natural companion to test-all"
  - "TwoFA docker-build uses cd .. for root context (multi-module build)"
metrics:
  duration: 541s
  completed: 2026-04-14
---

# Phase 12 Plan 04: Update Makefiles with Help Command and Root Makefile Summary

Self-documenting Makefiles with auto-help, Docker, mock, and lint targets across all services plus root-level system-wide operations Makefile.

## Task Results

### Task 1: Update per-service Makefiles with help, docker, mock, lint targets
**Commit:** f5d508d
**Files:** auth/Makefile, twofa/Makefile, mpc/Makefile

Updated all 3 per-service Makefiles with:
- `.DEFAULT_GOAL := help` and grep/awk auto-help pattern
- `## description` comments on all targets for self-documenting help output
- `docker-build` and `docker-run` targets (twofa uses `cd ..` for root context)
- `mock` target (auth: renamed from generate-mocks, added EventProducer; twofa/mpc: go generate)
- `lint` target (go vet + optional golangci-lint)

Auth-specific: renamed `generate-mocks` to `mock`, added EventProducer mock generation alongside Storage and SessionStorage.

### Task 2: Create root Makefile
**Commit:** 992276b
**Files:** Makefile (new)

Created root Makefile with system-wide targets:
- `up` / `down` for docker compose lifecycle
- `build-all` to build all 3 service Docker images
- `test-all` and `lint-all` for cross-service validation
- Self-documenting `help` as default goal

## Deviations from Plan

None - plan executed exactly as written.

## Verification

All 4 `make help` commands verified (auth, twofa, mpc, root) - exit 0 with formatted colored target lists.

## Self-Check: PASSED

All files exist, all commits verified.
