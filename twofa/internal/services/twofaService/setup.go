package twofaService

import (
	"context"
	"errors"
)

// Setup orchestrates the 2FA setup flow: generates TOTP secret, splits via Shamir,
// distributes shares to MPC nodes, creates backup codes, and returns provisioning URI.
// Full implementation in Phase 7 Plan 02.
func (s *TwoFAService) Setup(ctx context.Context, userID, email string) (string, []string, error) {
	return "", nil, errors.New("not implemented")
}
