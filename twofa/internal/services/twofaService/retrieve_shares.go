package twofaService

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/internal/crypto"
	"github.com/vbncursed/vkr/twofa/internal/crypto/shamir"
)

// ErrInsufficientShares is returned when fewer than 2 MPC nodes respond successfully.
var ErrInsufficientShares = fmt.Errorf("2fa: insufficient shares retrieved (need 2)")

// retrieveShares queries all 3 MPC nodes in parallel and returns the first 2
// successful share responses. Cancels remaining after 2 successes (per D-01).
// Returns ErrInsufficientShares if fewer than 2 nodes respond (per D-02).
// Logs failed nodes via slog without exposing share data (per D-03, SEC-05).
func (s *TwoFAService) retrieveShares(ctx context.Context, userID string) ([]shamir.Share, error) {
	type shareResult struct {
		share shamir.Share
		node  int
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan shareResult, 3)
	errs := make(chan error, 3)

	for i, client := range s.mpcClients {
		go func(idx int, c MPCClient) {
			callCtx, callCancel := context.WithTimeout(ctx, s.mpcTimeout)
			defer callCancel()

			shareData, err := c.RetrieveShare(callCtx, userID, idx+1)
			if err != nil {
				errs <- fmt.Errorf("node %d: %w", idx, err)
				return
			}
			results <- shareResult{
				share: shamir.Share{Index: byte(idx + 1), Data: shareData},
				node:  idx,
			}
		}(i, client)
	}

	var shares []shamir.Share
	var failures int
	for range 3 {
		select {
		case r := <-results:
			shares = append(shares, r.share)
			if len(shares) == 2 {
				cancel()
				return shares, nil
			}
		case err := <-errs:
			failures++
			slog.Warn("MPC node retrieval failed", "error", err, "user_id", userID)
			if failures > 1 {
				zeroizeShares(shares)
				return nil, ErrInsufficientShares
			}
		}
	}
	zeroizeShares(shares)
	return nil, ErrInsufficientShares
}

// zeroizeShares clears share data from memory on error paths
// where the caller's defer block won't execute.
func zeroizeShares(shares []shamir.Share) {
	for i := range shares {
		crypto.Zeroize(shares[i].Data)
	}
}
