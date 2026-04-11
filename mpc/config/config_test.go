package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	content := []byte(`
server:
  port: 9100

database:
  dsn: "postgres://mpc_user:mpc_pass@localhost:5435/mpc_db?sslmode=disable"

kafka:
  brokers:
    - "localhost:9094"
  topic: "mpc-events"

node:
  id: 1
  encryption_key: "0123456789abcdef0123456789abcdef"

shared_secret: "dev-shared-secret-change-in-production"
`)

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9100 {
		t.Errorf("Server.Port = %d, want 9100", cfg.Server.Port)
	}

	if cfg.Database.DSN == "" {
		t.Error("Database.DSN is empty")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		t.Error("Kafka.Brokers is empty")
	}

	if cfg.Kafka.Topic != "mpc-events" {
		t.Errorf("Kafka.Topic = %s, want mpc-events", cfg.Kafka.Topic)
	}

	if cfg.Node.ID == 0 {
		t.Error("Node.ID is 0")
	}

	if cfg.Node.EncryptionKey == "" {
		t.Error("Node.EncryptionKey is empty")
	}

	if cfg.SharedSecret == "" {
		t.Error("SharedSecret is empty")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("{{invalid yaml")); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}
