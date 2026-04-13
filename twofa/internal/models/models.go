package models

import (
	"errors"
	"time"
)

// Domain sentinel errors shared across layers.
var (
	// ErrCounterNotFound is returned by SessionStorage.GetUsedOTPCounter when no counter exists.
	ErrCounterNotFound = errors.New("otp counter not found")
)

// TwoFARecord represents a user's 2FA enrollment record.
type TwoFARecord struct {
	UserID    string
	IsEnabled bool
	CreatedAt time.Time
}

// BackupCode represents a single backup code for 2FA recovery.
type BackupCode struct {
	ID       string
	UserID   string
	CodeHash string
	IsUsed   bool
}

// BackupCodeRow represents a stored backup code with its ID and hash.
// Used by Storage interface for backup code verification.
type BackupCodeRow struct {
	ID       string
	CodeHash string
}
