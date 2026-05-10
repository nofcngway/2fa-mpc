// Package config loads and validates API Gateway configuration from YAML and environment variables.
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

// Default timeouts for the HTTP server.
const (
	DefaultReadTimeout  = 10 * time.Second
	DefaultWriteTimeout = 15 * time.Second
)

// Config holds all configuration for the API Gateway.
type Config struct {
	Server       ServerConfig      `yaml:"server"`
	AuthService  ServiceAddrConfig `yaml:"auth_service"`
	TwoFAService ServiceAddrConfig `yaml:"twofa_service"`
	Redis        RedisConfig       `yaml:"redis"`
	RateLimit    RateLimitConfig   `yaml:"rate_limit"`
	CORS         CORSConfig        `yaml:"cors"`
	Swagger      SwaggerConfig     `yaml:"swagger"`
	TLS          TLSConfig         `yaml:"tls"`
	Prometheus   PrometheusConfig  `yaml:"prometheus"`
}

// PrometheusConfig points the Gateway at the Prometheus query API used by the
// monitoring snapshot endpoint. Empty URL disables the endpoint.
type PrometheusConfig struct {
	URL string `yaml:"url"`
}

// TLSConfig configures mTLS for the gRPC client connections to Auth and
// TwoFA. When Enabled, the gateway presents CertFile/KeyFile and validates
// each downstream server cert against CAFile.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         int           `yaml:"port"`
	MetricsPort  int           `yaml:"metrics_port"`
	LogLevel     string        `yaml:"log_level"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// GetReadTimeout returns the configured read timeout or the default (10s).
func (s *ServerConfig) GetReadTimeout() time.Duration {
	return cmp.Or(s.ReadTimeout, DefaultReadTimeout)
}

// GetWriteTimeout returns the configured write timeout or the default (15s).
func (s *ServerConfig) GetWriteTimeout() time.Duration {
	return cmp.Or(s.WriteTimeout, DefaultWriteTimeout)
}

// ServiceAddrConfig holds connection settings for a downstream gRPC service.
type ServiceAddrConfig struct {
	Addr string `yaml:"addr"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// RateLimitConfig holds rate-limiting settings.
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	Burst             int `yaml:"burst"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// SwaggerConfig holds paths to Swagger/OpenAPI definitions.
type SwaggerConfig struct {
	Auth  string `yaml:"auth"`
	TwoFA string `yaml:"twofa"`
}

// Validate checks that all required configuration fields are populated.
func (c *Config) Validate() error {
	var errs []error
	if c.Server.Port == 0 {
		errs = append(errs, errors.New("server.port is required"))
	}
	if c.AuthService.Addr == "" {
		errs = append(errs, errors.New("auth_service.addr is required"))
	}
	if c.TwoFAService.Addr == "" {
		errs = append(errs, errors.New("twofa_service.addr is required"))
	}
	if c.Redis.Addr == "" {
		errs = append(errs, errors.New("redis.addr is required"))
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

func envBool(key string, target *bool) {
	if v := os.Getenv(key); v != "" {
		*target = v == "true" || v == "1"
	}
}

func applyEnvOverrides(cfg *Config) {
	envInt("GATEWAY_SERVER_PORT", &cfg.Server.Port)
	envInt("GATEWAY_SERVER_METRICS_PORT", &cfg.Server.MetricsPort)
	envString("GATEWAY_SERVER_LOG_LEVEL", &cfg.Server.LogLevel)
	envDuration("GATEWAY_SERVER_READ_TIMEOUT", &cfg.Server.ReadTimeout)
	envDuration("GATEWAY_SERVER_WRITE_TIMEOUT", &cfg.Server.WriteTimeout)
	envString("GATEWAY_AUTH_SERVICE_ADDR", &cfg.AuthService.Addr)
	envString("GATEWAY_TWOFA_SERVICE_ADDR", &cfg.TwoFAService.Addr)
	envString("GATEWAY_REDIS_ADDR", &cfg.Redis.Addr)
	envString("GATEWAY_REDIS_PASSWORD", &cfg.Redis.Password)
	envInt("GATEWAY_REDIS_DB", &cfg.Redis.DB)
	envInt("GATEWAY_RATE_LIMIT_REQUESTS_PER_MINUTE", &cfg.RateLimit.RequestsPerMinute)
	envInt("GATEWAY_RATE_LIMIT_BURST", &cfg.RateLimit.Burst)
	envStringSlice("GATEWAY_CORS_ALLOWED_ORIGINS", &cfg.CORS.AllowedOrigins)
	envString("GATEWAY_SWAGGER_AUTH", &cfg.Swagger.Auth)
	envString("GATEWAY_SWAGGER_TWOFA", &cfg.Swagger.TwoFA)
	envBool("GATEWAY_TLS_ENABLED", &cfg.TLS.Enabled)
	envString("GATEWAY_TLS_CERT_FILE", &cfg.TLS.CertFile)
	envString("GATEWAY_TLS_KEY_FILE", &cfg.TLS.KeyFile)
	envString("GATEWAY_TLS_CA_FILE", &cfg.TLS.CAFile)
	envString("GATEWAY_PROMETHEUS_URL", &cfg.Prometheus.URL)
}

// Load reads and parses the config file at the given path.
// If the file does not exist, configuration is loaded entirely from environment variables.
// Environment variables always override yaml values (GATEWAY_* prefix).
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
