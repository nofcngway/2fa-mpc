---
phase: 13
slug: add-kafka-ui-kafbat-kafka-ui-to-docker-compose
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | docker compose (config validation) |
| **Config file** | docker-compose.yml, auth/docker-compose.yaml, twofa/docker-compose.yaml, mpc/docker-compose.yaml |
| **Quick run command** | `docker compose config --quiet` |
| **Full suite command** | `docker compose config --quiet && docker compose -f auth/docker-compose.yaml config --quiet && docker compose -f twofa/docker-compose.yaml config --quiet && docker compose -f mpc/docker-compose.yaml config --quiet` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `docker compose config --quiet`
- **After wave complete:** Run full suite command (all 4 compose files)

---

## Validation Architecture

This phase adds a kafka-ui service to 4 docker-compose files. Validation focuses on:

1. **Compose syntax validity** — `docker compose config` exits 0 for all files
2. **Service definition** — kafka-ui service present with correct image, ports, and environment
3. **Network connectivity** — kafka-ui depends_on kafka and connects to correct bootstrap servers
4. **Port mapping** — host 8090 mapped to container 8080

No unit tests or integration tests needed — this is pure infrastructure configuration.
