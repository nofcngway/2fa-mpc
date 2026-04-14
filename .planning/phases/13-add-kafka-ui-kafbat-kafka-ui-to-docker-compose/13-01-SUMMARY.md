---
phase: 13-add-kafka-ui-kafbat-kafka-ui-to-docker-compose
plan: 01
subsystem: infrastructure
tags: [docker-compose, kafka-ui, dev-tooling]
dependency_graph:
  requires: [docker-compose.yml, auth/docker-compose.yaml, twofa/docker-compose.yaml, mpc/docker-compose.yaml]
  provides: [kafka-ui-dev-access]
  affects: [docker-compose]
tech_stack:
  added: [kafbat/kafka-ui]
  patterns: [depends_on-with-condition]
key_files:
  modified:
    - docker-compose.yml
    - auth/docker-compose.yaml
    - twofa/docker-compose.yaml
    - mpc/docker-compose.yaml
decisions:
  - Cluster names match service context (mpc-2fa for root, auth/twofa/mpc for per-service)
  - DYNAMIC_CONFIG_ENABLED for runtime flexibility in all instances
metrics:
  duration: 115s
  completed: 2026-04-14T10:37:30Z
  tasks_completed: 2
  tasks_total: 2
  files_modified: 4
---

# Phase 13 Plan 01: Add Kafka UI to Docker Compose Summary

kafbat/kafka-ui added to all 4 docker-compose files with correct Kafka bootstrap server connections and port 8090 mapping for development topic inspection.

## Task Results

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add kafka-ui to root docker-compose.yml | bcb0928 | docker-compose.yml |
| 2 | Add kafka-ui to per-service docker-compose files | f3e0fb1 | auth/docker-compose.yaml, twofa/docker-compose.yaml, mpc/docker-compose.yaml |

## What Changed

### Root docker-compose.yml
- Added `kafka-ui` service in Infrastructure section (after redis, before Application Services)
- Image: `kafbat/kafka-ui:latest`, port `8090:8080`
- Connects to shared `kafka:9092` with cluster name `mpc-2fa`
- `depends_on` kafka with `condition: service_started`

### Per-service docker-compose files
- **auth**: connects to `auth-kafka:9092`, cluster name `auth`
- **twofa**: connects to `twofa-kafka:9092`, cluster name `twofa`
- **mpc**: connects to `mpc-kafka:9092`, cluster name `mpc`
- All use same port mapping `8090:8080` (per-service composes run one at a time)
- All use `DYNAMIC_CONFIG_ENABLED: "true"`

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

All acceptance criteria passed:
- All 4 files contain exactly 1 `kafbat/kafka-ui` reference
- All 4 files map port `8090:8080`
- Root connects to `kafka:9092`, per-service files connect to their respective Kafka instances
- Cluster names correct: `mpc-2fa`, `auth`, `twofa`, `mpc`

## Self-Check: PASSED
