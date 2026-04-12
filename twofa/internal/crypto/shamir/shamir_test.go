package shamir

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
)

// --- Split/Combine roundtrip tests ---

func TestSplit_Combine_AllPairs_20Bytes(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
	for _, p := range pairs {
		recovered, err := Combine([]Share{shares[p[0]], shares[p[1]]})
		if err != nil {
			t.Fatalf("Combine(%d,%d): %v", p[0], p[1], err)
		}
		if !bytes.Equal(recovered, secret) {
			t.Errorf("Combine(%d,%d) did not recover secret", p[0], p[1])
		}
	}
}

func TestSplit_Combine_AllPairs_1Byte(t *testing.T) {
	secret := []byte{0x42}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
	for _, p := range pairs {
		recovered, err := Combine([]Share{shares[p[0]], shares[p[1]]})
		if err != nil {
			t.Fatalf("Combine(%d,%d): %v", p[0], p[1], err)
		}
		if !bytes.Equal(recovered, secret) {
			t.Errorf("Combine(%d,%d) did not recover secret", p[0], p[1])
		}
	}
}

func TestSplit_Combine_AllPairs_32Bytes(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
	for _, p := range pairs {
		recovered, err := Combine([]Share{shares[p[0]], shares[p[1]]})
		if err != nil {
			t.Fatalf("Combine(%d,%d): %v", p[0], p[1], err)
		}
		if !bytes.Equal(recovered, secret) {
			t.Errorf("Combine(%d,%d) did not recover secret", p[0], p[1])
		}
	}
}

func TestSplit_Combine_AllThreeShares(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	recovered, err := Combine(shares)
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if !bytes.Equal(recovered, secret) {
		t.Error("Combine with all 3 shares did not recover secret")
	}
}

// --- Security property tests ---

func TestCombine_SingleShare_DoesNotRecover(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	_, err = Combine([]Share{shares[0]})
	if !errors.Is(err, ErrTooFewShares) {
		t.Errorf("expected ErrTooFewShares, got %v", err)
	}
}

func TestSplit_SharesAreDifferent(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	// Each share's Data should be different from the others and from the secret.
	if bytes.Equal(shares[0].Data, shares[1].Data) {
		t.Error("share 0 and share 1 have identical Data")
	}
	if bytes.Equal(shares[0].Data, shares[2].Data) {
		t.Error("share 0 and share 2 have identical Data")
	}
	if bytes.Equal(shares[1].Data, shares[2].Data) {
		t.Error("share 1 and share 2 have identical Data")
	}
	if bytes.Equal(shares[0].Data, secret) {
		t.Error("share 0 Data is identical to secret")
	}
}

func TestSplit_DifferentCallsDifferentShares(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares1, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split 1: %v", err)
	}

	shares2, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split 2: %v", err)
	}

	// With 20 random bytes of coefficients, probability of collision is negligible.
	allSame := true
	for i := range shares1 {
		if !bytes.Equal(shares1[i].Data, shares2[i].Data) {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("two Split calls on same secret produced identical shares")
	}
}

// --- Input validation: Split errors ---

func TestSplit_EmptySecret(t *testing.T) {
	_, err := Split([]byte{}, 3, 2)
	if !errors.Is(err, ErrEmptySecret) {
		t.Errorf("expected ErrEmptySecret, got %v", err)
	}
}

func TestSplit_ThresholdTooLow(t *testing.T) {
	_, err := Split([]byte{0x01}, 3, 1)
	if !errors.Is(err, ErrThresholdTooLow) {
		t.Errorf("expected ErrThresholdTooLow, got %v", err)
	}
}

func TestSplit_NLessThanThreshold(t *testing.T) {
	_, err := Split([]byte{0x01}, 1, 2)
	if !errors.Is(err, ErrNotEnoughShares) {
		t.Errorf("expected ErrNotEnoughShares, got %v", err)
	}
}

func TestSplit_TooManyShares(t *testing.T) {
	_, err := Split([]byte{0x01}, 256, 2)
	if !errors.Is(err, ErrTooManyShares) {
		t.Errorf("expected ErrTooManyShares, got %v", err)
	}
}

// --- Input validation: Combine errors ---

func TestCombine_DuplicateIndices(t *testing.T) {
	shares := []Share{
		{Index: 1, Data: []byte{0x01}},
		{Index: 1, Data: []byte{0x02}},
	}
	_, err := Combine(shares)
	if !errors.Is(err, ErrDuplicateIndex) {
		t.Errorf("expected ErrDuplicateIndex, got %v", err)
	}
}

func TestCombine_EmptyShareData(t *testing.T) {
	shares := []Share{
		{Index: 1, Data: nil},
		{Index: 2, Data: []byte{0x01}},
	}
	_, err := Combine(shares)
	if !errors.Is(err, ErrEmptyShareData) {
		t.Errorf("expected ErrEmptyShareData, got %v", err)
	}
}

func TestCombine_MismatchedShareLengths(t *testing.T) {
	shares := []Share{
		{Index: 1, Data: []byte{0x01}},
		{Index: 2, Data: []byte{0x01, 0x02}},
	}
	_, err := Combine(shares)
	if !errors.Is(err, ErrShareDataMismatch) {
		t.Errorf("expected ErrShareDataMismatch, got %v", err)
	}
}

// --- Share structure tests ---

func TestSplit_ShareIndicesAre1Based(t *testing.T) {
	shares, err := Split([]byte{0x42}, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	if shares[0].Index != 1 {
		t.Errorf("shares[0].Index = %d, want 1", shares[0].Index)
	}
	if shares[1].Index != 2 {
		t.Errorf("shares[1].Index = %d, want 2", shares[1].Index)
	}
	if shares[2].Index != 3 {
		t.Errorf("shares[2].Index = %d, want 3", shares[2].Index)
	}
}

func TestSplit_ShareDataLength(t *testing.T) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}

	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	for i, s := range shares {
		if len(s.Data) != len(secret) {
			t.Errorf("shares[%d].Data length = %d, want %d", i, len(s.Data), len(secret))
		}
	}
}
