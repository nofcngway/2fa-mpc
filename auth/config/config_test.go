package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	configPath := filepath.Join("..", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("config.yaml not found, skipping")
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port == 0 {
		t.Error("server.port should not be 0")
	}

	if cfg.Database.DSN == "" {
		t.Error("database.dsn should not be empty")
	}

	if cfg.JWT.PrivateKeyPath == "" {
		t.Error("jwt.private_key_path should not be empty")
	}

	if cfg.JWT.AccessTokenTTL == 0 {
		t.Error("jwt.access_token_ttl should not be 0")
	}

	if cfg.JWT.RefreshTokenTTL == 0 {
		t.Error("jwt.refresh_token_ttl should not be 0")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		t.Error("kafka.brokers should not be empty")
	}

	if cfg.Redis.Addr == "" {
		t.Error("redis.addr should not be empty")
	}
}
