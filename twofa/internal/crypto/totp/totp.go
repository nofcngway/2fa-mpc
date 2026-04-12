package totp

import "errors"

// ErrSecretGeneration is returned when crypto/rand fails to generate random bytes.
var ErrSecretGeneration = errors.New("totp: failed to generate random secret")

// GenerateSecret creates a new 20-byte cryptographically random TOTP secret.
// Returns raw bytes, base32-encoded string (no padding), and error.
func GenerateSecret() ([]byte, string, error) { return nil, "", nil }

// GenerateOTP computes a 6-digit TOTP code for the given unix timestamp.
func GenerateOTP(secret []byte, unixTime int64) string { return "" }

// ValidateOTP checks if code is valid for the current time +-1 window.
func ValidateOTP(secret []byte, code string) bool { return false }

// validateOTPAt is the testable core (unexported).
func validateOTPAt(secret []byte, code string, unixTime int64) bool { return false }

// hotp computes a 6-digit HMAC-based OTP for the given counter (unexported).
func hotp(secret []byte, counter uint64) string { return "" }
