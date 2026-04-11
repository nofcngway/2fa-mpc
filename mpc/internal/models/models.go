// Package models defines domain models for the MPC service.
package models

import "time"

// Share represents an encrypted share stored by this MPC node.
type Share struct {
	ID            string
	UserID        string
	ShareIndex    int
	EncryptedData []byte
	Nonce         []byte
	CreatedAt     time.Time
}
