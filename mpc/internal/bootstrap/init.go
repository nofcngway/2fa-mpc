package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vbncursed/vkr/mpc/config"
	"github.com/vbncursed/vkr/mpc/internal/api/mpc_service_api"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
)

// InitServices wires all application dependencies and returns the gRPC API handler
// along with a cleanup function that closes all connections in the correct order.
func InitServices(cfg *config.Config, logger *slog.Logger) (*mpc_service_api.MPCServiceAPI, func()) {
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
	storage, err := NewPGStorage(ctx, cfg)
	if err != nil {
		fatalf("failed to connect to PostgreSQL", err)
	}

	// 2. Kafka
	kafkaProducer := NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// 3. MPC service
	key := []byte(cfg.Node.EncryptionKey)
	if len(key) != 32 {
		fatalf("invalid encryption key", fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key)))
	}

	service, err := mpcService.NewMPCService(mpcService.Deps{
		Storage:       storage,
		EncryptionKey: key,
		NodeID:        cfg.Node.ID,
		EventProducer: kafkaProducer,
	})
	if err != nil {
		fatalf("failed to create MPC service", err)
	}

	// 4. API
	api := mpc_service_api.NewMPCServiceAPI(service)

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
