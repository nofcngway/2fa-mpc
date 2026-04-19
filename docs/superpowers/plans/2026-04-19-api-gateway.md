# API Gateway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a REST API Gateway that translates HTTP to gRPC using grpc-gateway, with Redis rate limiting, JWT auth middleware, Prometheus metrics, and ScalarUI documentation auto-generated from proto annotations.

**Architecture:** gRPC-Gateway v2 generates REST handlers from proto `google.api.http` annotations. Gateway module copies proto files with HTTP annotations, generates Go + gateway + swagger code. Middleware chain wraps the gateway mux: Recovery → Metrics → Logging → CORS → RateLimit → Auth.

**Tech Stack:** Go 1.26.2, grpc-gateway/v2, go-redis/v9, prometheus, slog, ScalarUI (CDN)

---

### Task 1: Initialize Go Module and Config

**Files:**
- Create: `gateway/go.mod`
- Create: `gateway/config/config.go`
- Create: `gateway/config.yaml`

- [ ] **Step 1: Create go.mod**

```bash
cd /Users/vbncursed/programming/2fa/gateway
go mod init github.com/vbncursed/vkr/gateway
```

- [ ] **Step 2: Write config.yaml**

Create `gateway/config.yaml`:

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

swagger:
  auth: "internal/pb/auth_api"
  twofa: "internal/pb/twofa_api"
```

- [ ] **Step 3: Write config/config.go**

Create `gateway/config/config.go`:

```go
// Package config loads gateway configuration from YAML with environment overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	AuthService  ServiceConfig      `yaml:"auth_service"`
	TwoFAService ServiceConfig      `yaml:"twofa_service"`
	Redis        RedisConfig        `yaml:"redis"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	CORS         CORSConfig         `yaml:"cors"`
	Swagger      SwaggerConfig      `yaml:"swagger"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	MetricsPort  int           `yaml:"metrics_port"`
	LogLevel     string        `yaml:"log_level"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type ServiceConfig struct {
	Addr string `yaml:"addr"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	Burst             int `yaml:"burst"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type SwaggerConfig struct {
	Auth  string `yaml:"auth"`
	TwoFA string `yaml:"twofa"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("server.port is required")
	}
	if c.AuthService.Addr == "" {
		return fmt.Errorf("auth_service.addr is required")
	}
	if c.TwoFAService.Addr == "" {
		return fmt.Errorf("twofa_service.addr is required")
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	envInt("GATEWAY_SERVER_PORT", &cfg.Server.Port)
	envInt("GATEWAY_SERVER_METRICS_PORT", &cfg.Server.MetricsPort)
	envString("GATEWAY_SERVER_LOG_LEVEL", &cfg.Server.LogLevel)
	envString("GATEWAY_AUTH_SERVICE_ADDR", &cfg.AuthService.Addr)
	envString("GATEWAY_TWOFA_SERVICE_ADDR", &cfg.TwoFAService.Addr)
	envString("GATEWAY_REDIS_ADDR", &cfg.Redis.Addr)
	envString("GATEWAY_REDIS_PASSWORD", &cfg.Redis.Password)
	envInt("GATEWAY_REDIS_DB", &cfg.Redis.DB)
	envInt("GATEWAY_RATE_LIMIT_RPM", &cfg.RateLimit.RequestsPerMinute)
}

func envString(key string, target *string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

func envInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*target = n
		}
	}
}

func envStringSlice(key string, target *[]string) {
	if v := os.Getenv(key); v != "" {
		*target = strings.Split(v, ",")
	}
}
```

- [ ] **Step 4: Add dependencies**

```bash
cd /Users/vbncursed/programming/2fa/gateway
go get go.yaml.in/yaml/v4@v4.0.0-rc.4
go get google.golang.org/grpc@v1.80.0
go get google.golang.org/protobuf@v1.36.11
go get github.com/grpc-ecosystem/grpc-gateway/v2
go get github.com/redis/go-redis/v9@v9.18.0
go get github.com/prometheus/client_golang@v1.23.2
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/vbncursed/programming/2fa/gateway
go build ./config/...
```

- [ ] **Step 6: Commit**

```bash
git add gateway/
git commit -m "feat(gateway): initialize module with config"
```

---

### Task 2: Proto Files with HTTP Annotations

**Files:**
- Create: `gateway/api/google/api/annotations.proto`
- Create: `gateway/api/google/api/http.proto`
- Create: `gateway/api/models/models.proto`
- Create: `gateway/api/auth_api/auth_service.proto`
- Create: `gateway/api/twofa_api/twofa_service.proto`

- [ ] **Step 1: Create google/api annotation protos**

These are standard Google API proto definitions needed for HTTP annotations.

Create `gateway/api/google/api/annotations.proto`:

```protobuf
syntax = "proto3";

