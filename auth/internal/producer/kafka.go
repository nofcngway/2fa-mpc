// Package producer provides message broker implementations of the publisher.EventPublisher interface.
package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/vbncursed/vkr/auth/internal/publisher"
)

// Compile-time check: KafkaProducer must implement publisher.EventPublisher.
var _ publisher.EventPublisher = (*KafkaProducer)(nil)

// KafkaProducer implements EventPublisher using kafka-go Writer.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer for audit events.
// Returns NoOpProducer if brokers are empty or not configured.
func NewKafkaProducer(brokers []string, topic string) publisher.EventPublisher {
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
func (p *KafkaProducer) PublishEvent(ctx context.Context, event publisher.AuditEvent) error {
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
