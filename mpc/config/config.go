// Package config provides configuration loading for the MPC service.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
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
	Port int `yaml:"port"`
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

// Load reads and parses the configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
