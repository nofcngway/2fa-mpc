package bootstrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

// KafkaProducer implements EventProducer using kafka-go Writer.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer for audit events.
// Returns NoOpProducer if brokers are empty or not configured.
func NewKafkaProducer(brokers []string, topic string) authService.EventProducer {
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
		},
	}
}

// PublishEvent sends an audit event to Kafka.
func (p *KafkaProducer) PublishEvent(ctx context.Context, event authService.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.UserID),
		Value: data,
	})
}

// Close flushes pending messages and closes the Kafka writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

// NoOpProducer discards audit events when Kafka is unavailable.
type NoOpProducer struct{}

// PublishEvent is a no-op that always returns nil.
func (p *NoOpProducer) PublishEvent(_ context.Context, _ authService.AuditEvent) error {
	return nil
}

// Close is a no-op that always returns nil.
func (p *NoOpProducer) Close() error { return nil }