package google.api;

import "google/api/http.proto";
import "google/protobuf/descriptor.proto";

option go_package = "google.golang.org/genproto/googleapis/api/annotations;annotations";

extend google.protobuf.MethodOptions {
    HttpRule http = 72295728;
}
```

Create `gateway/api/google/api/http.proto`:

```protobuf
syntax = "proto3";

package google.api;

option go_package = "google.golang.org/genproto/googleapis/api/annotations;annotations";

message Http {
    repeated HttpRule rules = 1;
    bool fully_decode_reserved_expansion = 2;
}

message HttpRule {
    string selector = 1;
    oneof pattern {
        string get = 2;
        string put = 3;
        string post = 4;
        string delete = 5;
        string patch = 6;
        CustomHttpPattern custom = 8;
    }
    string body = 7;
    string response_body = 12;
    repeated HttpRule additional_bindings = 11;
}

message CustomHttpPattern {
    string kind = 1;
    string path = 2;
}
```

- [ ] **Step 2: Copy models.proto (same as auth)**

Create `gateway/api/models/models.proto`:

```protobuf
syntax = "proto3";
package auth_models;
option go_package = "github.com/vbncursed/vkr/gateway/internal/pb/models";

message User {
    string id = 1;
    string email = 2;
    string password_hash = 3;
    string created_at = 4;
    string updated_at = 5;
}

message TokenPair {
    string access_token = 1;
    string refresh_token = 2;
}
```

- [ ] **Step 3: Create auth_service.proto with HTTP annotations**

Create `gateway/api/auth_api/auth_service.proto`:

```protobuf
syntax = "proto3";
package auth_api;
option go_package = "github.com/vbncursed/vkr/gateway/internal/pb/auth_api";

import "google/api/annotations.proto";
import "models/models.proto";

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
    rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/refresh"
            body: "*"
        };
    }
    rpc Logout(LogoutRequest) returns (LogoutResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/logout"
            body: "*"
        };
    }
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/validate"
            body: "*"
        };
    }
    rpc LogoutAll(LogoutAllRequest) returns (LogoutAllResponse) {
        option (google.api.http) = {
            post: "/api/v1/auth/logout-all"
            body: "*"
        };
    }
}

message RegisterRequest {
    string email = 1;
    string password = 2;
}

message RegisterResponse {
    auth_models.TokenPair tokens = 1;
    auth_models.User user = 2;
}

message LoginRequest {
    string email = 1;
    string password = 2;
}

message LoginResponse {
    auth_models.TokenPair tokens = 1;
    auth_models.User user = 2;
}

message RefreshTokenRequest {
    string refresh_token = 1;
}

message RefreshTokenResponse {
    auth_models.TokenPair tokens = 1;
}

message LogoutRequest {
    string refresh_token = 1;
}

message LogoutResponse {}

message ValidateTokenRequest {
    string access_token = 1;
}

message ValidateTokenResponse {
    string user_id = 1;
    string email = 2;
}

message LogoutAllRequest {
    string user_id = 1;
}

message LogoutAllResponse {}
```

- [ ] **Step 4: Create twofa_service.proto with HTTP annotations**

Create `gateway/api/twofa_api/twofa_service.proto`:

```protobuf
syntax = "proto3";
package twofa_api;
option go_package = "github.com/vbncursed/vkr/gateway/internal/pb/twofa_api";

import "google/api/annotations.proto";

