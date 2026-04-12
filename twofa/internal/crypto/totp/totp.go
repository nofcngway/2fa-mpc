package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrSecretGeneration is returned when crypto/rand fails to generate random bytes.
var ErrSecretGeneration = errors.New("totp: failed to generate random secret")

// GenerateSecret creates a new 20-byte cryptographically random TOTP secret.
// Returns raw bytes, base32-encoded string (no padding), and error.
func GenerateSecret() ([]byte, string, error) {
	secret := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrSecretGeneration, err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return secret, encoded, nil
}

// GenerateOTP computes a 6-digit TOTP code for the given unix timestamp.
func GenerateOTP(secret []byte, unixTime int64) string {
	counter := uint64(unixTime) / 30
	return hotp(secret, counter)
}

// ValidateOTP checks if code is valid for the current time +-1 window.
func ValidateOTP(secret []byte, code string) bool {
	return validateOTPAt(secret, code, time.Now().Unix())
}

// validateOTPAt is the testable core -- checks code against T-1, T, T+1.
func validateOTPAt(secret []byte, code string, unixTime int64) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}

	counter := uint64(unixTime) / 30

	// Check current and next window always; check previous only if counter > 0.
	if subtle.ConstantTimeCompare([]byte(hotp(secret, counter)), []byte(code)) == 1 {
		return true
	}
	if subtle.ConstantTimeCompare([]byte(hotp(secret, counter+1)), []byte(code)) == 1 {
		return true
	}
	if counter > 0 {
		if subtle.ConstantTimeCompare([]byte(hotp(secret, counter-1)), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// ValidateOTPWithCounter checks if code is valid for the current time +-1 window
// and returns the matched counter value for reuse prevention.
// Returns (false, 0) if no window matches.
func ValidateOTPWithCounter(secret []byte, code string) (bool, int64) {
	return validateOTPWithCounterAt(secret, code, time.Now().Unix())
}

// validateOTPWithCounterAt is the testable core for ValidateOTPWithCounter.
func validateOTPWithCounterAt(secret []byte, code string, unixTime int64) (bool, int64) {
	if len(code) != 6 {
		return false, 0
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false, 0
		}
	}
	counter := int64(unixTime) / 30
	if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter))), []byte(code)) == 1 {
		return true, counter
	}
	if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter+1))), []byte(code)) == 1 {
		return true, counter + 1
	}
	if counter > 0 {
		if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(counter-1))), []byte(code)) == 1 {
			return true, counter - 1
		}
	}
	return false, 0
}

// hotp computes a 6-digit HMAC-based OTP for the given counter.
// Implements dynamic truncation per RFC 4226 Section 5.4.
// Panics if secret is empty — indicates a bug in the caller.
func hotp(secret []byte, counter uint64) string {
	if len(secret) == 0 {
		panic("totp: hotp called with empty secret")
	}

	// Encode counter as 8-byte big-endian.
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	// Compute HMAC-SHA1.
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	h := mac.Sum(nil) // 20 bytes

	// Dynamic truncation (RFC 4226 Section 5.4).
	offset := h[len(h)-1] & 0x0F
	truncated := uint32(h[offset])<<24 |
		uint32(h[offset+1])<<16 |
		uint32(h[offset+2])<<8 |
		uint32(h[offset+3])
	truncated &= 0x7FFFFFFF

	// Compute 6-digit code with zero-padding.
	code := truncated % 1_000_000
	return fmt.Sprintf("%06d", code)
}
