---
phase: 12-add-dockerfiles-update-makefiles-with-help-command-add-root-
plan: 03
subsystem: infrastructure
tags: [docker, docker-compose, infrastructure, orchestration]
dependency_graph:
  requires: [12-01-env-var-config]
  provides: [root-docker-compose, mpc-init-script]
  affects: [docker-compose.yml, scripts/init-mpc-dbs.sql]
tech_stack:
  added: []
  patterns: [docker-compose-orchestration, shared-infrastructure, per-node-config]
key_files:
  created:
    - docker-compose.yml
    - scripts/init-mpc-dbs.sql
  modified: []
decisions:
  - "Shared Kafka instance for all services (single KRaft broker, no Zookeeper)"
  - "Shared Redis with db-number isolation (auth=0, twofa=1)"
  - "Shared PostgreSQL for 3 MPC nodes with init script creating additional databases"
  - "Separate PostgreSQL instances for auth and twofa services"
metrics:
  duration: 147s
  completed: 2026-04-14
  tasks_completed: 2
  tasks_total: 2
---

# Phase 12 Plan 03: Root Docker Compose Summary

Root docker-compose.yml orchestrates the full MPC-2FA system with 10 services, enabling single-command startup via `docker compose up`.

## One-liner

Root docker-compose with 5 infrastructure (kafka, 3x postgres, redis) and 5 application services (auth, twofa, 3x mpc-node), shared secrets, health checks, and MPC multi-database init script.

## Changes Made

### Task 1: Create MPC init script

- Created `scripts/init-mpc-dbs.sql` with CREATE DATABASE for mpc_db_2 and mpc_db_3
- mpc_db_1 is created automatically by POSTGRES_DB env var on the mpc-postgres container
- Script runs via `/docker-entrypoint-initdb.d/` volume mount on first container start

**Commit:** `e83ec3e` feat(12-03): add MPC database init script for 3 nodes

### Task 2: Create root docker-compose.yml

**Infrastructure services (5):**
- `kafka` -- bitnami/kafka:4.1, KRaft mode, advertised as kafka:9092 (container hostname)
- `auth-postgres` -- postgres:17, auth_db/auth_user, health check
- `twofa-postgres` -- postgres:17, twofa_db/twofa_user, health check
- `mpc-postgres` -- postgres:17, mpc_db_1/mpc_user, init script mount, health check
- `redis` -- redis:8, appendonly, health check

**Application services (5):**
- `auth` -- builds from auth/Dockerfile, depends on auth-postgres + redis + kafka, mounts keys volume
- `twofa` -- builds from root context with twofa/Dockerfile (for mpc proto access), depends on all infrastructure + 3 MPC nodes
- `mpc-node-1` -- port 9200, node_id=1, mpc_db_1
- `mpc-node-2` -- port 9201, node_id=2, mpc_db_2, different encryption key
- `mpc-node-3` -- port 9202, node_id=3, mpc_db_3, different encryption key

**Key configuration:**
- Each MPC node has unique encryption key (AES-256-GCM at-rest security)
- SHARED_SECRET identical between twofa and all mpc-nodes
- Redis db isolation: auth uses db=0, twofa uses db=1
- `docker compose config --quiet` validates without errors

**Commit:** `1f8f0df` feat(12-03): add root docker-compose.yml for full system orchestration

## Deviations from Plan

None -- plan executed exactly as written.

## Verification

- `docker compose config --quiet` exits 0 (valid syntax)
- Init script contains CREATE DATABASE mpc_db_2 and mpc_db_3
- All 10 services defined with correct environment variables
- All acceptance criteria verified programmatically

## Self-Check: PASSED

All 2 files found (docker-compose.yml, scripts/init-mpc-dbs.sql). All 2 commits verified (e83ec3e, 1f8f0df).
