package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/vbncursed/vkr/twofa/config"
	"github.com/vbncursed/vkr/twofa/internal/api/twofa_service_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

// InitServices wires all application dependencies and returns the gRPC API handler
// along with a cleanup function that closes all connections in the correct order.
func InitServices(cfg *config.Config, logger *slog.Logger) (*twofa_service_api.TwoFAServiceAPI, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// fatalf releases the bootstrap context before exiting so deferred cleanups
	// (cancel()) actually run — `os.Exit` skips defers, so we call cancel
	// inline.
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

	// 2. Redis (session storage with rate limiting; falls back to NoOp)
	sessionStorage := NewSessionStorage(ctx, cfg)

	// 3. MPC clients
	mpcClients, mpcConns, err := NewMPCClients(cfg)
	if err != nil {
		fatalf("failed to create MPC clients", err)
	}

	// 4. Kafka
	kafkaProducer := NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 5. TwoFA service
	service, err := twofaService.NewTwoFAService(twofaService.Deps{
		Storage:        pgStorage,
		SessionStorage: sessionStorage,
		MPCClients:     mpcClients,
		EventProducer:  kafkaProducer,
		MPCTimeout:     cfg.GetMPCTimeout(),
	})
	if err != nil {
		fatalf("failed to create TwoFA service", err)
	}

	// 6. API
	api := twofa_service_api.NewTwoFAServiceAPI(service)

	cleanup := func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.Error("failed to close Kafka producer", "error", err)
		}
		logger.Info("Kafka producer closed")

		if closer, ok := sessionStorage.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logger.Error("failed to close Redis", "error", err)
			}
			logger.Info("Redis connection closed")
		}

		for i, conn := range mpcConns {
			if err := conn.Close(); err != nil {
				logger.Error("failed to close MPC connection", "index", i, "error", err)
			}
		}
		logger.Info("MPC connections closed")

		pgStorage.Close()
		logger.Info("PostgreSQL connection closed")
	}

	return api, cleanup
}
