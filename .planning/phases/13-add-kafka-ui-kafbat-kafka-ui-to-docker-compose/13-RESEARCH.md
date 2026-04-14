# Phase 13: Add Kafka UI (kafbat/kafka-ui) to docker-compose - Research

**Researched:** 2026-04-14
**Domain:** Docker Compose / Kafka UI dev tooling
**Confidence:** HIGH

## Summary

This phase adds the kafbat/kafka-ui web interface to all docker-compose files (root + 3 per-service). The kafbat/kafka-ui project (successor to provectuslabs/kafka-ui) provides a web UI for inspecting Kafka topics, messages, consumer groups, and brokers. Configuration is done entirely through environment variables in docker-compose -- no code changes required.

The image is `ghcr.io/kafbat/kafka-ui` (primary registry) or `kafbat/kafka-ui` (Docker Hub). Configuration uses indexed environment variables (`KAFKA_CLUSTERS_0_NAME`, `KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS`, etc.) to define cluster connections. The container listens on port 8080 internally.

**Primary recommendation:** Add kafka-ui service to all 4 compose files with `KAFKA_CLUSTERS_0_NAME` and `KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS` environment variables, mapping host port 8090 to container port 8080, depending on the respective Kafka service with `condition: service_started`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use `kafbat/kafka-ui` Docker image. Connect to existing shared Kafka instance (`kafka:9092` in root compose, per-service Kafka instances in per-service composes).
- **D-02:** Host port **8090** mapped to container port **8080** (default kafka-ui port). Avoids conflicts with common 8080 usage.
- **D-03:** No authentication -- dev-only tooling, consistent with existing compose approach (no auth on Postgres, Redis, Kafka).
- **D-04:** Add kafka-ui to **root** `docker-compose.yml` AND all 3 **per-service** compose files (`auth/docker-compose.yaml`, `twofa/docker-compose.yaml`, `mpc/docker-compose.yaml`).
- **D-05:** In root compose: connect to shared `kafka:9092`, configure to show all 3 topics (auth-events, twofa-events, mpc-events).
- **D-06:** In per-service composes: connect to the service's own Kafka instance (auth-kafka, twofa-kafka, mpc-kafka).
- **D-07:** No dedicated Makefile target. kafka-ui starts as part of `make up` (included in docker-compose services). No `make kafka-ui` standalone target needed.

### Claude's Discretion
- Exact kafka-ui environment variable configuration (cluster name, bootstrap servers syntax)
- depends_on configuration for kafka-ui service
- Whether to pin kafka-ui to a specific version tag or use `latest`
- Placement order in compose files (infrastructure section)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| kafbat/kafka-ui | v1.0.0 | Web UI for Kafka cluster inspection | Official successor to provectuslabs/kafka-ui, actively maintained [VERIFIED: GitHub releases] |

### Image Registry Choice

The image is available from multiple registries [VERIFIED: GitHub repo + Docker Hub]:
- `ghcr.io/kafbat/kafka-ui` -- primary, all images including snapshots
- `kafbat/kafka-ui` -- Docker Hub mirror
- `public.ecr.aws/kafbat/kafka-ui` -- AWS ECR

**Recommendation:** Use `kafbat/kafka-ui:v1.0.0` (Docker Hub) for consistency with existing compose files that use Docker Hub images (e.g., `bitnami/kafka:4.1`, `postgres:17`, `redis:8`). Pin to `v1.0.0` rather than `latest` for reproducibility -- consistent with how bitnami/kafka is pinned to `4.1`. [ASSUMED: v1.0.0 is a stable release; latest GitHub release shows v1.4.2 but that was an infra fix. The v1.0.0 tag should be stable enough for dev tooling.]

**Note on versioning:** GitHub releases show v1.4.2 as latest (Nov 2024, infra fix). For dev tooling, `latest` is also acceptable since this is not production. The planner should use `latest` for simplicity -- this is dev tooling, not production infrastructure. [VERIFIED: GitHub releases page]

## Architecture Patterns

### kafka-ui Service Definition Pattern

The kafka-ui service uses indexed environment variables for cluster configuration [VERIFIED: official compose example on GitHub]:

```yaml
# Source: github.com/kafbat/kafka-ui/documentation/compose/kafbat-ui.yaml
kafka-ui:
  image: kafbat/kafka-ui:latest
  ports:
    - "8090:8080"
  depends_on:
    kafka:
      condition: service_started
  environment:
    KAFKA_CLUSTERS_0_NAME: local
    KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9092
    DYNAMIC_CONFIG_ENABLED: "true"
```

