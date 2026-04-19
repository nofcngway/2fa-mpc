# API Gateway Design Spec

## Overview

REST API Gateway for MPC-2FA system. Single entry point for frontend, translates REST to gRPC using grpc-gateway. Auto-generates OpenAPI specs from proto annotations. ScalarUI for interactive API docs.

## Architecture

```
Frontend (Next.js) → HTTPS → Gateway (REST, :8080)
                                 ├── /api/v1/auth/*   → gRPC → Auth Service (:50051)
                                 ├── /api/v1/2fa/*    → gRPC → TwoFA Service (:50052)
                                 ├── /docs            → ScalarUI (CDN-loaded HTML)
                                 ├── /openapi/*.json  → generated swagger specs
                                 ├── /healthz         → health check
                                 └── /metrics         → Prometheus (separate :9103)
```

## Module

`github.com/vbncursed/vkr/gateway` — отдельный Go-модуль, Go 1.26.2.

## Directory Structure

```
gateway/
├── cmd/app/main.go               # Entry point, graceful shutdown
├── config/
│   └── config.go                  # YAML config + env overrides
├── api/                           # Proto definitions with google.api.http annotations
│   ├── google/api/                # annotations.proto, http.proto
│   ├── models/
│   │   └── models.proto
│   ├── auth_api/
│   │   └── auth_service.proto     # Auth RPCs with HTTP mapping
│   └── twofa_api/
│       └── twofa_service.proto    # TwoFA RPCs with HTTP mapping
├── internal/
│   ├── bootstrap/                 # DI factories
│   │   ├── server.go              # HTTP server + gRPC-Gateway mux setup
│   │   ├── grpc_clients.go        # Auth + TwoFA gRPC connections
│   │   ├── redis.go               # Redis client
│   │   ├── logger.go              # slog JSON logger
│   │   └── swagger.go             # ScalarUI + swagger file serving
│   ├── middleware/
│   │   ├── auth.go                # JWT validation via Auth.ValidateToken
│   │   ├── rate_limit.go          # Redis sliding window rate limiter
│   │   ├── cors.go                # CORS headers
│   │   ├── logging.go             # HTTP request logging (slog)
│   │   ├── metrics.go             # Prometheus HTTP metrics
│   │   └── recovery.go            # Panic recovery
│   └── pb/                        # Generated protobuf + gateway + swagger
│       ├── auth_api/
│       │   └── auth_service.swagger.json
│       ├── twofa_api/
│       │   └── twofa_service.swagger.json
│       └── models/
├── scripts/
│   └── generate.sh                # protoc generation script
├── config.yaml
├── go.mod
└── Makefile
```

## REST API Endpoints

All endpoints under `/api/v1/`.

### Auth Endpoints (public unless noted)

| Method | Path | gRPC RPC | Auth Required |
|--------|------|----------|---------------|
| POST | `/api/v1/auth/register` | Auth.Register | No |
| POST | `/api/v1/auth/login` | Auth.Login | No |
| POST | `/api/v1/auth/refresh` | Auth.RefreshToken | No |
| POST | `/api/v1/auth/logout` | Auth.Logout | Yes |
| POST | `/api/v1/auth/logout-all` | Auth.LogoutAll | Yes |
| POST | `/api/v1/auth/validate` | Auth.ValidateToken | No |

### TwoFA Endpoints (all require auth)

| Method | Path | gRPC RPC | Auth Required |
|--------|------|----------|---------------|
| POST | `/api/v1/2fa/setup` | TwoFA.Setup2FA | Yes |
| POST | `/api/v1/2fa/verify` | TwoFA.Verify2FA | Yes |
| POST | `/api/v1/2fa/disable` | TwoFA.Disable2FA | Yes |
| GET | `/api/v1/2fa/status` | TwoFA.Get2FAStatus | Yes |

### Infrastructure Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/docs` | ScalarUI interactive API docs |
| GET | `/openapi/auth.json` | Auth service OpenAPI spec |
| GET | `/openapi/twofa.json` | TwoFA service OpenAPI spec |
| GET | `/healthz` | Health check (200 OK) |

## Proto Annotations

Auth service proto with HTTP mapping:

```protobuf
import "google/api/annotations.proto";

service AuthService {
    rpc Register(RegisterRequest) returns (RegisterResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/register"
            body: "*"
        };
    }
    rpc Login(LoginRequest) returns (LoginResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/login"
            body: "*"
        };
    }
    // ... etc
}
```

