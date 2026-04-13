package config

import (
	"errors"
	"os"
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

// Load reads and parses the config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Load(data, &cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
