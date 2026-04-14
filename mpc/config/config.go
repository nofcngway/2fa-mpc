// Package config provides configuration loading for the MPC service.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Config represents the full MPC service configuration.
type Config struct {
	Server       ServerConfig   `yaml:"server"`
	Database     DatabaseConfig `yaml:"database"`
	Kafka        KafkaConfig    `yaml:"kafka"`
	Node         NodeConfig     `yaml:"node"`
	SharedSecret string         `yaml:"shared_secret"`
}

// ServerConfig holds gRPC server settings.
type ServerConfig struct {
	Port        int    `yaml:"port"`
	MetricsPort int    `yaml:"metrics_port"`
	LogLevel    string `yaml:"log_level"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// KafkaConfig holds Kafka broker settings for audit events.
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

// NodeConfig holds MPC node-specific settings.
type NodeConfig struct {
	ID            int    `yaml:"id"`
	EncryptionKey string `yaml:"encryption_key"`
}

// Validate checks that all required configuration fields are populated.
func (c *Config) Validate() error {
	var errs []error
	if c.Server.Port == 0 {
		errs = append(errs, fmt.Errorf("server.port is required"))
	}
	if c.Database.DSN == "" {
		errs = append(errs, fmt.Errorf("database.dsn is required"))
	}
	if c.Node.EncryptionKey == "" {
		errs = append(errs, fmt.Errorf("node.encryption_key is required"))
	}
	if c.SharedSecret == "" {
		errs = append(errs, fmt.Errorf("shared_secret is required"))
	}
	return errors.Join(errs...)
}

func envString(key string, target *string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

func envInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			*target = i
		}
	}
}

func envStringSlice(key string, target *[]string) {
	if v := os.Getenv(key); v != "" {
		*target = strings.Split(v, ",")
	}
}

func applyEnvOverrides(cfg *Config) {
	envInt("MPC_SERVER_PORT", &cfg.Server.Port)
	envInt("MPC_SERVER_METRICS_PORT", &cfg.Server.MetricsPort)
	envString("MPC_SERVER_LOG_LEVEL", &cfg.Server.LogLevel)
	envString("MPC_DATABASE_DSN", &cfg.Database.DSN)
	envStringSlice("MPC_KAFKA_BROKERS", &cfg.Kafka.Brokers)
	envString("MPC_KAFKA_TOPIC", &cfg.Kafka.Topic)
	envInt("MPC_NODE_ID", &cfg.Node.ID)
	envString("MPC_NODE_ENCRYPTION_KEY", &cfg.Node.EncryptionKey)
	envString("MPC_SHARED_SECRET", &cfg.SharedSecret)
}

// Load reads and parses the configuration file at the given path.
// If the file does not exist, configuration is loaded entirely from environment variables.
// Environment variables always override yaml values (MPC_* prefix).
func Load(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Load(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}
