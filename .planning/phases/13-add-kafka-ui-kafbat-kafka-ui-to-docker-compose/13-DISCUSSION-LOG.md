# Phase 13: Add Kafka UI (kafbat/kafka-ui) to docker-compose - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 13-add-kafka-ui-kafbat-kafka-ui-to-docker-compose
**Areas discussed:** Host port for UI, Scope: root-only vs per-service compose, Root Makefile integration

---

## Host port for UI

| Option | Description | Selected |
|--------|-------------|----------|
| 8080 | Standard default, easy to remember | |
| 8090 | Avoids conflicts, stays in 8xxx range | ✓ |
| You decide | Claude's discretion | |

**User's choice:** 8090
**Notes:** Avoids conflicts with common 8080 usage by other dev tools

---

## Scope: root-only vs per-service compose

| Option | Description | Selected |
|--------|-------------|----------|
| Root docker-compose.yml only | System-wide tool, per-service composes stay minimal | |
| Root + all per-service composes | Available in both full-system and isolated dev modes | ✓ |
| Root + only per-service composes that have Kafka | All three have Kafka, effectively same as option 2 | |

**User's choice:** Root + all per-service composes
**Notes:** None

---

## Root Makefile integration

| Option | Description | Selected |
|--------|-------------|----------|
| Just part of `make up` | Starts with everything else, no separate target | ✓ |
| Dedicated `make kafka-ui` target | Start independently when needed | |
| Both | Starts with `make up` and also standalone target | |
| You decide | Claude's discretion | |

**User's choice:** Just part of `make up`
**Notes:** None

---

## Claude's Discretion

- Exact kafka-ui environment variable configuration
- depends_on configuration
- Version pinning strategy
- Placement in compose files

## Deferred Ideas

None