service TwoFAService {
    rpc Setup2FA(Setup2FARequest) returns (Setup2FAResponse) {
        option (google.api.http) = {
            post: "/api/v1/2fa/setup"
            body: "*"
        };
    }
    rpc Verify2FA(Verify2FARequest) returns (Verify2FAResponse) {
        option (google.api.http) = {
            post: "/api/v1/2fa/verify"
            body: "*"
        };
    }
    rpc Disable2FA(Disable2FARequest) returns (Disable2FAResponse) {
        option (google.api.http) = {
            post: "/api/v1/2fa/disable"
            body: "*"
        };
    }
    rpc Get2FAStatus(Get2FAStatusRequest) returns (Get2FAStatusResponse) {
        option (google.api.http) = {
            get: "/api/v1/2fa/status"
        };
    }
}

message Setup2FARequest {
    string user_id = 1;
    string email = 2;
}

message Setup2FAResponse {
    string provisioning_uri = 1;
    repeated string backup_codes = 2;
}

message Verify2FARequest {
    string user_id = 1;
    string otp_code = 2;
}

message Verify2FAResponse {
    bool valid = 1;
    bool is_newly_enabled = 2;
}

message Disable2FARequest {
    string user_id = 1;
    string otp_code = 2;
}

message Disable2FAResponse {}

message Get2FAStatusRequest {
    string user_id = 1;
}

message Get2FAStatusResponse {
    bool is_enabled = 1;
    string created_at = 2;
}
```

- [ ] **Step 5: Commit**

```bash
git add gateway/api/
git commit -m "feat(gateway): add proto files with HTTP annotations"
```

---

### Task 3: Proto Generation Script

**Files:**
- Create: `gateway/scripts/generate.sh`
- Create: `gateway/Makefile`

- [ ] **Step 1: Write generate.sh**

Create `gateway/scripts/generate.sh`:

```bash
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$SERVICE_DIR/api"
OUT_DIR="$SERVICE_DIR/internal/pb"
MODULE="github.com/vbncursed/vkr/gateway"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

PROTOS=$(find "$PROTO_DIR" -name "*.proto" ! -path "*/google/*")

protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out="$OUT_DIR" \
    --grpc-gateway_opt=paths=source_relative \
    --grpc-gateway_opt=generate_unbound_methods=false \
    --openapiv2_out="$OUT_DIR" \
    --openapiv2_opt=logtostderr=true \
    $PROTOS

echo "Proto generation complete for gateway"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x /Users/vbncursed/programming/2fa/gateway/scripts/generate.sh
```

- [ ] **Step 3: Write Makefile**

Create `gateway/Makefile`:

```makefile
.PHONY: generate build run

generate:
	./scripts/generate.sh

build:
	go build -o bin/gateway ./cmd/app

run: build
	./bin/gateway
```

- [ ] **Step 4: Install protoc plugins (if not installed)**

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

- [ ] **Step 5: Run generation**

```bash
cd /Users/vbncursed/programming/2fa/gateway
make generate
```

Expected: `internal/pb/` populated with `*.pb.go`, `*.pb.gw.go`, `*.swagger.json`

- [ ] **Step 6: Run go mod tidy and verify build**

```bash
cd /Users/vbncursed/programming/2fa/gateway
go mod tidy
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add gateway/scripts/ gateway/Makefile gateway/internal/pb/ gateway/go.mod gateway/go.sum
git commit -m "feat(gateway): add proto generation with grpc-gateway and openapiv2"
```

---

### Task 4: Middleware — Recovery, Logging, Metrics

**Files:**
- Create: `gateway/internal/middleware/recovery.go`
- Create: `gateway/internal/middleware/logging.go`
- Create: `gateway/internal/middleware/metrics.go`

- [ ] **Step 1: Write recovery.go**

Create `gateway/internal/middleware/recovery.go`:

```go
// Package middleware provides HTTP middleware for the API gateway.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery catches panics and returns 500 Internal Server Error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				http.Error(w, `{"code":13,"message":"internal error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Write logging.go**

Create `gateway/internal/middleware/logging.go`:

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Logging logs HTTP requests with method, path, status, and duration.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
```

- [ ] **Step 3: Write metrics.go**

Create `gateway/internal/middleware/metrics.go`:

```go
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gateway_http_request_duration_seconds",
		Help: "HTTP request duration in seconds.",
	}, []string{"method", "path"})
)

// Metrics records Prometheus HTTP request metrics.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rec.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	})
}
```

- [ ] **Step 4: Verify build**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go build ./internal/middleware/...
```

- [ ] **Step 5: Commit**

