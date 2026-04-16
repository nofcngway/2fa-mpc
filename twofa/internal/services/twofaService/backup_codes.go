package twofaService

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const (
	// backupCodeCount is the number of backup codes generated per 2FA setup.
	backupCodeCount = 10
	// bcryptCost is the bcrypt hashing cost for backup codes.
	bcryptCost = 12
)

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

// generateBackupCodes creates backupCodeCount backup codes and their bcrypt hashes.
// Returns plaintext codes (to return to user once) and hashed codes (for storage).
// Plaintext codes are in "xxxx-xxxx" format. Hashes use bcrypt cost=12.
func generateBackupCodes() ([]string, []string, error) {
	plaintextCodes := make([]string, 0, backupCodeCount)
	hashedCodes := make([]string, 0, backupCodeCount)

	for i := range backupCodeCount {
		code, err := generateBackupCode()
		if err != nil {
			return nil, nil, fmt.Errorf("generate backup code %d: %w", i, err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcryptCost)
		if err != nil {
			return nil, nil, fmt.Errorf("hash backup code %d: %w", i, err)
		}

		plaintextCodes = append(plaintextCodes, code)
		hashedCodes = append(hashedCodes, string(hash))
	}

	return plaintextCodes, hashedCodes, nil
}
