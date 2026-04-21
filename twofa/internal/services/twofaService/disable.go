package twofaService

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/vbncursed/vkr/twofa/internal/crypto"
	"github.com/vbncursed/vkr/twofa/internal/crypto/shamir"
	"github.com/vbncursed/vkr/twofa/internal/crypto/totp"
	"github.com/vbncursed/vkr/twofa/internal/domain"
)

// verifyForDisable validates the user's code (backup code or TOTP) before disabling 2FA.
func (s *TwoFAService) verifyForDisable(ctx context.Context, userID, otpCode string) error {
	// Backup code path — if input matches "xxxx-xxxx" format
	if backupCodePattern.MatchString(otpCode) {
		if err := s.VerifyBackupCode(ctx, userID, otpCode); err != nil {
			return fmt.Errorf("verify backup code: %w", err)
		}
		return nil
	}

	// TOTP path
	shares, err := s.retrieveShares(ctx, userID)
	if err != nil {
		return fmt.Errorf("retrieve shares for disable: %w", err)
	}
	defer func() {
		for i := range shares {
			crypto.Zeroize(shares[i].Data)
		}
	}()

	secret, err := shamir.Combine(shares)
	if err != nil {
		return fmt.Errorf("combine shares for disable: %w", err)
	}
	defer crypto.Zeroize(secret)

	valid, matchedCounter := totp.ValidateOTPWithCounter(secret, otpCode)
	if !valid {
		return domain.ErrInvalidOTP
	}

	// Check OTP reuse (per D-10, M-13)
	lastCounter, counterErr := s.sessionStorage.GetUsedOTPCounter(ctx, userID)
	if counterErr != nil && !errors.Is(counterErr, domain.ErrCounterNotFound) {
		slog.Warn("get used OTP counter failed, proceeding", "user_id", userID, "error", counterErr)
	}
	if counterErr == nil && lastCounter == matchedCounter {
		return domain.ErrOTPReused
	}

	// Store the used counter to prevent replay
	if err := s.sessionStorage.SetUsedOTPCounter(ctx, userID, matchedCounter, otpCounterTTL); err != nil {
		slog.Warn("store used OTP counter failed", "user_id", userID, "error", err)
	}

	return nil
}

// Disable removes 2FA for a user after code verification (TOTP or backup code) (per D-12).
// Order: verify code -> delete shares (parallel) -> delete backup codes -> delete record -> cleanup Redis.
func (s *TwoFAService) Disable(ctx context.Context, userID, otpCode string) error {
	// 1. Check record exists and is enabled
	record, err := s.storage.GetTwoFARecord(ctx, userID)
	if err != nil {
		return fmt.Errorf("get twofa record: %w", err)
	}
	if record == nil {
		return domain.ErrNotSetUp
	}
	if !record.IsEnabled {
		return domain.ErrNotEnabled
	}

	// 2. Rate limit check (same pattern as Verify, per D-05, D-06)
	rateLimitKey := fmt.Sprintf("rate_limit:verify:%s", userID)
	count, rlErr := s.sessionStorage.IncrementRateLimit(ctx, rateLimitKey, rateLimitWindow)
	if rlErr != nil {
		slog.Warn("rate limit check failed, proceeding", "user_id", userID, "error", rlErr)
	} else if count > int64(rateLimitMaxAttempts) {
		return domain.ErrRateLimitExceeded
	}

	// 3. Verify code — backup code or TOTP
	if err := s.verifyForDisable(ctx, userID, otpCode); err != nil {
		return fmt.Errorf("verify for disable: %w", err)
	}

	// 4. Delete shares from ALL 3 MPC nodes in parallel (per D-12, D-13)
	if err := s.deleteSharesAll(ctx, userID); err != nil {
		return fmt.Errorf("delete shares: %w", err)
	}

	// 5. Delete backup codes (per D-12)
	if err := s.storage.DeleteBackupCodes(ctx, userID); err != nil {
		return fmt.Errorf("delete backup codes: %w", err)
	}

	// 6. Delete twofa_record (per D-12)
	if err := s.storage.DeleteTwoFARecord(ctx, userID); err != nil {
		return fmt.Errorf("delete twofa record: %w", err)
	}

	// 7. Cleanup Redis keys (per D-14)
	otpUsedKey := fmt.Sprintf("otp_used:%s", userID)
	if err := s.sessionStorage.DeleteKeys(ctx, rateLimitKey, otpUsedKey); err != nil {
		slog.Warn("failed to cleanup redis keys on disable", "user_id", userID, "error", err)
	}

	slog.Info("2FA disabled", "user_id", userID)

	// Fire-and-forget audit event
	if err := s.eventProducer.PublishEvent(ctx, NewAuditEvent(userID, "2fa.disabled", "success")); err != nil {
		slog.Warn("failed to publish audit event", "operation", "2fa.disabled", "error", err)
	}

	return nil
}

// deleteSharesAll deletes shares from all 3 MPC nodes in parallel using errgroup.
// Returns error if ANY node fails (per D-13).
func (s *TwoFAService) deleteSharesAll(ctx context.Context, userID string) error {
	g, gCtx := errgroup.WithContext(ctx)

	for i, client := range s.mpcClients {
		g.Go(func() error {
			callCtx, cancel := context.WithTimeout(gCtx, s.mpcTimeout)
			defer cancel()

			if err := client.DeleteShare(callCtx, userID); err != nil {
				return fmt.Errorf("delete share on node %d: %w", i, err)
			}
			return nil
		})
	}

	return g.Wait()
}
