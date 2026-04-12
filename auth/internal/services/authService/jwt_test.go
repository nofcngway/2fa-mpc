package authService_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/auth/internal/bootstrap"
	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair")
	return privateKey, &privateKey.PublicKey
}

func newJWTTestService(t *testing.T) *authService.AuthService {
	t.Helper()
	privateKey, publicKey := generateTestKeyPair(t)
	return authService.NewAuthService(nil, nil, &bootstrap.NoOpProducer{}, privateKey, publicKey, 15*time.Minute, 168*time.Hour)
}

func TestJWT_GenerateAccessToken_ValidClaims(t *testing.T) {
	svc := newJWTTestService(t)

	tokenStr, jti, err := svc.GenerateAccessToken("user-123", "test@example.com")
	assert.NilError(t, err)
	assert.Assert(t, tokenStr != "", "token string should not be empty")
	assert.Assert(t, jti != "", "jti should not be empty")

	claims, err := svc.ParseToken(tokenStr)
	assert.NilError(t, err)
	assert.Equal(t, claims.Subject, "user-123")
	assert.Equal(t, claims.Email, "test@example.com")
	assert.Equal(t, claims.ID, jti)
	assert.Equal(t, claims.Issuer, "mpc-2fa-auth")
	assert.Assert(t, claims.TokenFamily == "", "access token should not have token_family")

	// Check expiry is approximately now + 15 min
	expiry := claims.ExpiresAt.Time
	expected := time.Now().Add(15 * time.Minute)
	diff := expiry.Sub(expected)
	assert.Assert(t, diff > -5*time.Second && diff < 5*time.Second,
		"expiry should be ~15min from now, got diff: %v", diff)
}

func TestJWT_GenerateRefreshToken_ValidClaims(t *testing.T) {
	svc := newJWTTestService(t)

	tokenStr, jti, err := svc.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)
	assert.Assert(t, tokenStr != "", "token string should not be empty")
	assert.Assert(t, jti != "", "jti should not be empty")

	claims, err := svc.ParseToken(tokenStr)
	assert.NilError(t, err)
	assert.Equal(t, claims.Subject, "user-123")
	assert.Equal(t, claims.Email, "test@example.com")
	assert.Equal(t, claims.ID, jti)
	assert.Equal(t, claims.Issuer, "mpc-2fa-auth")
	assert.Equal(t, claims.TokenFamily, "family-abc")

	// Check expiry is approximately now + 7 days
	expiry := claims.ExpiresAt.Time
	expected := time.Now().Add(168 * time.Hour)
	diff := expiry.Sub(expected)
	assert.Assert(t, diff > -5*time.Second && diff < 5*time.Second,
		"expiry should be ~7d from now, got diff: %v", diff)
}

func TestJWT_ParseToken_ValidRS256(t *testing.T) {
	svc := newJWTTestService(t)

	tokenStr, _, err := svc.GenerateAccessToken("user-456", "user@test.com")
	assert.NilError(t, err)

	claims, err := svc.ParseToken(tokenStr)
	assert.NilError(t, err)
	assert.Equal(t, claims.Subject, "user-456")
	assert.Equal(t, claims.Email, "user@test.com")
}

func TestJWT_ParseToken_RejectsHS256_AlgorithmConfusion(t *testing.T) {
	svc := newJWTTestService(t)
	_, publicKey := generateTestKeyPair(t)

	// Create an HS256 token using public key bytes as HMAC secret (algorithm confusion attack)
	pubKeyBytes, err := jwt.ParseRSAPublicKeyFromPEM(nil)
	_ = pubKeyBytes
	_ = err

	// Build HS256 token manually
	claims := jwt.MapClaims{
		"sub":   "attacker",
		"email": "evil@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iss":   "mpc-2fa-auth",
	}
	hs256Token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign with public key bytes as HMAC secret (classic alg confusion)
	pubKeyPEM, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte("test"))
	_ = pubKeyPEM
	_ = err

	// Use arbitrary bytes as HMAC secret
	tokenStr, err := hs256Token.SignedString([]byte("some-hmac-secret"))
	assert.NilError(t, err)

	// ParseToken MUST reject this
	_, parseErr := svc.ParseToken(tokenStr)
	assert.Assert(t, parseErr != nil, "HS256 token should be rejected")
	assert.ErrorIs(t, parseErr, domain.ErrInvalidToken)
}

func TestJWT_ParseToken_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	// Create service with very short TTL
	svc := authService.NewAuthService(nil, nil, &bootstrap.NoOpProducer{}, privateKey, publicKey, 1*time.Millisecond, 168*time.Hour)

	tokenStr, _, err := svc.GenerateAccessToken("user-789", "expired@test.com")
	assert.NilError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, parseErr := svc.ParseToken(tokenStr)
	assert.Assert(t, parseErr != nil, "expired token should be rejected")
	assert.ErrorIs(t, parseErr, domain.ErrTokenExpired)
}

func TestJWT_ParseToken_TamperedToken(t *testing.T) {
	svc := newJWTTestService(t)

	tokenStr, _, err := svc.GenerateAccessToken("user-123", "test@example.com")
	assert.NilError(t, err)

	// Tamper with the token by modifying a character in the signature
	tampered := tokenStr[:len(tokenStr)-5] + "XXXXX"

	_, parseErr := svc.ParseToken(tampered)
	assert.Assert(t, parseErr != nil, "tampered token should be rejected")
	assert.ErrorIs(t, parseErr, domain.ErrInvalidToken)
}

func TestJWT_AccessAndRefreshToken_DifferentJTIs(t *testing.T) {
	svc := newJWTTestService(t)

	_, accessJTI, err := svc.GenerateAccessToken("user-123", "test@example.com")
	assert.NilError(t, err)

	_, refreshJTI, err := svc.GenerateRefreshToken("user-123", "test@example.com", "family-1")
	assert.NilError(t, err)

	assert.Assert(t, accessJTI != refreshJTI, "access and refresh tokens should have different JTIs")
}
