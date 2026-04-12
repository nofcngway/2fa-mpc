package twofaService

import (
	"context"
	"time"
)

//go:generate minimock -i EventProducer -o ./mocks/ -s _mock.go

// EventProducer publishes audit events to Kafka.
type EventProducer interface {
	PublishEvent(ctx context.Context, event AuditEvent) error
	Close() error
}

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	UserID    string `json:"user_id"`
	Operation string `json:"operation"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

// NewAuditEvent creates an AuditEvent with current UTC timestamp in RFC3339 format.
func NewAuditEvent(userID, operation, status string) AuditEvent {
	return AuditEvent{
		UserID:    userID,
		Operation: operation,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    status,
	}
}
