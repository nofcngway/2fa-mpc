# Phase 1: Project Scaffolding - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Create runnable Clean Architecture skeletons for all 3 services (auth, twofa, mpc) with proto generation, config loading, Docker Compose infrastructure, and bootstrap DI wiring.

</domain>

<decisions>
## Implementation Decisions

### Proto-contracts
- **D-01:** Define full RPC methods and message types for each service from the start (per CLAUDE.md spec), but handler implementations return `codes.Unimplemented` — stubs that compile and register correctly
- **D-02:** Proto models match the domain models described in workspace docs (User, TokenPair, Share, etc.)

### Docker Compose
- **D-03:** One docker-compose.yaml per service (per CLAUDE.md: "Docker Compose per service for local dependencies") — each contains PostgreSQL and Redis as needed
- **D-04:** Kafka included in docker-compose where needed (auth, twofa, mpc all publish audit events per requirements) — single shared Kafka instance referenced across services is acceptable for local dev
- **D-05:** MPC node uses a single docker-compose with one PostgreSQL instance — 3 separate node instances are configured via different config.yaml files (different ports, node IDs, encryption keys)

### Config pattern
- **D-06:** Include all config sections from the start (server, database, redis, kafka, jwt/encryption as applicable) with sensible local defaults — avoids config refactoring in later phases
- **D-07:** RSA key paths in auth config.yaml point to `keys/` directory within auth service — keys generated manually or via Makefile target, NOT committed to repo

### Protobuf tooling
- **D-08:** Use classic `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` — simpler setup, no buf dependency, matches academic project scope
- **D-09:** Each service has its own `scripts/generate.sh` that generates from local `api/` directory into `internal/pb/`

### Bootstrap and DI
- **D-10:** Bootstrap layer creates real dependencies (PGStorage, Redis, Kafka) but services can start even if some are unavailable — log warnings, don't panic on optional deps like Kafka
- **D-11:** Interfaces defined in service files, implementations in storage — standard Go Clean Architecture pattern per ADR-005

### Claude's Discretion
- Proto field types, naming, and package structure
- Makefile targets and scripts organization
- Exact docker-compose service names and port mappings
- initTables SQL schema details (minimal for skeleton phase)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Structure
- `CLAUDE.md` -- Full project spec, service structure template, conventions, constraints
- `workspace/01 - Architecture/Overview.md` -- System architecture overview
- `workspace/01 - Architecture/Services.md` -- Service responsibilities and boundaries
- `workspace/01 - Architecture/Data Flow.md` -- Inter-service data flows

### Service Documentation
- `workspace/02 - Services/Auth Service.md` -- Auth service API and responsibilities
- `workspace/02 - Services/TwoFA Service.md` -- TwoFA service API and responsibilities
- `workspace/02 - Services/MPC Node.md` -- MPC node API and responsibilities

### Decisions
- `workspace/04 - Decisions/ADR Log.md` -- All architecture decisions (ADR-001 through ADR-006)

### Requirements
- `.planning/REQUIREMENTS.md` -- Phase 1 requirements: INFRA-01, INFRA-02, INFRA-08, INFRA-09, INFRA-10, INFRA-11

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None -- service directories are empty, everything built from scratch

### Established Patterns
- Clean Architecture pattern fully specified in CLAUDE.md and ADR-005
- Naming conventions, import organization, error handling all documented in .planning/codebase/CONVENTIONS.md

### Integration Points
- Each service is an independent Go module (`github.com/vbncursed/vkr/{auth,twofa,mpc}`)
- Proto definitions in `api/` per service, generated code in `internal/pb/`
- Config loaded from `config.yaml` via `config/config.go`

</code_context>

<specifics>
## Specific Ideas

No specific requirements -- user chose to skip discussion, all decisions at Claude's discretion following CLAUDE.md and ADR specs.

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 01-project-scaffolding*
*Context gathered: 2026-04-11*