```bash
git add gateway/internal/middleware/
git commit -m "feat(gateway): add recovery, logging, and metrics middleware"
```

---

### Task 5: Middleware — CORS

**Files:**
- Create: `gateway/internal/middleware/cors.go`

- [ ] **Step 1: Write cors.go**

Create `gateway/internal/middleware/cors.go`:

```go
package middleware

import (
	"net/http"
	"slices"
)

// CORS adds Cross-Origin Resource Sharing headers.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add gateway/internal/middleware/cors.go
git commit -m "feat(gateway): add CORS middleware"
```

---

### Task 6: Middleware — Redis Rate Limiting

**Files:**
- Create: `gateway/internal/middleware/rate_limit.go`

- [ ] **Step 1: Write rate_limit.go**

Create `gateway/internal/middleware/rate_limit.go`:

```go
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit enforces per-IP rate limiting using Redis sliding window.
func RateLimit(rdb *redis.Client, requestsPerMinute, burst int) func(http.Handler) http.Handler {
	window := time.Minute
	limit := int64(requestsPerMinute + burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			key := fmt.Sprintf("gateway:rate_limit:%s", ip)

			count, err := rdb.Incr(r.Context(), key).Result()
			if err != nil {
				slog.Warn("rate limit check failed, allowing request", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(context.Background(), key, window)
			}

			if count > limit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"code":8,"message":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add gateway/internal/middleware/rate_limit.go
git commit -m "feat(gateway): add Redis rate limiting middleware"
```

---

### Task 7: Middleware — Auth (JWT Validation via gRPC)

**Files:**
- Create: `gateway/internal/middleware/auth.go`

- [ ] **Step 1: Write auth.go**

Create `gateway/internal/middleware/auth.go`:

