package authService

import "github.com/vbncursed/vkr/auth/internal/domain"

//go:generate minimock -i EventProducer -o ./mocks/ -s _mock.go

// EventProducer is an alias for domain.EventProducer.
type EventProducer = domain.EventProducer

// AuditEvent is an alias for domain.AuditEvent.
type AuditEvent = domain.AuditEvent

// NewAuditEvent delegates to domain.NewAuditEvent.
var NewAuditEvent = domain.NewAuditEvent
