package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

const testYAML = `server:
  port: 9090
  metrics_port: 9100
  log_level: "info"
database:
  dsn: "postgres://user:pass@localhost:5432/db?sslmode=disable"
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
kafka:
  brokers:
    - "localhost:9092"
  topic: "auth-events"
jwt:
  private_key_path: "keys/private.pem"
  public_key_path: "keys/public.pem"
  access_token_ttl: 15m
  refresh_token_ttl: 168h
`

func writeTestYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(testYAML), 0o644)
	assert.NilError(t, err)
	return path
}

func TestLoad_YAMLOnly(t *testing.T) {
	path := writeTestYAML(t)

	cfg, err := Load(path)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Server.Port, 9090)
	assert.Equal(t, cfg.Server.MetricsPort, 9100)
	assert.Equal(t, cfg.Server.LogLevel, "info")
	assert.Equal(t, cfg.Database.DSN, "postgres://user:pass@localhost:5432/db?sslmode=disable")
	assert.Equal(t, cfg.Redis.Addr, "localhost:6379")
	assert.Equal(t, cfg.Redis.DB, 0)
	assert.Equal(t, cfg.JWT.PrivateKeyPath, "keys/private.pem")
	assert.Equal(t, cfg.JWT.PublicKeyPath, "keys/public.pem")
	assert.Equal(t, cfg.JWT.AccessTokenTTL, 15*time.Minute)
	assert.Equal(t, cfg.JWT.RefreshTokenTTL, 168*time.Hour)
}

func TestLoad_EnvVarsOnly(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("AUTH_SERVER_PORT", "8080")
	t.Setenv("AUTH_SERVER_METRICS_PORT", "8081")
	t.Setenv("AUTH_SERVER_LOG_LEVEL", "debug")
	t.Setenv("AUTH_DATABASE_DSN", "postgres://env:env@db:5432/envdb")
	t.Setenv("AUTH_REDIS_ADDR", "redis:6379")
	t.Setenv("AUTH_REDIS_PASSWORD", "secret")
	t.Setenv("AUTH_REDIS_DB", "2")
	t.Setenv("AUTH_KAFKA_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("AUTH_KAFKA_TOPIC", "env-events")
	t.Setenv("AUTH_JWT_PRIVATE_KEY_PATH", "/keys/priv.pem")
	t.Setenv("AUTH_JWT_PUBLIC_KEY_PATH", "/keys/pub.pem")
	t.Setenv("AUTH_JWT_ACCESS_TOKEN_TTL", "30m")
	t.Setenv("AUTH_JWT_REFRESH_TOKEN_TTL", "24h")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Server.Port, 8080)
	assert.Equal(t, cfg.Server.MetricsPort, 8081)
	assert.Equal(t, cfg.Server.LogLevel, "debug")
	assert.Equal(t, cfg.Database.DSN, "postgres://env:env@db:5432/envdb")
	assert.Equal(t, cfg.Redis.Addr, "redis:6379")
	assert.Equal(t, cfg.Redis.Password, "secret")
	assert.Equal(t, cfg.Redis.DB, 2)
	assert.DeepEqual(t, cfg.Kafka.Brokers, []string{"broker1:9092", "broker2:9092"})
	assert.Equal(t, cfg.Kafka.Topic, "env-events")
	assert.Equal(t, cfg.JWT.PrivateKeyPath, "/keys/priv.pem")
	assert.Equal(t, cfg.JWT.PublicKeyPath, "/keys/pub.pem")
	assert.Equal(t, cfg.JWT.AccessTokenTTL, 30*time.Minute)
	assert.Equal(t, cfg.JWT.RefreshTokenTTL, 24*time.Hour)
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("AUTH_DATABASE_DSN", "postgres://override:override@db:5432/overridedb")
	t.Setenv("AUTH_SERVER_PORT", "7070")

	cfg, err := Load(path)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Database.DSN, "postgres://override:override@db:5432/overridedb")
	assert.Equal(t, cfg.Server.Port, 7070)
	assert.Equal(t, cfg.Redis.Addr, "localhost:6379")
	assert.Equal(t, cfg.JWT.PrivateKeyPath, "keys/private.pem")
}

func TestLoad_ValidationFailsMissingRequired(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	_, err := Load(nonexistent)
	assert.ErrorContains(t, err, "config validation")
	assert.ErrorContains(t, err, "server.port is required")
	assert.ErrorContains(t, err, "database.dsn is required")
}

func TestLoad_KafkaBrokersCommaSeparated(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("AUTH_SERVER_PORT", "9090")
	t.Setenv("AUTH_DATABASE_DSN", "postgres://u:p@h:5432/d")
	t.Setenv("AUTH_REDIS_ADDR", "redis:6379")
	t.Setenv("AUTH_JWT_PRIVATE_KEY_PATH", "/k/priv.pem")
	t.Setenv("AUTH_JWT_PUBLIC_KEY_PATH", "/k/pub.pem")
	t.Setenv("AUTH_JWT_ACCESS_TOKEN_TTL", "15m")
	t.Setenv("AUTH_JWT_REFRESH_TOKEN_TTL", "168h")
	t.Setenv("AUTH_KAFKA_BROKERS", "broker1:9092,broker2:9092")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)
	assert.Equal(t, len(cfg.Kafka.Brokers), 2)
	assert.Equal(t, cfg.Kafka.Brokers[0], "broker1:9092")
	assert.Equal(t, cfg.Kafka.Brokers[1], "broker2:9092")
}

func TestLoad_DurationParsing(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("AUTH_SERVER_PORT", "9090")
	t.Setenv("AUTH_DATABASE_DSN", "postgres://u:p@h:5432/d")
	t.Setenv("AUTH_REDIS_ADDR", "redis:6379")
	t.Setenv("AUTH_JWT_PRIVATE_KEY_PATH", "/k/priv.pem")
	t.Setenv("AUTH_JWT_PUBLIC_KEY_PATH", "/k/pub.pem")
	t.Setenv("AUTH_JWT_ACCESS_TOKEN_TTL", "15m")
	t.Setenv("AUTH_JWT_REFRESH_TOKEN_TTL", "168h")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)
	assert.Equal(t, cfg.JWT.AccessTokenTTL, 15*time.Minute)
	assert.Equal(t, cfg.JWT.RefreshTokenTTL, 168*time.Hour)
}

func TestLoad_RedisDBIntParsing(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("AUTH_REDIS_DB", "2")

	cfg, err := Load(path)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Redis.DB, 2)
}
