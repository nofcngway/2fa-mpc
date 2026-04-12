package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultMPCTimeout is the default timeout for MPC node gRPC calls.
const DefaultMPCTimeout = 5 * time.Second

// Config represents the TwoFA service configuration.
type Config struct {
	Server       ServerConfig    `yaml:"server"`
	Database     DatabaseConfig  `yaml:"database"`
	Redis        RedisConfig     `yaml:"redis"`
	Kafka        KafkaConfig     `yaml:"kafka"`
	MPCNodes     []MPCNodeConfig `yaml:"mpc_nodes"`
	SharedSecret string          `yaml:"shared_secret"`
	MPCTimeout   time.Duration   `yaml:"mpc_timeout"`
}

// GetMPCTimeout returns the configured MPC timeout or the default (5s).
func (c *Config) GetMPCTimeout() time.Duration {
	if c.MPCTimeout == 0 {
		return DefaultMPCTimeout
	}
	return c.MPCTimeout
}

// ServerConfig holds gRPC server settings.
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

// MPCNodeConfig holds connection settings for a single MPC node.
type MPCNodeConfig struct {
	Addr string `yaml:"addr"`
}

// Load reads and parses the configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &cfg, nil
}
