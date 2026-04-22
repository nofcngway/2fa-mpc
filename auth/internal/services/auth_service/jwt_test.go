package auth_service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/golang-jwt/jwt/v5"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/auth/internal/producer"
	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/services/auth_service"
	"github.com/vbncursed/vkr/auth/internal/services/auth_service/mocks"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair")
	return privateKey, &privateKey.PublicKey
}

func newJWTTestService(t *testing.T) *auth_service.AuthService {
	t.Helper()
	mc := minimock.NewController(t)
	privateKey, publicKey := generateTestKeyPair(t)
	svc, err := auth_service.NewAuthService(auth_service.Deps{
		Storage:         mocks.NewStorageMock(mc),
		SessionStorage:  mocks.NewSessionStorageMock(mc),
		EventPublisher:   &producer.NoOpProducer{},
		PrivateKey:      privateKey,
		PublicKey:        publicKey,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})
	assert.NilError(t, err, "failed to create auth service")
	return svc
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

	// Build HS256 token manually
	claims := jwt.MapClaims{
		"sub":   "attacker",
		"email": "evil@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iss":   "mpc-2fa-auth",
	}
	hs256Token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

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
	mc := minimock.NewController(t)
	svc, err := auth_service.NewAuthService(auth_service.Deps{
		Storage:         mocks.NewStorageMock(mc),
		SessionStorage:  mocks.NewSessionStorageMock(mc),
		EventPublisher:   &producer.NoOpProducer{},
		PrivateKey:      privateKey,
		PublicKey:        publicKey,
		AccessTokenTTL:  1 * time.Millisecond,
		RefreshTokenTTL: 168 * time.Hour,
	})
	assert.NilError(t, err)

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
