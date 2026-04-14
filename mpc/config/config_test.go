package config

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

const testYAML = `server:
  port: 9100
  metrics_port: 9102
  log_level: "info"
database:
  dsn: "postgres://mpc:pass@localhost:5435/mpc_db?sslmode=disable"
kafka:
  brokers:
    - "localhost:9094"
  topic: "mpc-events"
node:
  id: 1
  encryption_key: "0123456789abcdef0123456789abcdef"
shared_secret: "dev-secret"
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

	assert.Equal(t, cfg.Server.Port, 9100)
	assert.Equal(t, cfg.Server.MetricsPort, 9102)
	assert.Equal(t, cfg.Database.DSN, "postgres://mpc:pass@localhost:5435/mpc_db?sslmode=disable")
	assert.Equal(t, cfg.Node.ID, 1)
	assert.Equal(t, cfg.Node.EncryptionKey, "0123456789abcdef0123456789abcdef")
	assert.Equal(t, cfg.SharedSecret, "dev-secret")
}

func TestLoad_EnvVarsOnly(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("MPC_SERVER_PORT", "9200")
	t.Setenv("MPC_SERVER_METRICS_PORT", "9202")
	t.Setenv("MPC_SERVER_LOG_LEVEL", "debug")
	t.Setenv("MPC_DATABASE_DSN", "postgres://env:env@db:5435/envdb")
	t.Setenv("MPC_KAFKA_BROKERS", "broker1:9094,broker2:9094")
	t.Setenv("MPC_KAFKA_TOPIC", "env-mpc-events")
	t.Setenv("MPC_NODE_ID", "2")
	t.Setenv("MPC_NODE_ENCRYPTION_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("MPC_SHARED_SECRET", "env-secret")

	cfg, err := Load(nonexistent)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Server.Port, 9200)
	assert.Equal(t, cfg.Server.MetricsPort, 9202)
	assert.Equal(t, cfg.Server.LogLevel, "debug")
	assert.Equal(t, cfg.Database.DSN, "postgres://env:env@db:5435/envdb")
	assert.DeepEqual(t, cfg.Kafka.Brokers, []string{"broker1:9094", "broker2:9094"})
	assert.Equal(t, cfg.Kafka.Topic, "env-mpc-events")
	assert.Equal(t, cfg.Node.ID, 2)
	assert.Equal(t, cfg.Node.EncryptionKey, "abcdef0123456789abcdef0123456789")
	assert.Equal(t, cfg.SharedSecret, "env-secret")
}

func TestLoad_NodeIDIntParsing(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("MPC_NODE_ID", "3")

	cfg, err := Load(path)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Node.ID, 3)
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	path := writeTestYAML(t)

	t.Setenv("MPC_DATABASE_DSN", "postgres://override:override@db:5435/overridedb")

	cfg, err := Load(path)
	assert.NilError(t, err)

	assert.Equal(t, cfg.Database.DSN, "postgres://override:override@db:5435/overridedb")
	// Non-overridden stays from yaml
	assert.Equal(t, cfg.Server.Port, 9100)
	assert.Equal(t, cfg.Node.EncryptionKey, "0123456789abcdef0123456789abcdef")
}

func TestLoad_ValidationRejectsEmptyDSN(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("MPC_SERVER_PORT", "9200")
	t.Setenv("MPC_NODE_ENCRYPTION_KEY", "key")
	t.Setenv("MPC_SHARED_SECRET", "secret")
	// No DSN set

	_, err := Load(nonexistent)
	assert.ErrorContains(t, err, "database.dsn is required")
}

func TestLoad_ValidationRejectsEmptyEncryptionKey(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "nonexistent.yaml")

	t.Setenv("MPC_SERVER_PORT", "9200")
	t.Setenv("MPC_DATABASE_DSN", "postgres://u:p@h:5435/d")
	t.Setenv("MPC_SHARED_SECRET", "secret")
	// No encryption key set

	_, err := Load(nonexistent)
	assert.ErrorContains(t, err, "node.encryption_key is required")
}
