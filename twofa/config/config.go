package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
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
	return cmp.Or(c.MPCTimeout, DefaultMPCTimeout)
}

// Validate checks all required configuration fields.
func (c *Config) Validate() error {
	var errs []error
	if c.Server.Port == 0 {
		errs = append(errs, fmt.Errorf("server.port is required"))
	}
	if c.Database.DSN == "" {
		errs = append(errs, fmt.Errorf("database.dsn is required"))
	}
	if c.SharedSecret == "" {
		errs = append(errs, fmt.Errorf("shared_secret is required"))
	}
	if len(c.MPCNodes) != 3 {
		errs = append(errs, fmt.Errorf("exactly 3 mpc_nodes required for Shamir 2-of-3, got %d", len(c.MPCNodes)))
	}
	for i, node := range c.MPCNodes {
		if node.Addr == "" {
			errs = append(errs, fmt.Errorf("mpc_nodes[%d].addr is required", i))
		}
	}
	return errors.Join(errs...)
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
// Environment variable overrides: TWOFA_SHARED_SECRET, TWOFA_DATABASE_DSN.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Load(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Environment variable overrides for secrets
	if v := os.Getenv("TWOFA_SHARED_SECRET"); v != "" {
		cfg.SharedSecret = v
	}
	if v := os.Getenv("TWOFA_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}
