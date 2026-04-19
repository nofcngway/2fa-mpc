package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vbncursed/vkr/auth/config"
	"github.com/vbncursed/vkr/auth/internal/api/auth_service_api"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

// InitServices wires all application dependencies and returns the gRPC API handler
// along with a cleanup function that closes all connections in the correct order.
func InitServices(cfg *config.Config, logger *slog.Logger) (*auth_service_api.AuthServiceAPI, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. PostgreSQL
	pgStorage, err := NewPGStorage(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// 2. Redis
	redisStorage, err := NewRedisStorage(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	// 3. Kafka
	kafkaProducer := NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 4. Auth service
	privateKey, publicKey, err := authService.LoadRSAKeys(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath)
	if err != nil {
		logger.Error("failed to load RSA keys", "error", fmt.Errorf("load RSA keys: %w", err))
		os.Exit(1)
	}

	authSvc, err := authService.NewAuthService(authService.Deps{
		Storage:         pgStorage,
		SessionStorage:  redisStorage,
		EventProducer:   kafkaProducer,
		PrivateKey:      privateKey,
		PublicKey:        publicKey,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
	if err != nil {
		logger.Error("failed to create auth service", "error", err)
		os.Exit(1)
	}

	// 5. API
	api := auth_service_api.NewAuthServiceAPI(authSvc)

	cleanup := func() {
		if err := kafkaProducer.Close(); err != nil {
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
