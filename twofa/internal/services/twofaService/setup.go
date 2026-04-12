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
	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
)

// ErrAlreadyEnabled is returned when 2FA is already active for the user.
var ErrAlreadyEnabled = errors.New("2fa: already enabled for this user")

// Setup orchestrates 2FA enrollment for a user (per 2FA-01, 2FA-02, 2FA-08, SEC-04).
//
// Flow:
//  1. Check for duplicate setup (D-12): if twofa_records has is_enabled=true, return ErrAlreadyEnabled
//  2. Generate TOTP secret; defer zeroize immediately (D-08)
//  3. Shamir split (2-of-3); defer zeroize each share's Data (D-09)
//  4. Distribute shares to 3 MPC nodes in parallel via errgroup (D-01)
//     - Per-call timeout from config (D-04)
//     - On ANY failure: compensating DeleteShare on ALL nodes with fresh context (D-02)
//  5. Create TwoFARecord with is_enabled=false (D-14)
//  6. Generate 10 backup codes, bcrypt-hash, batch store (D-05, D-06)
//  7. Build provisioning URI
//  8. Return URI + plaintext backup codes
func (s *TwoFAService) Setup(ctx context.Context, userID, email string) (string, []string, error) {
	// 1. Duplicate check (D-12)
	existing, err := s.storage.GetTwoFARecord(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("check existing 2fa: %w", err)
	}
	if existing != nil && existing.IsEnabled {
		return "", nil, ErrAlreadyEnabled
	}

	// 2. Generate TOTP secret (SEC-04: defer zeroize immediately, D-08)
	raw, base32Secret, err := totp.GenerateSecret()
	if err != nil {
		return "", nil, fmt.Errorf("generate totp secret: %w", err)
	}
	defer crypto.Zeroize(raw)

	// 3. Shamir split (2-of-3)
	shares, err := shamir.Split(raw, 3, 2)
	if err != nil {
		return "", nil, fmt.Errorf("shamir split: %w", err)
	}
	// D-09: zeroize each share's Data after distribution
	defer func() {
		for i := range shares {
			crypto.Zeroize(shares[i].Data)
		}
	}()

	// 4. Distribute shares in parallel via errgroup (D-01)
	if err := s.distributeShares(ctx, userID, shares); err != nil {
		return "", nil, fmt.Errorf("distribute shares: %w", err)
	}

	// 5. Create TwoFARecord (D-14: is_enabled=false)
	if existing == nil {
		if err := s.storage.CreateTwoFARecord(ctx, userID); err != nil {
			return "", nil, fmt.Errorf("create twofa record: %w", err)
		}
	}

	// 6. Generate backup codes (D-05, D-06)
	plaintextCodes, hashedCodes, err := generateBackupCodes()
	if err != nil {
		return "", nil, fmt.Errorf("generate backup codes: %w", err)
	}

	if err := s.storage.StoreBatchBackupCodes(ctx, userID, hashedCodes); err != nil {
		return "", nil, fmt.Errorf("store backup codes: %w", err)
	}

	// 7. Build provisioning URI
	uri := totp.GenerateProvisioningURI(base32Secret, email)

	slog.Info("2FA setup completed", "user_id", userID)

	// 8. Return URI + plaintext codes (D-06: plaintext returned only once)
	return uri, plaintextCodes, nil
}

// distributeShares sends shares to MPC nodes in parallel using errgroup (D-01).
// On ANY failure, runs compensating DeleteShare on ALL nodes with fresh context (D-02).
func (s *TwoFAService) distributeShares(ctx context.Context, userID string, shares []shamir.Share) error {
	g, gCtx := errgroup.WithContext(ctx)

	for i, share := range shares {
		i, share := i, share // capture loop variables
		g.Go(func() error {
			// D-04: per-call timeout
			callCtx, cancel := context.WithTimeout(gCtx, s.mpcTimeout)
			defer cancel()

			_, err := s.mpcClients[i].StoreShare(callCtx, &mpc_api.StoreShareRequest{
				UserId:     userID,
				ShareIndex: int32(share.Index),
				ShareData:  share.Data,
			})
			if err != nil {
				return fmt.Errorf("store share on node %d: %w", i, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// D-02: compensating delete on ALL nodes with fresh context (Pitfall 4)
		s.deleteSharesFromAllNodes(userID)
		return err
	}

	return nil
}

// deleteSharesFromAllNodes calls DeleteShare on all MPC nodes using a fresh
// background context with timeout (NOT the errgroup-cancelled context).
// Errors are logged but not propagated -- this is best-effort cleanup (D-02).
func (s *TwoFAService) deleteSharesFromAllNodes(userID string) {
	for i, client := range s.mpcClients {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), s.mpcTimeout)
		_, err := client.DeleteShare(cleanupCtx, &mpc_api.DeleteShareRequest{
			UserId: userID,
		})
		cancel()
		if err != nil {
			slog.Error("compensating delete failed", "node", i, "user_id", userID, "error", err)
		}
	}
}
