package totp

import (
	"encoding/base32"
	"testing"
)

// TestGenerateOTP_RFC6238 verifies all 6 SHA-1 test vectors from RFC 6238 Appendix B.
// Secret is the ASCII string "12345678901234567890" (20 bytes).
// The 8-digit RFC codes mod 1_000_000 yield the expected 6-digit codes.
func TestGenerateOTP_RFC6238(t *testing.T) {
	secret := []byte("12345678901234567890")
	vectors := []struct {
		name   string
		time   int64
		expect string
	}{
		{"t=59", 59, "287082"},
		{"t=1111111109", 1111111109, "081804"},
		{"t=1111111111", 1111111111, "050471"},
		{"t=1234567890", 1234567890, "005924"},
		{"t=2000000000", 2000000000, "279037"},
		{"t=20000000000", 20000000000, "353130"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			got := GenerateOTP(secret, v.time)
			if got != v.expect {
				t.Errorf("GenerateOTP(secret, %d) = %q, want %q", v.time, got, v.expect)
			}
		})
	}
}

// TestGenerateOTP_ZeroPadding verifies that leading zeros are preserved in 6-digit codes.
// time=1234567890 should produce "005924" (not "5924").
func TestGenerateOTP_ZeroPadding(t *testing.T) {
	secret := []byte("12345678901234567890")
	got := GenerateOTP(secret, 1234567890)
	if got != "005924" {
		t.Errorf("GenerateOTP zero-padding: got %q, want %q", got, "005924")
	}
	if len(got) != 6 {
		t.Errorf("GenerateOTP code length = %d, want 6", len(got))
	}
}

// TestGenerateSecret verifies that GenerateSecret produces a 20-byte secret
// with a valid base32-encoded string (no padding), and that decoding the
// base32 string yields the original bytes.
func TestGenerateSecret(t *testing.T) {
	raw, encoded, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	// Raw secret must be 20 bytes.
	if len(raw) != 20 {
		t.Errorf("raw secret length = %d, want 20", len(raw))
	}

	// Base32 encoding of 20 bytes = 32 characters (no padding).
	if len(encoded) != 32 {
		t.Errorf("encoded length = %d, want 32", len(encoded))
	}

	// No padding characters.
	for _, c := range encoded {
		if c == '=' {
			t.Error("encoded string contains padding character '='")
			break
		}
	}

	// Decoding must yield the original bytes.
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		t.Fatalf("base32 decode error: %v", err)
	}
	if len(decoded) != len(raw) {
		t.Errorf("decoded length = %d, want %d", len(decoded), len(raw))
	}
	for i := range raw {
		if i < len(decoded) && raw[i] != decoded[i] {
			t.Errorf("decoded[%d] = %d, want %d", i, decoded[i], raw[i])
		}
	}
}

// TestGenerateSecret_Uniqueness verifies that two calls produce different secrets.
func TestGenerateSecret_Uniqueness(t *testing.T) {
	raw1, _, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret 1: %v", err)
	}
	raw2, _, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret 2: %v", err)
	}

	allSame := true
	for i := range raw1 {
		if i < len(raw2) && raw1[i] != raw2[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("two GenerateSecret calls produced identical secrets")
	}
}

// TestValidateOTP_CurrentWindow verifies that validateOTPAt accepts a code
// generated for the exact same time counter.
func TestValidateOTP_CurrentWindow(t *testing.T) {
	secret := []byte("12345678901234567890")
	// time=1234567890, counter=41152263, expected code="005924"
	unixTime := int64(1234567890)
	code := "005924"

	if !validateOTPAt(secret, code, unixTime) {
		t.Errorf("validateOTPAt should accept code %q at exact time %d", code, unixTime)
	}
}

// TestValidateOTP_AdjacentWindows verifies that validateOTPAt accepts codes
// from T-1 and T+1 time windows.
func TestValidateOTP_AdjacentWindows(t *testing.T) {
	secret := []byte("12345678901234567890")
	// Use time=1234567890 (counter=41152263).
	// Generate codes for counter-1 and counter+1, then validate at original time.
	baseTime := int64(1234567890)
	baseCounter := uint64(baseTime) / 30

	// Code for previous window (counter-1).
	codePrev := hotp(secret, baseCounter-1)
	if !validateOTPAt(secret, codePrev, baseTime) {
		t.Errorf("validateOTPAt should accept code %q from T-1 window", codePrev)
	}

	// Code for next window (counter+1).
	codeNext := hotp(secret, baseCounter+1)
	if !validateOTPAt(secret, codeNext, baseTime) {
		t.Errorf("validateOTPAt should accept code %q from T+1 window", codeNext)
	}
}

// TestValidateOTP_OutsideWindow verifies that validateOTPAt rejects codes
// from T-2 and T+2 time windows.
func TestValidateOTP_OutsideWindow(t *testing.T) {
	secret := []byte("12345678901234567890")
	baseTime := int64(1234567890)
	baseCounter := uint64(baseTime) / 30

	// Code for T-2 window.
	codeFar := hotp(secret, baseCounter-2)
	if validateOTPAt(secret, codeFar, baseTime) {
		t.Errorf("validateOTPAt should reject code %q from T-2 window", codeFar)
	}

	// Code for T+2 window.
	codeFarNext := hotp(secret, baseCounter+2)
	if validateOTPAt(secret, codeFarNext, baseTime) {
		t.Errorf("validateOTPAt should reject code %q from T+2 window", codeFarNext)
	}
}

// TestValidateOTP_WrongCode verifies that validateOTPAt rejects an incorrect code.
func TestValidateOTP_WrongCode(t *testing.T) {
	secret := []byte("12345678901234567890")
	if validateOTPAt(secret, "000000", 1234567890) {
		t.Error("validateOTPAt should reject wrong code '000000'")
	}
}

// TestValidateOTP_EmptyCode verifies that validateOTPAt rejects an empty string.
func TestValidateOTP_EmptyCode(t *testing.T) {
	secret := []byte("12345678901234567890")
	if validateOTPAt(secret, "", 1234567890) {
		t.Error("validateOTPAt should reject empty code")
	}
}

// TestValidateOTP_ShortCode verifies that validateOTPAt rejects a 5-digit code.
func TestValidateOTP_ShortCode(t *testing.T) {
	secret := []byte("12345678901234567890")
	if validateOTPAt(secret, "12345", 1234567890) {
		t.Error("validateOTPAt should reject 5-digit code")
	}
}

// TestValidateOTP_LongCode verifies that validateOTPAt rejects a 7-digit code.
func TestValidateOTP_LongCode(t *testing.T) {
	secret := []byte("12345678901234567890")
	if validateOTPAt(secret, "1234567", 1234567890) {
		t.Error("validateOTPAt should reject 7-digit code")
	}
}

// TestValidateOTP_NonNumeric verifies that validateOTPAt rejects a code with
// non-numeric characters.
func TestValidateOTP_NonNumeric(t *testing.T) {
	secret := []byte("12345678901234567890")
	if validateOTPAt(secret, "12345a", 1234567890) {
		t.Error("validateOTPAt should reject non-numeric code")
	}
}

// TestValidateOTP_BoundaryTransition verifies that codes at the boundary of
// two time periods are different. t=29 (end of period 0) and t=30 (start of
// period 1) should produce different codes.
func TestValidateOTP_BoundaryTransition(t *testing.T) {
	secret := []byte("12345678901234567890")
	code29 := GenerateOTP(secret, 29)
	code30 := GenerateOTP(secret, 30)
	if code29 == code30 {
		t.Errorf("codes at boundary should differ: t=29 %q, t=30 %q", code29, code30)
	}
}
