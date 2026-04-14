---
phase: 12-add-dockerfiles-update-makefiles-with-help-command-add-root-
plan: 02
subsystem: docker
tags: [docker, containerization, multi-stage-build, scratch]
dependency_graph:
  requires: [12-01]
  provides: [auth-dockerfile, mpc-dockerfile, twofa-dockerfile, dockerignore-files]
  affects: [deployment, ci-cd]
tech_stack:
  added: [docker-multi-stage, scratch-runtime]
  patterns: [multi-stage-build, layer-caching, static-binary]
key_files:
  created:
    - auth/Dockerfile
    - auth/.dockerignore
    - mpc/Dockerfile
    - mpc/.dockerignore
    - twofa/Dockerfile
    - .dockerignore
  modified: []
decisions:
  - "TwoFA uses project root as Docker build context due to replace directive on mpc/"
  - "All images use scratch as runtime base for minimal attack surface (~10MB)"
  - "No config.yaml embedded in images -- all configuration via runtime env vars"
metrics:
  duration: 101s
  completed: 2026-04-14T09:11:18Z
  tasks_completed: 2
  tasks_total: 2
  files_created: 6
  files_modified: 0
---

# Phase 12 Plan 02: Create Multi-Stage Dockerfiles Summary

Multi-stage Dockerfiles for Auth, MPC, and TwoFA services using golang:1.26 builder and scratch runtime with static binaries and layer-cached dependency downloads.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Auth and MPC Dockerfiles + .dockerignore | e55e7df | auth/Dockerfile, auth/.dockerignore, mpc/Dockerfile, mpc/.dockerignore |
| 2 | TwoFA Dockerfile + root .dockerignore | dfafef2 | twofa/Dockerfile, .dockerignore |

## Implementation Details

### Auth and MPC Dockerfiles (Task 1)

Both follow identical pattern:
- **Builder stage**: `golang:1.26`, copies go.mod/go.sum first for layer caching, then full source
- **Build**: `CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/app`
- **Runtime stage**: `FROM scratch` with only the binary
- **Build context**: Each service's own directory (`auth/`, `mpc/`)

`.dockerignore` files exclude `bin/`, `*.md`, `.git/`, `docker-compose.yaml`, and `keys/` (auth only).

### TwoFA Dockerfile (Task 2)

Special handling due to `replace github.com/vbncursed/vkr/mpc => ../mpc` in go.mod:
- **Build context**: Project root (invoked as `docker build -f twofa/Dockerfile .`)
- **Copies both modules**: mpc/go.mod + twofa/go.mod first, then full mpc/ and twofa/ directories
- **WORKDIR switching**: Between `/build` and `/build/twofa` for correct module resolution

Root `.dockerignore` excludes `auth/`, `gateway/`, `migration/`, `monitoring/`, `workspace/`, `.obsidian/`, `.planning/`, `.git/`, `*.md` -- keeping only `mpc/` and `twofa/` in build context.

## Deviations from Plan

### Docker Build Verification

Docker daemon was not running on the build machine, so Docker image builds could not be verified. Go builds (`go build ./...`) were verified successfully for all 3 services, confirming the source code compiles correctly. Docker image builds should be verified when Docker is available.

## Threat Surface

| Mitigation | Status | Details |
|------------|--------|---------|
| T-12-03: .dockerignore excludes keys/, .planning/, .git/ | Implemented | auth/.dockerignore excludes keys/, root .dockerignore excludes .planning/ and .git/ |
| T-12-04: No config.yaml in image | Implemented | No COPY config.yaml in any Dockerfile |
