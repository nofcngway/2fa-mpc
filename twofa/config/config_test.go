package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

const testYAML = `server:
  port: 9091
  metrics_port: 9101
  log_level: "info"
database:
  dsn: "postgres://twofa:pass@localhost:5434/twofa_db?sslmode=disable"
redis:
  addr: "localhost:6381"
  password: ""
  db: 0
kafka:
  brokers:
    - "localhost:9093"
  topic: "twofa-events"
mpc_nodes:
  - addr: "localhost:9200"
  - addr: "localhost:9201"
  - addr: "localhost:9202"
shared_secret: "dev-secret"
mpc_timeout: 5s
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

	assert.Equal(t, cfg.Server.Port, 9091)
	assert.Equal(t, cfg.Server.MetricsPort, 9101)
	assert.Equal(t, cfg.Database.DSN, "postgres://twofa:pass@localhost:5434/twofa_db?sslmode=disable")
	assert.Equal(t, cfg.SharedSecret, "dev-secret")
	assert.Equal(t, len(cfg.MPCNodes), 3)
	assert.Equal(t, cfg.MPCTimeout, 5*time.Second)
}

func TestLoad_EnvVarsOnly(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("TWOFA_SERVER_PORT", "8091")
	t.Setenv("TWOFA_SERVER_METRICS_PORT", "8101")
	t.Setenv("TWOFA_SERVER_LOG_LEVEL", "debug")
	t.Setenv("TWOFA_DATABASE_DSN", "postgres://env:env@db:5434/envdb")
	t.Setenv("TWOFA_REDIS_ADDR", "redis:6381")
	t.Setenv("TWOFA_REDIS_PASSWORD", "secret")
	t.Setenv("TWOFA_REDIS_DB", "1")
	t.Setenv("TWOFA_KAFKA_BROKERS", "broker1:9093")
	t.Setenv("TWOFA_KAFKA_TOPIC", "env-twofa-events")
	t.Setenv("TWOFA_MPC_NODES", "mpc-node-1:9200,mpc-node-2:9201,mpc-node-3:9202")
	t.Setenv("TWOFA_SHARED_SECRET", "env-secret")
	t.Setenv("TWOFA_MPC_TIMEOUT", "10s")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Server.Port, 8091)
	assert.Equal(t, cfg.Server.MetricsPort, 8101)
	assert.Equal(t, cfg.Server.LogLevel, "debug")
	assert.Equal(t, cfg.Database.DSN, "postgres://env:env@db:5434/envdb")
	assert.Equal(t, cfg.Redis.Addr, "redis:6381")
	assert.Equal(t, cfg.Redis.Password, "secret")
	assert.Equal(t, cfg.Redis.DB, 1)
	assert.DeepEqual(t, cfg.Kafka.Brokers, []string{"broker1:9093"})
	assert.Equal(t, cfg.Kafka.Topic, "env-twofa-events")
	assert.Equal(t, len(cfg.MPCNodes), 3)
	assert.Equal(t, cfg.MPCNodes[0].Addr, "mpc-node-1:9200")
	assert.Equal(t, cfg.MPCNodes[1].Addr, "mpc-node-2:9201")
	assert.Equal(t, cfg.MPCNodes[2].Addr, "mpc-node-3:9202")
	assert.Equal(t, cfg.SharedSecret, "env-secret")
	assert.Equal(t, cfg.MPCTimeout, 10*time.Second)
}

func TestLoad_MPCNodesCommaSeparated(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("TWOFA_SERVER_PORT", "9091")
	t.Setenv("TWOFA_DATABASE_DSN", "postgres://u:p@h:5434/d")
	t.Setenv("TWOFA_SHARED_SECRET", "s")
	t.Setenv("TWOFA_MPC_NODES", "mpc-node-1:9200, mpc-node-2:9201, mpc-node-3:9202")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)
	assert.Equal(t, len(cfg.MPCNodes), 3)
	assert.Equal(t, cfg.MPCNodes[0].Addr, "mpc-node-1:9200")
	assert.Equal(t, cfg.MPCNodes[1].Addr, "mpc-node-2:9201")
	assert.Equal(t, cfg.MPCNodes[2].Addr, "mpc-node-3:9202")
}

func TestLoad_MPCTimeoutDurationParsing(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("TWOFA_MPC_TIMEOUT", "10s")

	cfg, err := Load(path)
	assert.NilError(t, err)
	assert.Equal(t, cfg.MPCTimeout, 10*time.Second)
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("TWOFA_DATABASE_DSN", "postgres://override:override@db:5434/overridedb")

	cfg, err := Load(path)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Database.DSN, "postgres://override:override@db:5434/overridedb")
	// Non-overridden stays from yaml
	assert.Equal(t, cfg.Server.Port, 9091)
	assert.Equal(t, cfg.SharedSecret, "dev-secret")
}

func TestLoad_ValidationRejectsNon3MPCNodes(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("TWOFA_SERVER_PORT", "9091")
	t.Setenv("TWOFA_DATABASE_DSN", "postgres://u:p@h:5434/d")
	t.Setenv("TWOFA_SHARED_SECRET", "s")
	t.Setenv("TWOFA_MPC_NODES", "node1:9200,node2:9201")

	_, err := Load(nonexistent)
	assert.ErrorContains(t, err, "exactly 3 mpc_nodes required")
}
