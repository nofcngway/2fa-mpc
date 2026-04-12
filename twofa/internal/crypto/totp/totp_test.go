package totp

import (
	"encoding/base32"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// totpSuite holds shared test fixtures for TOTP tests.
type totpSuite struct {
	secret    []byte
	baseTime  int64
	baseCode  string
	baseCount uint64
}

func newTOTPSuite() *totpSuite {
	secret := []byte("12345678901234567890")
	baseTime := int64(1234567890)
	return &totpSuite{
		secret:    secret,
		baseTime:  baseTime,
		baseCode:  "005924",
		baseCount: uint64(baseTime) / 30,
	}
}

// --- GenerateOTP Tests ---

// TestGenerateOTP_RFC6238 verifies all 6 SHA-1 test vectors from RFC 6238 Appendix B.
func TestGenerateOTP_RFC6238(t *testing.T) {
	s := newTOTPSuite()
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
			got := GenerateOTP(s.secret, v.time)
			assert.Equal(t, got, v.expect, "GenerateOTP(secret, %d)", v.time)
		})
	}
}

// TestGenerateOTP_ZeroPadding verifies that leading zeros are preserved in 6-digit codes.
func TestGenerateOTP_ZeroPadding(t *testing.T) {
	s := newTOTPSuite()
	got := GenerateOTP(s.secret, 1234567890)
	assert.Equal(t, got, "005924")
	assert.Equal(t, len(got), 6)
}

// --- GenerateSecret Tests ---

// TestGenerateSecret verifies that GenerateSecret produces a 20-byte secret
// with a valid base32-encoded string (no padding), and that decoding the
// base32 string yields the original bytes.
func TestGenerateSecret(t *testing.T) {
	raw, encoded, err := GenerateSecret()
	assert.NilError(t, err)

	// Raw secret must be 20 bytes.
	assert.Equal(t, len(raw), 20)

	// Base32 encoding of 20 bytes = 32 characters (no padding).
	assert.Equal(t, len(encoded), 32)

	// No padding characters.
	assert.Assert(t, !strings.Contains(encoded, "="), "encoded string contains padding character '='")

	// Decoding must yield the original bytes.
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	assert.NilError(t, err)
	assert.DeepEqual(t, raw, decoded)
}

// TestGenerateSecret_Uniqueness verifies that two calls produce different secrets.
func TestGenerateSecret_Uniqueness(t *testing.T) {
	raw1, _, err := GenerateSecret()
	assert.NilError(t, err)
	raw2, _, err := GenerateSecret()
	assert.NilError(t, err)

	allSame := true
	for i := range raw1 {
		if i < len(raw2) && raw1[i] != raw2[i] {
			allSame = false
			break
		}
	}
	assert.Assert(t, !allSame, "two GenerateSecret calls produced identical secrets")
}

// --- ValidateOTP Tests ---

// TestValidateOTP_CurrentWindow verifies that validateOTPAt accepts a code
// generated for the exact same time counter.
func TestValidateOTP_CurrentWindow(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, validateOTPAt(s.secret, s.baseCode, s.baseTime),
		"should accept code %q at exact time %d", s.baseCode, s.baseTime)
}

// TestValidateOTP_AdjacentWindows verifies that validateOTPAt accepts codes
// from T-1 and T+1 time windows.
func TestValidateOTP_AdjacentWindows(t *testing.T) {
	s := newTOTPSuite()

	codePrev := hotp(s.secret, s.baseCount-1)
	assert.Assert(t, validateOTPAt(s.secret, codePrev, s.baseTime),
		"should accept code %q from T-1 window", codePrev)

	codeNext := hotp(s.secret, s.baseCount+1)
	assert.Assert(t, validateOTPAt(s.secret, codeNext, s.baseTime),
		"should accept code %q from T+1 window", codeNext)
}

// TestValidateOTP_OutsideWindow verifies that validateOTPAt rejects codes
// from T-2 and T+2 time windows.
func TestValidateOTP_OutsideWindow(t *testing.T) {
	s := newTOTPSuite()

	codeFar := hotp(s.secret, s.baseCount-2)
	assert.Assert(t, !validateOTPAt(s.secret, codeFar, s.baseTime),
		"should reject code %q from T-2 window", codeFar)

	codeFarNext := hotp(s.secret, s.baseCount+2)
	assert.Assert(t, !validateOTPAt(s.secret, codeFarNext, s.baseTime),
		"should reject code %q from T+2 window", codeFarNext)
}

