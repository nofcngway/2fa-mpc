package domain

import "time"

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
