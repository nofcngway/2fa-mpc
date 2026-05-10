package twofaService

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// BenchmarkGenerateBackupCodes measures the latency of generating + hashing
// 10 backup codes at the production bcrypt cost. Use to validate the parallel
// bcrypt optimization vs. a sequential baseline.
func BenchmarkGenerateBackupCodes(b *testing.B) {
	for b.Loop() {
		_, _, err := generateBackupCodes(DefaultBackupCodeBcryptCost)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateBackupCodesSerial provides the sequential baseline so the
// speedup from parallelization is measurable. Identical to the original
// implementation prior to the parallel-bcrypt change.
func BenchmarkGenerateBackupCodesSerial(b *testing.B) {
	for b.Loop() {
		plaintext := make([]string, backupCodeCount)
		hashed := make([]string, backupCodeCount)
		for i := range backupCodeCount {
			code, err := generateBackupCode()
			if err != nil {
				b.Fatal(err)
			}
			plaintext[i] = code
			hash, err := bcrypt.GenerateFromPassword([]byte(code), DefaultBackupCodeBcryptCost)
			if err != nil {
				b.Fatal(err)
			}
			hashed[i] = string(hash)
		}
	}
}
