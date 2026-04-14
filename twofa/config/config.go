package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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

func envDuration(key string, target *time.Duration) {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*target = d
		}
	}
}

func envStringSlice(key string, target *[]string) {
	if v := os.Getenv(key); v != "" {
		*target = strings.Split(v, ",")
	}
}

func envMPCNodes(key string, target *[]MPCNodeConfig) {
	if v := os.Getenv(key); v != "" {
		addrs := strings.Split(v, ",")
		nodes := make([]MPCNodeConfig, len(addrs))
		for i, addr := range addrs {
			nodes[i] = MPCNodeConfig{Addr: strings.TrimSpace(addr)}
		}
		*target = nodes
	}
}

func applyEnvOverrides(cfg *Config) {
	envInt("TWOFA_SERVER_PORT", &cfg.Server.Port)
	envInt("TWOFA_SERVER_METRICS_PORT", &cfg.Server.MetricsPort)
	envString("TWOFA_SERVER_LOG_LEVEL", &cfg.Server.LogLevel)
	envString("TWOFA_DATABASE_DSN", &cfg.Database.DSN)
	envString("TWOFA_REDIS_ADDR", &cfg.Redis.Addr)
	envString("TWOFA_REDIS_PASSWORD", &cfg.Redis.Password)
	envInt("TWOFA_REDIS_DB", &cfg.Redis.DB)
	envStringSlice("TWOFA_KAFKA_BROKERS", &cfg.Kafka.Brokers)
	envString("TWOFA_KAFKA_TOPIC", &cfg.Kafka.Topic)
	envMPCNodes("TWOFA_MPC_NODES", &cfg.MPCNodes)
	envString("TWOFA_SHARED_SECRET", &cfg.SharedSecret)
	envDuration("TWOFA_MPC_TIMEOUT", &cfg.MPCTimeout)
}

// Load reads and parses the configuration file at the given path.
// If the file does not exist, configuration is loaded entirely from environment variables.
// Environment variables always override yaml values (TWOFA_* prefix).
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