// TestValidateOTP_WrongCode verifies that validateOTPAt rejects an incorrect code.
func TestValidateOTP_WrongCode(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, !validateOTPAt(s.secret, "000000", s.baseTime))
}

// TestValidateOTP_EmptyCode verifies that validateOTPAt rejects an empty string.
func TestValidateOTP_EmptyCode(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, !validateOTPAt(s.secret, "", s.baseTime))
}

// TestValidateOTP_ShortCode verifies that validateOTPAt rejects a 5-digit code.
func TestValidateOTP_ShortCode(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, !validateOTPAt(s.secret, "12345", s.baseTime))
}

// TestValidateOTP_LongCode verifies that validateOTPAt rejects a 7-digit code.
func TestValidateOTP_LongCode(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, !validateOTPAt(s.secret, "1234567", s.baseTime))
}

// TestValidateOTP_NonNumeric verifies that validateOTPAt rejects a code with
// non-numeric characters.
func TestValidateOTP_NonNumeric(t *testing.T) {
	s := newTOTPSuite()
	assert.Assert(t, !validateOTPAt(s.secret, "12345a", s.baseTime))
}

// TestValidateOTP_BoundaryTransition verifies that codes at the boundary of
// two time periods are different.
func TestValidateOTP_BoundaryTransition(t *testing.T) {
	s := newTOTPSuite()
	code29 := GenerateOTP(s.secret, 29)
	code30 := GenerateOTP(s.secret, 30)
	assert.Assert(t, code29 != code30,
		"codes at boundary should differ: t=29 %q, t=30 %q", code29, code30)
}

// TestValidateOTP_CounterZero verifies no underflow when counter is 0.
func TestValidateOTP_CounterZero(t *testing.T) {
	s := newTOTPSuite()
	// time=15 gives counter=0, should not panic or produce wrong results.
	code := GenerateOTP(s.secret, 15)
	assert.Assert(t, validateOTPAt(s.secret, code, 15))
}

// TestHOTP_EmptySecretPanics verifies that hotp panics on empty secret.
func TestHOTP_EmptySecretPanics(t *testing.T) {
	defer func() {
		r := recover()
		assert.Assert(t, r != nil, "hotp with empty secret should panic")
	}()
	hotp([]byte{}, 0)
}

// --- URI Tests ---

// --- ValidateOTPWithCounter Tests ---

// TestValidateOTPWithCounter_CurrentWindow verifies that validateOTPWithCounterAt
// returns (true, counter) for a code generated at the exact time counter.
func TestValidateOTPWithCounter_CurrentWindow(t *testing.T) {
	s := newTOTPSuite()
	code := hotp(s.secret, s.baseCount)
	valid, counter := validateOTPWithCounterAt(s.secret, code, s.baseTime)
	assert.Assert(t, valid, "should accept code %q at exact time %d", code, s.baseTime)
	assert.Equal(t, counter, int64(s.baseCount))
}

// TestValidateOTPWithCounter_PrevWindow verifies that validateOTPWithCounterAt
// returns (true, counter-1) for a code generated at T-1 window.
func TestValidateOTPWithCounter_PrevWindow(t *testing.T) {
	s := newTOTPSuite()
	prevCounter := s.baseCount - 1
	code := hotp(s.secret, prevCounter)
	valid, counter := validateOTPWithCounterAt(s.secret, code, s.baseTime)
	assert.Assert(t, valid, "should accept code %q from T-1 window", code)
	assert.Equal(t, counter, int64(prevCounter))
}

// TestValidateOTPWithCounter_NextWindow verifies that validateOTPWithCounterAt
// returns (true, counter+1) for a code generated at T+1 window.
func TestValidateOTPWithCounter_NextWindow(t *testing.T) {
	s := newTOTPSuite()
	nextCounter := s.baseCount + 1
	code := hotp(s.secret, nextCounter)
	valid, counter := validateOTPWithCounterAt(s.secret, code, s.baseTime)
	assert.Assert(t, valid, "should accept code %q from T+1 window", code)
	assert.Equal(t, counter, int64(nextCounter))
}

