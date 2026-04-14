# Phase 13: Add Kafka UI (kafbat/kafka-ui) to docker-compose - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Add kafbat/kafka-ui web interface for inspecting Kafka topics, messages, and consumer groups. Added to root docker-compose.yml and all per-service docker-compose files.

</domain>

<decisions>
## Implementation Decisions

### Service Configuration
- **D-01:** Use `kafbat/kafka-ui` Docker image. Connect to existing shared Kafka instance (`kafka:9092` in root compose, per-service Kafka instances in per-service composes).
- **D-02:** Host port **8090** mapped to container port **8080** (default kafka-ui port). Avoids conflicts with common 8080 usage.
- **D-03:** No authentication — dev-only tooling, consistent with existing compose approach (no auth on Postgres, Redis, Kafka).

### Scope
- **D-04:** Add kafka-ui to **root** `docker-compose.yml` AND all 3 **per-service** compose files (`auth/docker-compose.yaml`, `twofa/docker-compose.yaml`, `mpc/docker-compose.yaml`).
- **D-05:** In root compose: connect to shared `kafka:9092`, configure to show all 3 topics (auth-events, twofa-events, mpc-events).
- **D-06:** In per-service composes: connect to the service's own Kafka instance (auth-kafka, twofa-kafka, mpc-kafka).

### Makefile
- **D-07:** No dedicated Makefile target. kafka-ui starts as part of `make up` (included in docker-compose services). No `make kafka-ui` standalone target needed.

### Claude's Discretion
- Exact kafka-ui environment variable configuration (cluster name, bootstrap servers syntax)
- depends_on configuration for kafka-ui service
- Whether to pin kafka-ui to a specific version tag or use `latest`
- Placement order in compose files (infrastructure section)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Docker Compose Files
- `docker-compose.yml` — Root compose with shared Kafka on `kafka:9092`, all services
- `auth/docker-compose.yaml` — Auth per-service compose, Kafka as `auth-kafka`
- `twofa/docker-compose.yaml` — TwoFA per-service compose, Kafka as `twofa-kafka`
- `mpc/docker-compose.yaml` — MPC per-service compose, Kafka as `mpc-kafka`

### Prior Phase Context
- `.planning/phases/12-add-dockerfiles-update-makefiles-with-help-command-add-root-/12-CONTEXT.md` — Phase 12 decisions on docker-compose structure, Kafka topics, port conventions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Root `docker-compose.yml` already has Kafka (`bitnami/kafka:4.1`) with KRaft mode configured
- Per-service composes each have their own Kafka instance with same `bitnami/kafka:4.1` image

### Established Patterns
- Infrastructure services grouped under `# --- Infrastructure ---` comment in root compose
- Services use `condition: service_started` for Kafka dependency (no healthcheck on Kafka)
- Per-service Kafka instances use different advertised listener ports (9092, 9093, 9094)

### Integration Points
- Root compose: kafka-ui connects to `kafka:9092` (internal Docker network)
- `auth/docker-compose.yaml`: kafka-ui connects to `auth-kafka:9092`
- `twofa/docker-compose.yaml`: kafka-ui connects to `twofa-kafka:9092`
- `mpc/docker-compose.yaml`: kafka-ui connects to `mpc-kafka:9092`

</code_context>

<specifics>
## Specific Ideas

- Port 8090 chosen to avoid conflicts with common 8080 usage by other dev tools

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 13-add-kafka-ui-kafbat-kafka-ui-to-docker-compose*
*Context gathered: 2026-04-14*
