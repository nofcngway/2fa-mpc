package twofaService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/vbncursed/vkr/twofa/internal/crypto"
	"github.com/vbncursed/vkr/twofa/internal/crypto/shamir"
	"github.com/vbncursed/vkr/twofa/internal/crypto/totp"
	"github.com/vbncursed/vkr/twofa/internal/models"
)

// Domain errors for 2FA verification.
var (
	ErrRateLimitExceeded = errors.New("2fa: rate limit exceeded")
	ErrOTPReused         = errors.New("2fa: OTP code already used")
	ErrNotSetUp          = errors.New("2fa: not set up for this user")
)

// backupCodePattern matches the "xxxx-xxxx" backup code format.
var backupCodePattern = regexp.MustCompile(`^\d{4}-\d{4}$`)

const (
	rateLimitMaxAttempts = 5
	rateLimitWindow      = 5 * time.Minute // 300 seconds (per D-05)
	otpCounterTTL        = 90 * time.Second // covers 3 TOTP windows (per D-09)
)

// Verify orchestrates OTP verification with rate limiting, reuse prevention, and
// first-verify enablement (per 2FA-03, 2FA-04, 2FA-05, 2FA-09).
//
// Returns (valid, isNewlyEnabled, error):
//   - valid: whether the OTP code matched
//   - isNewlyEnabled: true if this was the first successful verification (enabling 2FA)
//   - error: domain errors (ErrNotSetUp, ErrRateLimitExceeded, ErrOTPReused) or internal errors
//
// Flow:
//  1. Check TwoFARecord exists
//  2. Rate limit check (increment before validation, per D-06)
//  3. Retrieve 2 shares from MPC nodes (first-2-wins)
//  4. Shamir combine to reconstruct secret
//  5. OTP reuse check
//  6. TOTP validation
//  7. Store used counter
//  8. Enable on first successful verification
func (s *TwoFAService) Verify(ctx context.Context, userID, otpCode string) (bool, bool, error) {
	// 1. Check TwoFARecord exists
	record, err := s.storage.GetTwoFARecord(ctx, userID)
	if err != nil {
		return false, false, fmt.Errorf("get twofa record: %w", err)
	}
	if record == nil {
		return false, false, ErrNotSetUp
	}

	// 2. Rate limit check (per D-05, D-06: increment BEFORE validation)
	rateLimitKey := fmt.Sprintf("rate_limit:verify:%s", userID)
	count, err := s.sessionStorage.IncrementRateLimit(ctx, rateLimitKey, rateLimitWindow)
	if err != nil {
		// D-07: proceed without rate check on Redis failure
		slog.Warn("rate limit check failed, proceeding", "user_id", userID, "error", err)
	} else if count > int64(rateLimitMaxAttempts) {
		return false, false, ErrRateLimitExceeded
	}

	// 3. Backup code path — if input matches "xxxx-xxxx" format, verify as backup code
	if backupCodePattern.MatchString(otpCode) {
		if err := s.VerifyBackupCode(ctx, userID, otpCode); err != nil {
			if errors.Is(err, ErrInvalidBackupCode) {
				return false, false, nil
			}
			return false, false, fmt.Errorf("verify backup code: %w", err)
		}
		return true, false, nil
	}

	// 4. Retrieve shares (first-2-wins, per D-01)
	shares, err := s.retrieveShares(ctx, userID)
	if err != nil {
		return false, false, fmt.Errorf("retrieve shares: %w", err)
	}
	// D-04: zeroize each share's Data after use
	defer func() {
		for i := range shares {
			crypto.Zeroize(shares[i].Data)
		}
	}()

	// 4. Shamir combine to reconstruct secret
	secret, err := shamir.Combine(shares)
	if err != nil {
		return false, false, fmt.Errorf("shamir combine: %w", err)
	}
	// D-04: zeroize combined secret after use
	defer crypto.Zeroize(secret)

	// 5. OTP reuse check (per D-09, D-10, M-04, M-13)
	lastCounter, counterErr := s.sessionStorage.GetUsedOTPCounter(ctx, userID)
	if counterErr != nil && !errors.Is(counterErr, models.ErrCounterNotFound) {
		slog.Warn("get used OTP counter failed, proceeding", "user_id", userID, "error", counterErr)
	}

	// 6. TOTP validation (per 2FA-03)
	valid, matchedCounter := totp.ValidateOTPWithCounter(secret, otpCode)
	if !valid {
		return false, false, nil
	}

	// Check OTP reuse — only when a previous counter was found (per D-10, M-13)
	if counterErr == nil && lastCounter == matchedCounter {
		return false, false, ErrOTPReused
	}

	// 7. Store used counter (per D-09)
	if err := s.sessionStorage.SetUsedOTPCounter(ctx, userID, matchedCounter, otpCounterTTL); err != nil {
		slog.Warn("store used OTP counter failed", "user_id", userID, "error", err)
	}

	// 8. Enable on first successful verification (per 2FA-04)
	if !record.IsEnabled {
		if err := s.storage.EnableTwoFA(ctx, userID); err != nil {
			return false, false, fmt.Errorf("enable twofa: %w", err)
		}
		slog.Info("2FA enabled on first verification", "user_id", userID)

		// Fire-and-forget audit event
		if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "2fa.verified", "success")); err != nil {
			slog.Warn("failed to publish audit event", "operation", "2fa.verified", "error", err)
		}

		return true, true, nil
	}

	slog.Info("2FA verification successful", "user_id", userID)

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "2fa.verified", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "2fa.verified", "error", err)
	}

	return true, false, nil
}
