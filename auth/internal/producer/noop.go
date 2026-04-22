package producer

import (
	"context"

	"github.com/vbncursed/vkr/auth/internal/publisher"
)

// Compile-time check: NoOpProducer must implement publisher.EventPublisher.
var _ publisher.EventPublisher = (*NoOpProducer)(nil)

// NoOpProducer discards audit events when Kafka is unavailable.
type NoOpProducer struct{}

// PublishEvent is a no-op that always returns nil.
func (p *NoOpProducer) PublishEvent(_ context.Context, _ publisher.AuditEvent) error {
	return nil
}

// Close is a no-op that always returns nil.
func (p *NoOpProducer) Close() error { return nil }