```go
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"

	pb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// EmailKey is the context key for the authenticated user's email.
	EmailKey contextKey = "email"
)

// publicPaths lists paths that do not require authentication.
var publicPaths = map[string]bool{
	"/api/v1/auth/register": true,
	"/api/v1/auth/login":    true,
	"/api/v1/auth/refresh":  true,
	"/api/v1/auth/validate": true,
	"/docs":                 true,
	"/healthz":              true,
}

// Auth validates Bearer tokens on protected paths by calling Auth.ValidateToken.
func Auth(authClient pb.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip public paths and infrastructure endpoints
			if publicPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, "/openapi/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			token, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found || token == "" {
				http.Error(w, `{"code":16,"message":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			resp, err := authClient.ValidateToken(r.Context(), &pb.ValidateTokenRequest{
				AccessToken: token,
			})
			if err != nil {
				slog.Debug("token validation failed", "error", err)
				http.Error(w, `{"code":16,"message":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Inject user_id and email into gRPC metadata for downstream services
			md := metadata.Pairs("x-user-id", resp.UserId, "x-user-email", resp.Email)
			ctx := metadata.NewOutgoingContext(r.Context(), md)
			ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
			ctx = context.WithValue(ctx, EmailKey, resp.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add gateway/internal/middleware/auth.go
git commit -m "feat(gateway): add auth middleware with gRPC token validation"
```

---

### Task 8: Bootstrap — Logger, Redis, gRPC Clients

**Files:**
- Create: `gateway/internal/bootstrap/logger.go`
- Create: `gateway/internal/bootstrap/redis.go`
- Create: `gateway/internal/bootstrap/grpc_clients.go`

- [ ] **Step 1: Write logger.go**

Create `gateway/internal/bootstrap/logger.go`:

```go
// Package bootstrap provides dependency injection factories for the API gateway.
package bootstrap

import (
	"log/slog"
	"os"

	"github.com/vbncursed/vkr/gateway/config"
)

// NewLogger creates a structured JSON logger with configurable level.
func NewLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Server.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
```

- [ ] **Step 2: Write redis.go**

Create `gateway/internal/bootstrap/redis.go`:

```go
package bootstrap

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/vbncursed/vkr/gateway/config"
)

// NewRedisClient creates a Redis client and verifies connectivity.
func NewRedisClient(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return rdb, nil
}
```

- [ ] **Step 3: Write grpc_clients.go**

Create `gateway/internal/bootstrap/grpc_clients.go`:

```go
package bootstrap

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vbncursed/vkr/gateway/config"
	authpb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
	twofapb "github.com/vbncursed/vkr/gateway/internal/pb/twofa_api"
)

// GRPCClients holds gRPC client connections to backend services.
type GRPCClients struct {
	AuthConn  *grpc.ClientConn
	TwoFAConn *grpc.ClientConn
	Auth      authpb.AuthServiceClient
	TwoFA     twofapb.TwoFAServiceClient
}

// NewGRPCClients creates gRPC connections to Auth and TwoFA services.
func NewGRPCClients(cfg *config.Config) (*GRPCClients, error) {
	// TODO: configure TLS/mTLS for production deployment
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())

	authConn, err := grpc.NewClient(cfg.AuthService.Addr, opts)
	if err != nil {
		return nil, fmt.Errorf("connect to auth service at %s: %w", cfg.AuthService.Addr, err)
	}

	twofaConn, err := grpc.NewClient(cfg.TwoFAService.Addr, opts)
	if err != nil {
		authConn.Close()
		return nil, fmt.Errorf("connect to twofa service at %s: %w", cfg.TwoFAService.Addr, err)
	}

	return &GRPCClients{
		AuthConn:  authConn,
		TwoFAConn: twofaConn,
		Auth:      authpb.NewAuthServiceClient(authConn),
		TwoFA:     twofapb.NewTwoFAServiceClient(twofaConn),
	}, nil
}

// Close closes all gRPC connections.
func (c *GRPCClients) Close() {
	if c.AuthConn != nil {
		c.AuthConn.Close()
	}
	if c.TwoFAConn != nil {
		c.TwoFAConn.Close()
	}
}
```

- [ ] **Step 4: Verify build**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go build ./internal/bootstrap/...
```

- [ ] **Step 5: Commit**

```bash
git add gateway/internal/bootstrap/
git commit -m "feat(gateway): add bootstrap for logger, redis, and gRPC clients"
```

---

### Task 9: Bootstrap — Server with gRPC-Gateway Mux and ScalarUI

**Files:**
- Create: `gateway/internal/bootstrap/server.go`
- Create: `gateway/internal/bootstrap/swagger.go`

- [ ] **Step 1: Write swagger.go**

Create `gateway/internal/bootstrap/swagger.go`:

```go
package bootstrap

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const scalarHTML = `<!DOCTYPE html>
<html>
<head>
    <title>MPC-2FA API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
        Scalar.createApiReference('#app', {
            sources: [
                { title: 'Auth Service', url: '/openapi/auth.json' },
                { title: 'TwoFA Service', url: '/openapi/twofa.json' }
            ]
        })
    </script>
</body>
</html>`

// DocsHandler serves the ScalarUI HTML page.
func DocsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(scalarHTML))
	}
}

// SwaggerFileHandler serves a swagger JSON file found in the given directory.
func SwaggerFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := findSwaggerFile(dir)
		if path == "" {
			http.Error(w, "swagger spec not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, path)
	}
}

func findSwaggerFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if result := findSwaggerFile(full); result != "" {
				return result
			}
		} else if strings.HasSuffix(e.Name(), ".swagger.json") {
			return full
		}
	}
	return ""
}
```

- [ ] **Step 2: Write server.go**

Create `gateway/internal/bootstrap/server.go`:

```go
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vbncursed/vkr/gateway/config"
	"github.com/vbncursed/vkr/gateway/internal/middleware"
	authpb "github.com/vbncursed/vkr/gateway/internal/pb/auth_api"
	twofapb "github.com/vbncursed/vkr/gateway/internal/pb/twofa_api"
)

