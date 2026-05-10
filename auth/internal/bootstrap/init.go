package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
	"github.com/vbncursed/vkr/auth/internal/producer"
	"github.com/vbncursed/vkr/auth/internal/services/auth_service"
)

// InitServices wires all application dependencies and returns the gRPC API handler
// along with a cleanup function that closes all connections in the correct order.
func InitServices(cfg *config.Config, logger *slog.Logger) (*auth_service_api.AuthServiceAPI, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// fatalf releases the bootstrap context before exiting so deferred
	// cleanups (cancel()) actually run — `os.Exit` skips defers.
	fatalf := func(msg string, err error) {
		logger.Error(msg, "error", err)
		cancel()
		os.Exit(1)
	}

	// 1. PostgreSQL
	pgStorage, err := NewPGStorage(ctx, cfg)
	if err != nil {
		fatalf("failed to connect to PostgreSQL", err)
	}

	// 2. Redis
	redisStorage, err := NewRedisStorage(ctx, cfg)
	if err != nil {
		fatalf("failed to connect to Redis", err)
	}

	// 3. Kafka
	eventPublisher := producer.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 4. Auth service
	privateKey, publicKey, err := auth_service.LoadRSAKeys(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		fatalf("failed to load RSA keys", fmt.Errorf("load RSA keys: %w", err))
	}

	authSvc, err := auth_service.NewAuthService(auth_service.Deps{
		Storage:         pgStorage,
		SessionStorage:  redisStorage,
		EventPublisher:  eventPublisher,
		PrivateKey:      privateKey,
		PublicKey:       publicKey,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
	if err != nil {
		fatalf("failed to create auth service", err)
	}

	// 5. API
	api := auth_service_api.NewAuthServiceAPI(authSvc)

	cleanup := func() {
		if err := eventPublisher.Close(); err != nil {
			logger.Error("failed to close Kafka producer", "error", err)
		}
		logger.Info("Kafka producer closed")

		if err := redisStorage.Close(); err != nil {
			logger.Error("failed to close Redis", "error", err)
		}
		logger.Info("Redis connection closed")

		pgStorage.Close()
		logger.Info("PostgreSQL connection closed")
	}

	return api, cleanup
}
