package shamir

import "testing"

// TestGF256_ExpLogTableConsistency verifies expTable[logTable[x]] == x for all x in 1..255.
func TestGF256_ExpLogTableConsistency(t *testing.T) {
	for x := 1; x <= 255; x++ {
		got := expTable[logTable[byte(x)]]
		if got != byte(x) {
			t.Fatalf("expTable[logTable[%d]] = %d, want %d", x, got, x)
		}
	}
}

// TestGF256_AddCommutativity verifies gfAdd(a, b) == gfAdd(b, a) for all 256x256 pairs.
func TestGF256_AddCommutativity(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			ab := gfAdd(byte(a), byte(b))
			ba := gfAdd(byte(b), byte(a))
			if ab != ba {
				t.Fatalf("gfAdd(%d, %d) = %d != gfAdd(%d, %d) = %d", a, b, ab, b, a, ba)
			}
		}
	}
}

// TestGF256_AddAssociativity verifies gfAdd(gfAdd(a, b), c) == gfAdd(a, gfAdd(b, c)) for sampled triples.
func TestGF256_AddAssociativity(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			for c := 0; c < 256; c++ {
				lhs := gfAdd(gfAdd(byte(a), byte(b)), byte(c))
				rhs := gfAdd(byte(a), gfAdd(byte(b), byte(c)))
				if lhs != rhs {
					t.Fatalf("associativity failed for (%d, %d, %d): %d != %d", a, b, c, lhs, rhs)
				}
			}
		}
	}
}

// TestGF256_AddIdentity verifies gfAdd(a, 0) == a for all 256 elements.
func TestGF256_AddIdentity(t *testing.T) {
	for a := 0; a < 256; a++ {
		got := gfAdd(byte(a), 0)
		if got != byte(a) {
			t.Fatalf("gfAdd(%d, 0) = %d, want %d", a, got, a)
		}
	}
}

// TestGF256_AddInverse verifies gfAdd(a, a) == 0 for all 256 elements (self-inverse in GF(2^n)).
func TestGF256_AddInverse(t *testing.T) {
	for a := 0; a < 256; a++ {
		got := gfAdd(byte(a), byte(a))
		if got != 0 {
			t.Fatalf("gfAdd(%d, %d) = %d, want 0", a, a, got)
		}
	}
}

// TestGF256_MulCommutative verifies gfMul(a, b) == gfMul(b, a) for all 256x256 pairs.
func TestGF256_MulCommutative(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			ab := gfMul(byte(a), byte(b))
			ba := gfMul(byte(b), byte(a))
			if ab != ba {
				t.Fatalf("gfMul(%d, %d) = %d != gfMul(%d, %d) = %d", a, b, ab, b, a, ba)
			}
		}
	}
}

// TestGF256_MulAssociative verifies gfMul(gfMul(a, b), c) == gfMul(a, gfMul(b, c)) for sampled triples.
func TestGF256_MulAssociative(t *testing.T) {
	// Sample a subset to keep test runtime reasonable while still covering edge cases.
	samples := []byte{0, 1, 2, 3, 127, 128, 254, 255}
	for _, a := range samples {
		for _, b := range samples {
			for _, c := range samples {
				lhs := gfMul(gfMul(a, b), c)
				rhs := gfMul(a, gfMul(b, c))
				if lhs != rhs {
					t.Fatalf("associativity failed for (%d, %d, %d): %d != %d", a, b, c, lhs, rhs)
				}
			}
		}
	}
}

// TestGF256_MulIdentity verifies gfMul(a, 1) == a for all 256 elements.
func TestGF256_MulIdentity(t *testing.T) {
	for a := 0; a < 256; a++ {
		got := gfMul(byte(a), 1)
		if got != byte(a) {
			t.Fatalf("gfMul(%d, 1) = %d, want %d", a, got, a)
		}
	}
}

// TestGF256_MulZero verifies gfMul(a, 0) == 0 and gfMul(0, b) == 0 for all 256 elements.
func TestGF256_MulZero(t *testing.T) {
	for a := 0; a < 256; a++ {
		if got := gfMul(byte(a), 0); got != 0 {
			t.Fatalf("gfMul(%d, 0) = %d, want 0", a, got)
		}
		if got := gfMul(0, byte(a)); got != 0 {
			t.Fatalf("gfMul(0, %d) = %d, want 0", a, got)
		}
	}
}

// TestGF256_MulInverse verifies gfMul(a, gfInv(a)) == 1 for all 255 non-zero elements.
func TestGF256_MulInverse(t *testing.T) {
	for a := 1; a <= 255; a++ {
		inv := gfInv(byte(a))
		got := gfMul(byte(a), inv)
		if got != 1 {
			t.Fatalf("gfMul(%d, gfInv(%d)) = %d, want 1", a, a, got)
		}
	}
}

// TestGF256_Distributive verifies gfMul(a, gfAdd(b, c)) == gfAdd(gfMul(a, b), gfMul(a, c)) for all 16M+ triples.
func TestGF256_Distributive(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			for c := 0; c < 256; c++ {
				lhs := gfMul(byte(a), gfAdd(byte(b), byte(c)))
				rhs := gfAdd(gfMul(byte(a), byte(b)), gfMul(byte(a), byte(c)))
				if lhs != rhs {
					t.Fatalf("distributivity failed for (%d, %d, %d): %d != %d", a, b, c, lhs, rhs)
				}
			}
		}
	}
}

// TestGF256_DivInverse verifies gfDiv(gfMul(a, b), b) == a for all non-zero b, all a.
func TestGF256_DivInverse(t *testing.T) {
	for b := 1; b <= 255; b++ {
		for a := 0; a < 256; a++ {
			product := gfMul(byte(a), byte(b))
			got := gfDiv(product, byte(b))
			if got != byte(a) {
				t.Fatalf("gfDiv(gfMul(%d, %d), %d) = %d, want %d", a, b, b, got, a)
			}
		}
	}
}

// TestGF256_DivByZeroPanics verifies gfDiv(x, 0) panics for any x.
func TestGF256_DivByZeroPanics(t *testing.T) {
	for x := 0; x < 256; x++ {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("gfDiv(%d, 0) did not panic", x)
				}
			}()
			gfDiv(byte(x), 0)
		}()
	}
}

// TestGF256_InvZeroPanics verifies gfInv(0) panics.
func TestGF256_InvZeroPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("gfInv(0) did not panic")
		}
	}()
	gfInv(0)
}
