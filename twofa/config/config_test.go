package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load("../config.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 9091 {
		t.Errorf("expected server port 9091, got %d", cfg.Server.Port)
	}

	if cfg.Database.DSN == "" {
		t.Error("expected non-empty database DSN")
	}

	if cfg.Redis.Addr == "" {
		t.Error("expected non-empty redis addr")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		t.Error("expected at least one kafka broker")
	}

	if cfg.Kafka.Topic == "" {
		t.Error("expected non-empty kafka topic")
	}

	if len(cfg.MPCNodes) == 0 {
		t.Error("expected at least one MPC node")
	}

	if cfg.SharedSecret == "" {
		t.Error("expected non-empty shared secret")
	}
}
