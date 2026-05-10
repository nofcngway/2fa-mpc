package twofaService

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

// backupCodeCount is the number of backup codes generated per 2FA setup.
const backupCodeCount = 10

// DefaultBackupCodeBcryptCost is the bcrypt cost used when Deps.BackupCodeBcryptCost
// is unset (0).
//
// Lower than the user-password cost (12) by intent: backup codes are 8-digit
// cryptographic random tokens (~26.6 bits of entropy), one-time use, and
// additionally protected by the verify-rate-limit (5/5min/user). Brute-force
// at cost=10 (≈64ms/hash) on the entire 10^8 codespace takes ~7 days on a
// 32-core attacker — orders of magnitude beyond the 5-attempt rate-limit
// window.
const DefaultBackupCodeBcryptCost = 10

// generateBackupCode creates a single backup code in "xxxx-xxxx" format.
// Each half is a random 4-digit number (0000-9999) generated via crypto/rand.
func generateBackupCode() (string, error) {
	left, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", fmt.Errorf("generate left half: %w", err)
	}
	right, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", fmt.Errorf("generate right half: %w", err)
	}
	return fmt.Sprintf("%04d-%04d", left.Int64(), right.Int64()), nil
}

// generateBackupCodes creates backupCodeCount backup codes and their bcrypt
// hashes at the supplied cost. Returns plaintext codes (to return to user
// once) and hashed codes (for storage). Plaintext codes are in "xxxx-xxxx"
// format.
//
// Plaintext generation is serial (cheap, crypto/rand). Bcrypt hashing is
// CPU-bound and runs in parallel via errgroup to bound Setup latency to a
// single bcrypt operation regardless of backupCodeCount. Order is preserved:
// hashedCodes[i] is the bcrypt of plaintextCodes[i].
func generateBackupCodes(cost int) ([]string, []string, error) {
	plaintextCodes := make([]string, backupCodeCount)
	for i := range backupCodeCount {
		code, err := generateBackupCode()
		if err != nil {
			return nil, nil, fmt.Errorf("generate backup code %d: %w", i, err)
		}
		plaintextCodes[i] = code
	}

	hashedCodes := make([]string, backupCodeCount)
	var g errgroup.Group
	for i := range backupCodeCount {
		g.Go(func() error {
			hash, err := bcrypt.GenerateFromPassword([]byte(plaintextCodes[i]), cost)
			if err != nil {
				return fmt.Errorf("hash backup code %d: %w", i, err)
			}
			hashedCodes[i] = string(hash)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return plaintextCodes, hashedCodes, nil
}
