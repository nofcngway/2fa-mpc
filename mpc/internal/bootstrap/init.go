package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
)

// InitServices wires all application dependencies and returns the gRPC API handler
// along with a cleanup function that closes all connections in the correct order.
func InitServices(cfg *config.Config, logger *slog.Logger) (*mpc_service_api.MPCServiceAPI, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. PostgreSQL
	storage, err := NewPGStorage(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// 2. Kafka
	kafkaProducer := NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 3. MPC service
	service, err := NewMPCService(storage, cfg, kafkaProducer)
	if err != nil {
		logger.Error("failed to create MPC service", "error", err)
		os.Exit(1)
	}

	// 4. API
	api := NewMPCServiceAPI(service)

	cleanup := func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.Error("failed to close Kafka producer", "error", err)
		}
		logger.Info("Kafka producer closed")

		storage.Close()
		logger.Info("PostgreSQL connection closed")
	}

	return api, cleanup
}
