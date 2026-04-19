// Package bootstrap provides dependency injection factories for the MPC Node service.
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
)

var (
	_ mpcService.EventProducer = (*KafkaProducer)(nil)
	_ mpcService.EventProducer = (*NoOpProducer)(nil)
)

// KafkaProducer implements EventProducer using kafka-go Writer.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer for audit events.
// Returns NoOpProducer if brokers are empty or not configured.
func NewKafkaProducer(brokers []string, topic string) mpcService.EventProducer {
	if len(brokers) == 0 || brokers[0] == "" {
		slog.Warn("Kafka not configured, audit events disabled")
		return &NoOpProducer{}
	}
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
			MaxAttempts:  3,
			Async:        true,
			ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
				slog.Error("kafka writer error", "message", fmt.Sprintf(msg, args...))
			}),
		},
	}
}

// PublishEvent sends an audit event to Kafka.
func (p *KafkaProducer) PublishEvent(ctx context.Context, event mpcService.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.UserID),
		Value: data,
	}); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// Close flushes pending messages and closes the Kafka writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

// NoOpProducer discards audit events when Kafka is unavailable.
type NoOpProducer struct{}

// PublishEvent is a no-op that always returns nil.
func (p *NoOpProducer) PublishEvent(_ context.Context, _ mpcService.AuditEvent) error {
	return nil
}

// Close is a no-op that always returns nil.
func (p *NoOpProducer) Close() error { return nil }
