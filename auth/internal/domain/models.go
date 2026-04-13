package domain

import (
	"context"
	"time"
)

// User represents the domain model for a user account.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RefreshTokenData holds the data associated with a stored refresh token.
type RefreshTokenData struct {
	UserID      string `json:"user_id"`
	TokenFamily string `json:"token_family"`
	IssuedAt    string `json:"issued_at"`
}

// EventProducer publishes audit events to a message broker.
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