TwoFA service proto with HTTP mapping:

```protobuf
service TwoFAService {
    rpc Setup2FA(Setup2FARequest) returns (Setup2FAResponse) {
        option (google.api.http) = {
            post: "/api/v1/2fa/setup"
            body: "*"
        };
    }
    rpc Get2FAStatus(Get2FAStatusRequest) returns (Get2FAStatusResponse) {
        option (google.api.http) = {
            get: "/api/v1/2fa/status"
        };
    }
    // ... etc
}
```

## Middleware Chain

```
Request → Recovery → Metrics → Logging → CORS → RateLimit → AuthCheck → gRPC-Gateway
```

### AuthCheck Middleware

- Reads `Authorization: Bearer <token>` header
- For protected paths: calls Auth.ValidateToken via gRPC
- On success: injects `user_id` and `email` into request context (gRPC metadata)
- On failure: returns 401 Unauthorized
- Public paths bypass auth check

Protected path prefixes:
- `/api/v1/auth/logout`
- `/api/v1/auth/logout-all`
- `/api/v1/2fa/*`

### Rate Limiting

- Redis sliding window, keyed by client IP
- Default: 60 requests/minute, burst 10
- Returns 429 Too Many Requests on exceed
- Graceful degradation: if Redis unavailable, allow request (log warning)

### CORS

- Configurable `allowed_origins` list
- Methods: GET, POST, OPTIONS
- Headers: Content-Type, Authorization
- Credentials: true

## Configuration

```yaml
server:
  port: 8080
  metrics_port: 9103
  log_level: info
  read_timeout: 10s
  write_timeout: 15s

auth_service:
  addr: "localhost:50051"

twofa_service:
  addr: "localhost:50052"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

rate_limit:
  requests_per_minute: 60
  burst: 10

cors:
  allowed_origins:
    - "http://localhost:3000"
```

Environment variable overrides with `GATEWAY_` prefix:
- `GATEWAY_SERVER_PORT`
- `GATEWAY_AUTH_SERVICE_ADDR`
- `GATEWAY_TWOFA_SERVICE_ADDR`
- `GATEWAY_REDIS_ADDR`

## ScalarUI Integration

Served at `/docs` as inline HTML loading Scalar from CDN:

```html
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
<script>
    Scalar.createApiReference('#app', {
        sources: [
            { title: 'Auth Service', url: '/openapi/auth.json' },
            { title: 'TwoFA Service', url: '/openapi/twofa.json' }
        ]
    })
</script>
```

Swagger JSON files served from `internal/pb/auth_api/` and `internal/pb/twofa_api/` directories (generated by protoc alongside the Go code).

## Dependencies

```
google.golang.org/grpc v1.80.0
google.golang.org/protobuf v1.36.11
github.com/grpc-ecosystem/grpc-gateway/v2
github.com/redis/go-redis/v9 v9.18.0
github.com/prometheus/client_golang v1.23.2
go.yaml.in/yaml/v4 v4.0.0-rc.4
```

Replace directives for sibling proto modules:
```
replace github.com/vbncursed/vkr/auth => ../auth
replace github.com/vbncursed/vkr/twofa => ../twofa
```

## Graceful Shutdown

Order (30s timeout):
1. Stop HTTP server (drain connections)
2. Close Redis
3. Close gRPC connections (Auth, TwoFA)
4. Stop metrics server

## Error Response Format

gRPC-Gateway translates gRPC status codes to HTTP:

| gRPC Code | HTTP Status |
|-----------|-------------|
| InvalidArgument | 400 Bad Request |
| Unauthenticated | 401 Unauthorized |
| PermissionDenied | 403 Forbidden |
| NotFound | 404 Not Found |
| AlreadyExists | 409 Conflict |
| FailedPrecondition | 412 Precondition Failed |
| ResourceExhausted | 429 Too Many Requests |
| Internal | 500 Internal Server Error |
| Unavailable | 503 Service Unavailable |

Response body (JSON):
```json
{
    "code": 3,
    "message": "invalid email format",
    "details": []
}
```

## What Is NOT Included

- OAuth/SSO
- WebSocket support
- Request caching
- Circuit breaker (future)
- TLS termination (handled by reverse proxy/LB in production)
