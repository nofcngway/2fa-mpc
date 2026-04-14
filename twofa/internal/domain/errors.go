package domain

import "errors"

// 2FA state errors.
var (
	ErrAlreadyEnabled = errors.New("2fa: already enabled for this user")
	ErrNotEnabled     = errors.New("2fa: not enabled for this user")
	ErrNotSetUp       = errors.New("2fa: not set up for this user")
)

// Verification errors.
var (
	ErrRateLimitExceeded  = errors.New("2fa: rate limit exceeded")
	ErrOTPReused          = errors.New("2fa: OTP code already used")
	ErrInsufficientShares = errors.New("2fa: insufficient shares retrieved (need 2)")
	ErrInvalidBackupCode  = errors.New("2fa: invalid backup code")
)

// Storage errors.
var (
	ErrCounterNotFound = errors.New("otp counter not found")
)