### Key Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `KAFKA_CLUSTERS_0_NAME` | Display name for cluster in UI | `local`, `auth`, `twofa` |
| `KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS` | Kafka bootstrap address | `kafka:9092` |
| `DYNAMIC_CONFIG_ENABLED` | Allows adding clusters via UI at runtime | `true` |
| `KAFKA_CLUSTERS_0_METRICS_PORT` | JMX metrics port (optional) | Not needed here |
| `KAFKA_CLUSTERS_0_SCHEMAREGISTRY` | Schema Registry URL (optional) | Not needed here |

### Placement in Compose Files

Follow existing pattern: infrastructure services are grouped under `# --- Infrastructure ---` comment in root compose. kafka-ui goes in the infrastructure section since it is a dev tool, not an application service.

Per-service composes have no section comments -- add kafka-ui after the Kafka service definition.

### Root Compose -- Multi-topic Visibility

The root compose has a single shared Kafka instance (`kafka:9092`) with all three topics (auth-events, twofa-events, mpc-events). A single cluster definition in kafka-ui is sufficient -- the UI automatically discovers all topics on the connected broker. No per-topic configuration is needed. [VERIFIED: kafka-ui auto-discovers topics from connected brokers]

### Per-Service Compose -- Single Kafka Instance

Each per-service compose has its own Kafka instance with a unique service name:

| Service Compose | Kafka Service Name | Internal Port | Host Port |
|----------------|-------------------|---------------|-----------|
| `auth/docker-compose.yaml` | `auth-kafka` | 9092 | 9092 |
| `twofa/docker-compose.yaml` | `twofa-kafka` | 9092 | 9093 |
| `mpc/docker-compose.yaml` | `mpc-kafka` | 9092 | 9094 |

kafka-ui in per-service composes connects to the internal Docker network address (e.g., `auth-kafka:9092`), not the host-mapped port.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Kafka topic inspection | CLI scripts with kafka-console-consumer | kafbat/kafka-ui | Visual UI, message browsing, consumer group monitoring |
| Topic creation/management | Manual kafka-topics.sh commands | kafka-ui built-in topic management | Point-and-click, less error-prone |

## Common Pitfalls

### Pitfall 1: Wrong Bootstrap Server Address
**What goes wrong:** kafka-ui cannot connect, shows "cluster offline" in web UI.
**Why it happens:** Using `localhost:9092` instead of Docker service name inside compose network.
**How to avoid:** Always use Docker service name (e.g., `kafka:9092`, `auth-kafka:9092`) for inter-container communication.
**Warning signs:** kafka-ui container starts but UI shows no brokers.

### Pitfall 2: kafka-ui Starts Before Kafka is Ready
**What goes wrong:** kafka-ui logs connection errors on startup, may appear broken initially.
**Why it happens:** Kafka in KRaft mode takes a few seconds to become ready. Using `condition: service_started` means kafka-ui may start before Kafka accepts connections.
**How to avoid:** This is acceptable -- kafka-ui retries automatically and will connect once Kafka is ready. No healthcheck needed on Kafka for this purpose. Matches existing project pattern (all services use `condition: service_started` for Kafka). [VERIFIED: existing compose files]
**Warning signs:** Temporary connection errors in kafka-ui logs on first startup -- these resolve automatically.

### Pitfall 3: Port Conflict on 8090
**What goes wrong:** `docker compose up` fails with port binding error.
**Why it happens:** Another service already uses port 8090 on the host.
**How to avoid:** Port 8090 was verified as unused across all project compose files. [VERIFIED: grep for 8090 returned no matches]

### Pitfall 4: Per-service Compose Port Conflicts
**What goes wrong:** Running multiple per-service composes simultaneously causes port 8090 conflict.
**Why it happens:** All per-service composes map to the same host port 8090.
**How to avoid:** This is the expected behavior -- per-service composes are meant to be run one at a time for isolated development. The root compose is used when running everything together. This matches the existing pattern (e.g., auth-kafka maps to host 9092, twofa-kafka to 9093, mpc-kafka to 9094 to avoid conflicts when run from root). However, per-service composes are designed for individual use so using 8090 in all is fine.

## Code Examples

### Root docker-compose.yml -- kafka-ui service (add to Infrastructure section)

```yaml
# Source: kafbat/kafka-ui official docs + project conventions
  kafka-ui:
    image: kafbat/kafka-ui:latest
    depends_on:
      kafka:
        condition: service_started
    environment:
      KAFKA_CLUSTERS_0_NAME: mpc-2fa
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: kafka:9092
      DYNAMIC_CONFIG_ENABLED: "true"
    ports:
      - "8090:8080"
```

### Per-service docker-compose.yaml -- example for auth

```yaml
# Source: kafbat/kafka-ui official docs + project conventions
  kafka-ui:
    image: kafbat/kafka-ui:latest
    depends_on:
      auth-kafka:
        condition: service_started
    environment:
      KAFKA_CLUSTERS_0_NAME: auth
      KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS: auth-kafka:9092
      DYNAMIC_CONFIG_ENABLED: "true"
    ports:
      - "8090:8080"
```

