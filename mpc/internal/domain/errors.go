package domain

import "errors"

// Domain errors for the MPC service.
var (
	// ErrDuplicateShare is returned when a share with the same (user_id, share_index) already exists.
	ErrDuplicateShare = errors.New("duplicate share")

	// ErrShareNotFound is returned when no share matches the query.
	ErrShareNotFound = errors.New("share not found")
)
