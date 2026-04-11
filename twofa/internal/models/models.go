package models

import "time"

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
