package twofaService

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BACKUP_CODE_COUNT is the number of backup codes generated per 2FA setup.
	BACKUP_CODE_COUNT = 10
	// COST_BCRYPT is the bcrypt hashing cost for backup codes.
	COST_BCRYPT = 12
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

// generateBackupCodes creates BACKUP_CODE_COUNT backup codes and their bcrypt hashes.
// Returns plaintext codes (to return to user once) and hashed codes (for storage).
// Plaintext codes are in "xxxx-xxxx" format. Hashes use bcrypt cost=12.
func generateBackupCodes() ([]string, []string, error) {
	plaintextCodes := make([]string, 0, BACKUP_CODE_COUNT)
	hashedCodes := make([]string, 0, BACKUP_CODE_COUNT)

	for i := range BACKUP_CODE_COUNT {
		code, err := generateBackupCode()
		if err != nil {
			return nil, nil, fmt.Errorf("generate backup code %d: %w", i, err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(code), COST_BCRYPT)
		if err != nil {
			return nil, nil, fmt.Errorf("hash backup code %d: %w", i, err)
		}

		plaintextCodes = append(plaintextCodes, code)
		hashedCodes = append(hashedCodes, string(hash))
	}

	return plaintextCodes, hashedCodes, nil
}