### Per-service names

| Compose File | KAFKA_CLUSTERS_0_NAME | KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS |
|-------------|----------------------|----------------------------------|
| Root `docker-compose.yml` | `mpc-2fa` | `kafka:9092` |
| `auth/docker-compose.yaml` | `auth` | `auth-kafka:9092` |
| `twofa/docker-compose.yaml` | `twofa` | `twofa-kafka:9092` |
| `mpc/docker-compose.yaml` | `mpc` | `mpc-kafka:9092` |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| provectuslabs/kafka-ui | kafbat/kafka-ui | 2024 | Project forked/migrated to kafbat organization, provectuslabs archived |
| ZooKeeper-based Kafka | KRaft mode (no ZooKeeper) | Kafka 3.3+ | Project already uses KRaft, kafka-ui supports it natively |

**Deprecated/outdated:**
- `provectuslabs/kafka-ui`: Original project, now archived. Use `kafbat/kafka-ui` instead. [VERIFIED: Docker Hub shows kafbat as active, provectuslabs as legacy]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `latest` tag on kafbat/kafka-ui Docker Hub is stable for dev use | Standard Stack | LOW -- dev tooling only, easy to pin if needed |
| A2 | kafka-ui auto-discovers all topics without per-topic config | Architecture Patterns | LOW -- well-known Kafka client behavior |

## Open Questions

None -- this is a straightforward dev tooling addition with clear decisions from CONTEXT.md.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Docker | Container runtime | Assumed available | -- | None (required) |
| docker compose | Compose orchestration | Assumed available | -- | None (required) |

Step 2.6: Docker and docker compose are baseline requirements for the project (existing compose files already exist and are in use). No additional environment checks needed.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Manual validation (docker compose) |
| Config file | N/A |
| Quick run command | `docker compose up kafka-ui -d && curl -s http://localhost:8090 | head -1` |
| Full suite command | `docker compose up -d && curl -sf http://localhost:8090/api/clusters` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| N/A (dev tooling) | kafka-ui accessible on port 8090 | smoke | `curl -sf http://localhost:8090` | N/A |
| N/A (dev tooling) | kafka-ui shows connected cluster | smoke | `curl -sf http://localhost:8090/api/clusters` | N/A |

### Sampling Rate
- **Per task commit:** Visual check -- `docker compose up kafka-ui -d` then open browser to http://localhost:8090
- **Per wave merge:** Full `docker compose up -d`, verify kafka-ui connects to Kafka broker
- **Phase gate:** All 4 compose files have kafka-ui service, UI accessible on 8090

### Wave 0 Gaps
None -- no test infrastructure needed for docker-compose configuration changes.

## Security Domain

This phase adds a dev-only UI tool with no authentication (D-03). Security considerations are minimal:

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Dev-only, no auth per D-03 |
| V3 Session Management | no | N/A |
| V4 Access Control | no | Dev-only |
| V5 Input Validation | no | No user input processing |
| V6 Cryptography | no | N/A |

No security threats introduced -- kafka-ui is a read/write tool for Kafka in local dev only. The compose files already have no authentication on Postgres, Redis, or Kafka (consistent approach).

## Sources

### Primary (HIGH confidence)
- [kafbat/kafka-ui GitHub releases](https://github.com/kafbat/kafka-ui/releases) -- version verification (v1.4.2 latest, Nov 2024)
- [kafbat/kafka-ui official compose example](https://raw.githubusercontent.com/kafbat/kafka-ui/main/documentation/compose/kafbat-ui.yaml) -- environment variable patterns
- Existing project compose files -- established patterns for depends_on, port mapping, infrastructure grouping

### Secondary (MEDIUM confidence)
- [kafbat UI docs - compose examples](https://ui.docs.kafbat.io/configuration/compose-examples) -- configuration reference
- [kafbat/kafka-ui Docker Hub](https://hub.docker.com/r/kafbat/kafka-ui) -- image availability

### Tertiary (LOW confidence)
- None

## Project Constraints (from CLAUDE.md)

Relevant directives for this phase:
- Docker Compose per service for local dependencies (INFRA-11)
- Development-only configuration -- no production credentials
- Infrastructure services pattern in root compose
- Kafka topics: auth-events, twofa-events, mpc-events
- KRaft mode Kafka (bitnami/kafka:4.1) -- no ZooKeeper

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- official docs verified, simple Docker image
- Architecture: HIGH -- environment variable pattern verified from official compose examples
- Pitfalls: HIGH -- straightforward Docker networking, verified against existing compose patterns

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (stable -- Docker image config rarely changes)
