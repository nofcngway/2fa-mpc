package domain

import (
	"errors"
	"strings"
)

// Authentication errors.
var (
	ErrDuplicateEmail = errors.New("user with this email already exists")
	ErrInvalidEmail   = errors.New("invalid email format")
)

// Password validation errors.
var (
	ErrPasswordTooShort   = errors.New("password must be at least 12 characters")
	ErrMissingUppercase   = errors.New("password must contain at least one uppercase letter")
	ErrMissingLowercase   = errors.New("password must contain at least one lowercase letter")
	ErrMissingDigit       = errors.New("password must contain at least one digit")
	ErrMissingSpecialChar = errors.New("password must contain at least one special character")
	ErrSequentialChars    = errors.New("password must not contain 4 or more sequential characters")
	ErrRepeatedChars      = errors.New("password must not contain 4 or more identical consecutive characters")
)

// PasswordValidationError aggregates all password validation violations.
type PasswordValidationError struct {
	Violations []error
}

// Error returns a semicolon-separated string of all violation messages.
func (e *PasswordValidationError) Error() string {
	msgs := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		msgs[i] = v.Error()
	}
	return strings.Join(msgs, "; ")
}