// NewHTTPServer creates the main HTTP server with gRPC-Gateway mux and middleware.
func NewHTTPServer(ctx context.Context, cfg *config.Config, clients *GRPCClients, rdb *redis.Client) (*http.Server, error) {
	// gRPC-Gateway mux
	gwMux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, gwMux, cfg.AuthService.Addr, opts); err != nil {
		return nil, fmt.Errorf("register auth gateway: %w", err)
	}
	if err := twofapb.RegisterTwoFAServiceHandlerFromEndpoint(ctx, gwMux, cfg.TwoFAService.Addr, opts); err != nil {
		return nil, fmt.Errorf("register twofa gateway: %w", err)
	}

	// HTTP router for static routes + gateway mux
	router := http.NewServeMux()
	router.HandleFunc("GET /docs", DocsHandler())
	router.HandleFunc("GET /openapi/auth.json", SwaggerFileHandler(cfg.Swagger.Auth))
	router.HandleFunc("GET /openapi/twofa.json", SwaggerFileHandler(cfg.Swagger.TwoFA))
	router.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	router.Handle("/", gwMux)

	// Middleware chain: Recovery → Metrics → Logging → CORS → RateLimit → Auth → Router
	var handler http.Handler = router
	handler = middleware.Auth(clients.Auth)(handler)
	handler = middleware.RateLimit(rdb, cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)(handler)
	handler = middleware.CORS(cfg.CORS.AllowedOrigins)(handler)
	handler = middleware.Logging(handler)
	handler = middleware.Metrics(handler)
	handler = middleware.Recovery(handler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	slog.Info("HTTP server configured",
		"port", cfg.Server.Port,
		"auth_service", cfg.AuthService.Addr,
		"twofa_service", cfg.TwoFAService.Addr,
	)

	return srv, nil
}
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go build ./internal/bootstrap/...
```

- [ ] **Step 4: Commit**

```bash
git add gateway/internal/bootstrap/server.go gateway/internal/bootstrap/swagger.go
git commit -m "feat(gateway): add HTTP server with gRPC-Gateway mux and ScalarUI"
```

---

### Task 10: Entry Point — main.go with Graceful Shutdown

**Files:**
- Create: `gateway/cmd/app/main.go`

- [ ] **Step 1: Write main.go**

Create `gateway/cmd/app/main.go`:

```go
// Package main is the entry point for the API Gateway service.
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vbncursed/vkr/gateway/config"
	"github.com/vbncursed/vkr/gateway/internal/bootstrap"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(bootstrap.NewLogger(cfg))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// gRPC clients to backend services
	clients, err := bootstrap.NewGRPCClients(cfg)
	if err != nil {
		slog.Error("failed to create gRPC clients", "error", err)
		os.Exit(1)
	}

	// Redis for rate limiting
	rdb, err := bootstrap.NewRedisClient(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	// HTTP server with gRPC-Gateway
	httpServer, err := bootstrap.NewHTTPServer(ctx, cfg, clients, rdb)
	if err != nil {
		slog.Error("failed to create HTTP server", "error", err)
		os.Exit(1)
	}

	// Metrics HTTP server on separate port
	metricsPort := cmp.Or(cfg.Server.MetricsPort, 9103)
	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", metricsPort),
		Handler:      promhttp.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("metrics server started", "port", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server error", "error", err)
		}
	}()

	// Start HTTP server
	go func() {
		slog.Info("gateway started", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down gateway")

	// Ordered shutdown with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. Stop HTTP server (drain connections)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown HTTP server", "error", err)
	}
	slog.Info("HTTP server stopped")

	// 2. Close Redis
	if err := rdb.Close(); err != nil {
		slog.Error("failed to close Redis", "error", err)
	}
	slog.Info("Redis connection closed")

	// 3. Close gRPC connections
	clients.Close()
	slog.Info("gRPC connections closed")

	// 4. Stop metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}
	slog.Info("metrics server stopped")

	slog.Info("gateway shutdown complete")
}
```

- [ ] **Step 2: Verify full build**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add gateway/cmd/
git commit -m "feat(gateway): add entry point with graceful shutdown"
```

---

### Task 11: Final Verification

- [ ] **Step 1: Full build**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go build ./...
```

- [ ] **Step 2: go vet**

```bash
cd /Users/vbncursed/programming/2fa/gateway && go vet ./...
```

- [ ] **Step 3: Verify swagger files exist**

```bash
ls gateway/internal/pb/auth_api/*.swagger.json
ls gateway/internal/pb/twofa_api/*.swagger.json
```

- [ ] **Step 4: Verify all other services still build**

```bash
cd /Users/vbncursed/programming/2fa/auth && go build ./...
cd /Users/vbncursed/programming/2fa/twofa && go build ./...
cd /Users/vbncursed/programming/2fa/mpc && go build ./...
```

- [ ] **Step 5: Final commit if any outstanding changes**

```bash
git add gateway/
git commit -m "feat(gateway): complete API gateway implementation"
```
