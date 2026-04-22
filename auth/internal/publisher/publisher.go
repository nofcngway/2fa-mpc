// Package publisher defines the event publishing contract and event types
// used across the auth service for audit trail and observability.
package publisher

import (
	"context"
	"time"
)

// EventPublisher publishes audit events to a message broker.
type EventPublisher interface {
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