// TestValidateOTPWithCounter_WrongCode verifies that validateOTPWithCounterAt
// returns (false, 0) for an incorrect code.
func TestValidateOTPWithCounter_WrongCode(t *testing.T) {
	s := newTOTPSuite()
	valid, counter := validateOTPWithCounterAt(s.secret, "000000", s.baseTime)
	assert.Assert(t, !valid, "should reject wrong code")
	assert.Equal(t, counter, int64(0))
}

// TestValidateOTPWithCounter_NonSixDigit verifies that validateOTPWithCounterAt
// returns (false, 0) for non-6-digit input.
func TestValidateOTPWithCounter_NonSixDigit(t *testing.T) {
	s := newTOTPSuite()

	valid, counter := validateOTPWithCounterAt(s.secret, "12345", s.baseTime)
	assert.Assert(t, !valid, "should reject 5-digit code")
	assert.Equal(t, counter, int64(0))

	valid, counter = validateOTPWithCounterAt(s.secret, "1234567", s.baseTime)
	assert.Assert(t, !valid, "should reject 7-digit code")
	assert.Equal(t, counter, int64(0))

	valid, counter = validateOTPWithCounterAt(s.secret, "12345a", s.baseTime)
	assert.Assert(t, !valid, "should reject non-numeric code")
	assert.Equal(t, counter, int64(0))
}

// TestGenerateProvisioningURI_BasicFormat verifies the URI structure with a standard email.
func TestGenerateProvisioningURI_BasicFormat(t *testing.T) {
	uri := GenerateProvisioningURI("JBSWY3DPEHPK3PXP", "user@example.com")

	assert.Assert(t, strings.HasPrefix(uri, "otpauth://totp/"), "URI prefix")
	assert.Assert(t, strings.Contains(uri, "MPC-2FA:"), "issuer in label")
	// @ may be encoded as %40 by url.PathEscape.
	assert.Assert(t,
		strings.Contains(uri, "user%40example.com") || strings.Contains(uri, "user@example.com"),
		"email in URI: %s", uri)
	assert.Assert(t, strings.Contains(uri, "secret=JBSWY3DPEHPK3PXP"), "secret param")
	assert.Assert(t, strings.Contains(uri, "issuer=MPC-2FA"), "issuer param")
	assert.Assert(t, strings.Contains(uri, "algorithm=SHA1"), "algorithm param")
	assert.Assert(t, strings.Contains(uri, "digits=6"), "digits param")
	assert.Assert(t, strings.Contains(uri, "period=30"), "period param")
}

// TestGenerateProvisioningURI_SpecialCharsEmail verifies URL encoding of special characters in email.
func TestGenerateProvisioningURI_SpecialCharsEmail(t *testing.T) {
	uri := GenerateProvisioningURI("JBSWY3DPEHPK3PXP", "user+tag@exam ple.com")

	assert.Assert(t, strings.HasPrefix(uri, "otpauth://totp/"), "URI prefix")
	assert.Assert(t, strings.Contains(uri, "%20"), "space encoded as %%20")
	assert.Assert(t,
		strings.Contains(uri, "%2B") || strings.Contains(uri, "+"),
		"plus (raw or encoded) in URI: %s", uri)
}

// TestGenerateProvisioningURI_FullRoundtrip generates a secret and builds a URI,
// verifying the generated base32 secret appears in the output.
func TestGenerateProvisioningURI_FullRoundtrip(t *testing.T) {
	_, encoded, err := GenerateSecret()
	assert.NilError(t, err)

	uri := GenerateProvisioningURI(encoded, "roundtrip@test.com")

	assert.Assert(t, strings.Contains(uri, "secret="+encoded), "generated secret in URI")
	assert.Assert(t, strings.HasPrefix(uri, "otpauth://totp/MPC-2FA:"), "URI prefix with issuer")
}

// TestGenerateProvisioningURI_EmptyEmail verifies that an empty email produces a valid URI.
func TestGenerateProvisioningURI_EmptyEmail(t *testing.T) {
	uri := GenerateProvisioningURI("JBSWY3DPEHPK3PXP", "")

	assert.Assert(t, strings.HasPrefix(uri, "otpauth://totp/MPC-2FA:"), "URI prefix")
	assert.Assert(t, strings.Contains(uri, "secret=JBSWY3DPEHPK3PXP"), "secret param")
}
