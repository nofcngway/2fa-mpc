package shamir

import (
	"crypto/rand"
	"errors"
	"io"

	"github.com/vbncursed/vkr/twofa/internal/crypto"
)

// Share represents a single share from Shamir Secret Sharing.
// Index is the 1-based x-coordinate (evaluation point), and Data contains
// the y-values for each byte of the secret polynomial.
type Share struct {
	Index byte   // x-coordinate: 1, 2, 3, ... (1-based, never 0)
	Data  []byte // y-values, len == len(secret)
}

// Sentinel errors for Split and Combine input validation.
var (
	ErrEmptySecret       = errors.New("shamir: secret must not be empty")
	ErrThresholdTooLow   = errors.New("shamir: threshold must be at least 2")
	ErrNotEnoughShares   = errors.New("shamir: n must be >= threshold")
	ErrTooManyShares     = errors.New("shamir: n must be <= 255")
	ErrDuplicateIndex    = errors.New("shamir: duplicate share index")
	ErrTooFewShares      = errors.New("shamir: need at least 2 shares to combine")
	ErrEmptyShareData    = errors.New("shamir: share data must not be empty")
	ErrShareDataMismatch = errors.New("shamir: all shares must have equal data length")
	ErrZeroIndex         = errors.New("shamir: share index must not be 0 (reserved for secret)")
)

// evalPolynomial evaluates a polynomial at point x in GF(256) using Horner's method.
// coeffs[0] is the constant term (the secret byte), coeffs[1] is the x coefficient, etc.
func evalPolynomial(coeffs []byte, x byte) byte {
	result := byte(0)
	for i := len(coeffs) - 1; i >= 0; i-- {
		result = gfAdd(gfMul(result, x), coeffs[i])
	}
	return result
}

// lagrangeInterpolateAtZero reconstructs the secret (f(0)) from a set of
// evaluation points (xs) and their values (ys) using Lagrange interpolation
// in GF(256). In GF(2^n), subtraction is identical to addition (XOR).
func lagrangeInterpolateAtZero(xs, ys []byte) byte {
	secret := byte(0)
	for i := range xs {
		// Compute Lagrange basis polynomial L_i(0).
		// L_i(0) = product over j!=i of (0 - x_j) / (x_i - x_j)
		// In GF(2^n): 0 - x_j = x_j and x_i - x_j = x_i XOR x_j = gfAdd(x_i, x_j)
		basis := byte(1)
		for j := range xs {
			if i == j {
				continue
			}
			basis = gfMul(basis, gfDiv(xs[j], gfAdd(xs[i], xs[j])))
		}
		secret = gfAdd(secret, gfMul(ys[i], basis))
	}
	return secret
}

// Split divides a secret into n shares such that any threshold shares can
// reconstruct the original secret using Lagrange interpolation in GF(256).
// Each byte of the secret is treated independently with a fresh random polynomial.
// Share indices are 1-based (1, 2, ..., n); x=0 is reserved for the secret value.
func Split(secret []byte, n, threshold int) ([]Share, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}
	if threshold < 2 {
		return nil, ErrThresholdTooLow
	}
	if n < threshold {
		return nil, ErrNotEnoughShares
	}
	if n > 255 {
		return nil, ErrTooManyShares
	}

	// Initialize shares with 1-based indices.
	shares := make([]Share, n)
	for i := range shares {
		shares[i] = Share{
			Index: byte(i + 1),
			Data:  make([]byte, len(secret)),
		}
	}

	// For each byte of the secret, create a random polynomial of degree (threshold-1)
	// where the constant term is the secret byte, then evaluate at each share index.
	coeffs := make([]byte, threshold)
	defer crypto.Zeroize(coeffs)

	for byteIdx, secretByte := range secret {
		coeffs[0] = secretByte

		// Fill random coefficients for degrees 1..threshold-1.
		if _, err := io.ReadFull(rand.Reader, coeffs[1:]); err != nil {
			return nil, err
		}

		// Evaluate polynomial at each share's x-coordinate.
		for i := range shares {
			shares[i].Data[byteIdx] = evalPolynomial(coeffs, shares[i].Index)
		}
	}

	return shares, nil
}

// Combine reconstructs the original secret from a set of shares using
// Lagrange interpolation at x=0 in GF(256). At least 2 shares are required
// (matching the minimum threshold). All shares must have the same Data length
// and unique indices.
func Combine(shares []Share) ([]byte, error) {
	if len(shares) < 2 {
		return nil, ErrTooFewShares
	}

	// Check for empty share data.
	for _, s := range shares {
		if len(s.Data) == 0 {
			return nil, ErrEmptyShareData
		}
	}

	// Check all shares have equal data length.
	dataLen := len(shares[0].Data)
	for _, s := range shares[1:] {
		if len(s.Data) != dataLen {
			return nil, ErrShareDataMismatch
		}
	}

	// Check for zero and duplicate indices.
	seen := make(map[byte]bool, len(shares))
	for _, s := range shares {
		if s.Index == 0 {
			return nil, ErrZeroIndex
		}
		if seen[s.Index] {
			return nil, ErrDuplicateIndex
		}
		seen[s.Index] = true
	}

	// Extract x-coordinates.
	xs := make([]byte, len(shares))
	for i, s := range shares {
		xs[i] = s.Index
	}

	// Reconstruct each byte of the secret via Lagrange interpolation at x=0.
	secret := make([]byte, dataLen)
	ys := make([]byte, len(shares))
	for byteIdx := range dataLen {
		for i, s := range shares {
			ys[i] = s.Data[byteIdx]
		}
		secret[byteIdx] = lagrangeInterpolateAtZero(xs, ys)
	}

	return secret, nil
}
