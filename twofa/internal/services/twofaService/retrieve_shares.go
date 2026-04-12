package twofaService

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vbncursed/vkr/twofa/internal/crypto/shamir"
	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
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

			resp, err := c.RetrieveShare(callCtx, &mpc_api.RetrieveShareRequest{
				UserId:     userID,
				ShareIndex: int32(idx + 1),
			})
			if err != nil {
				errs <- fmt.Errorf("node %d: %w", idx, err)
				return
			}
			results <- shareResult{
				share: shamir.Share{Index: byte(idx + 1), Data: resp.ShareData},
				node:  idx,
			}
		}(i, client)
	}

	var shares []shamir.Share
	var failures int
	for i := 0; i < 3; i++ {
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
				return nil, ErrInsufficientShares
			}
		}
	}
	return nil, ErrInsufficientShares
}
