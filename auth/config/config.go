// Package config loads and validates Auth service configuration from YAML and environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

// Config holds all configuration for the Auth service.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	JWT      JWTConfig      `yaml:"jwt"`
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

// KafkaConfig holds Kafka producer settings.
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

// JWTConfig holds JWT signing key paths and token TTLs.
type JWTConfig struct {
	PrivateKeyPath  string        `yaml:"private_key_path"`
	PublicKeyPath   string        `yaml:"public_key_path"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
}

// Validate checks that all required configuration fields are populated.
func (c *Config) Validate() error {
	var errs []error
	if c.Server.Port == 0 {
		errs = append(errs, errors.New("server.port is required"))
	}
	if c.Database.DSN == "" {
		errs = append(errs, errors.New("database.dsn is required"))
	}
	if c.Redis.Addr == "" {
		errs = append(errs, errors.New("redis.addr is required"))
	}
	if c.JWT.PrivateKeyPath == "" {
		errs = append(errs, errors.New("jwt.private_key_path is required"))
	}
	if c.JWT.PublicKeyPath == "" {
		errs = append(errs, errors.New("jwt.public_key_path is required"))
	}
	if c.JWT.AccessTokenTTL == 0 {
		errs = append(errs, errors.New("jwt.access_token_ttl is required"))
	}
	if c.JWT.RefreshTokenTTL == 0 {
		errs = append(errs, errors.New("jwt.refresh_token_ttl is required"))
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

func applyEnvOverrides(cfg *Config) {
	envInt("AUTH_SERVER_PORT", &cfg.Server.Port)
	envInt("AUTH_SERVER_METRICS_PORT", &cfg.Server.MetricsPort)
	envString("AUTH_SERVER_LOG_LEVEL", &cfg.Server.LogLevel)
	envString("AUTH_DATABASE_DSN", &cfg.Database.DSN)
	envString("AUTH_REDIS_ADDR", &cfg.Redis.Addr)
	envString("AUTH_REDIS_PASSWORD", &cfg.Redis.Password)
	envInt("AUTH_REDIS_DB", &cfg.Redis.DB)
	envStringSlice("AUTH_KAFKA_BROKERS", &cfg.Kafka.Brokers)
	envString("AUTH_KAFKA_TOPIC", &cfg.Kafka.Topic)
	envString("AUTH_JWT_PRIVATE_KEY_PATH", &cfg.JWT.PrivateKeyPath)
	envString("AUTH_JWT_PUBLIC_KEY_PATH", &cfg.JWT.PublicKeyPath)
	envDuration("AUTH_JWT_ACCESS_TOKEN_TTL", &cfg.JWT.AccessTokenTTL)
	envDuration("AUTH_JWT_REFRESH_TOKEN_TTL", &cfg.JWT.RefreshTokenTTL)
}

// Load reads and parses the config file at the given path.
// If the file does not exist, configuration is loaded entirely from environment variables.
// Environment variables always override yaml values (AUTH_* prefix).
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
