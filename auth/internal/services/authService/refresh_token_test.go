package authService_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/services/authService/mocks"
)

// refreshSuite holds shared setup for refresh token tests.
type refreshSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *authService.AuthService
	privateKey     *rsa.PrivateKey
}

func newRefreshSuite(t *testing.T) *refreshSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair for test")

	service := authService.NewAuthService(
		storage, sessionStorage,
		privateKey, &privateKey.PublicKey,
		15*time.Minute, 168*time.Hour,
	)
	return &refreshSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
		privateKey:     privateKey,
	}
}

func TestRefreshToken_Success(t *testing.T) {
	s := newRefreshSuite(t)

	// Generate a valid refresh token
	refreshToken, refreshJTI, err := s.service.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)

	// Mock: GetRefreshToken returns existing data
	s.sessionStorage.GetRefreshTokenMock.Expect(minimock.AnyContext, refreshJTI).Return(&domain.RefreshTokenData{
		UserID:      "user-123",
		TokenFamily: "family-abc",
	}, nil)

	// Mock: DeleteRefreshToken for old JTI
	s.sessionStorage.DeleteRefreshTokenMock.Expect(minimock.AnyContext, refreshJTI).Return(nil)

	// Mock: StoreRefreshToken for new token
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		assert.Assert(t, jti != refreshJTI, "new JTI should differ from old JTI")
		assert.Equal(t, userID, "user-123")
		assert.Equal(t, tokenFamily, "family-abc", "token family must be preserved during rotation")
		assert.Equal(t, ttl, 168*time.Hour)
		return nil
	})

	newAccess, newRefresh, err := s.service.RefreshToken(context.Background(), refreshToken)

	assert.NilError(t, err)
	assert.Assert(t, newAccess != "", "new access token should not be empty")
	assert.Assert(t, newRefresh != "", "new refresh token should not be empty")
	assert.Assert(t, newRefresh != refreshToken, "new refresh token should differ from old one")
}

func TestRefreshToken_TheftDetection(t *testing.T) {
	s := newRefreshSuite(t)

	// Generate a valid refresh token
	refreshToken, refreshJTI, err := s.service.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)

	// Mock: GetRefreshToken returns nil -- JTI not in Redis (already rotated/stolen)
	s.sessionStorage.GetRefreshTokenMock.Expect(minimock.AnyContext, refreshJTI).Return(nil, nil)

	// Mock: DeleteTokenFamily should be called for theft detection
	familyDeleted := false
	s.sessionStorage.DeleteTokenFamilyMock.Set(func(_ context.Context, family string) error {
		familyDeleted = true
		assert.Equal(t, family, "family-abc", "should delete the correct token family")
		return nil
	})

	_, _, err = s.service.RefreshToken(context.Background(), refreshToken)

	assert.Assert(t, err != nil, "expected error for theft detection")
	assert.Assert(t, errors.Is(err, domain.ErrTokenRevoked),
		"expected ErrTokenRevoked, got: %v", err)
	assert.Assert(t, familyDeleted, "DeleteTokenFamily should have been called")
}

func TestRefreshToken_InvalidJWT(t *testing.T) {
	s := newRefreshSuite(t)

	_, _, err := s.service.RefreshToken(context.Background(), "invalid-token-string")

	assert.Assert(t, err != nil, "expected error for invalid JWT")
	assert.Assert(t, errors.Is(err, domain.ErrInvalidToken),
		"expected ErrInvalidToken, got: %v", err)
}

func TestRefreshToken_ExpiredJWT(t *testing.T) {
	s := newRefreshSuite(t)

	// Create a service with very short TTL to generate an already-expired token
	shortService := authService.NewAuthService(
		s.storage, s.sessionStorage,
		s.privateKey, &s.privateKey.PublicKey,
		15*time.Minute, 1*time.Nanosecond,
	)

	refreshToken, _, err := shortService.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Millisecond)

	_, _, err = s.service.RefreshToken(context.Background(), refreshToken)

	assert.Assert(t, err != nil, "expected error for expired JWT")
}

func TestRefreshToken_DeletesOldAndStoresNew(t *testing.T) {
	s := newRefreshSuite(t)

	refreshToken, refreshJTI, err := s.service.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)

	s.sessionStorage.GetRefreshTokenMock.Expect(minimock.AnyContext, refreshJTI).Return(&domain.RefreshTokenData{
		UserID:      "user-123",
		TokenFamily: "family-abc",
	}, nil)

	deleteOldCalled := false
	s.sessionStorage.DeleteRefreshTokenMock.Set(func(_ context.Context, jti string) error {
		deleteOldCalled = true
		assert.Equal(t, jti, refreshJTI, "should delete the old JTI")
		return nil
	})

	storeNewCalled := false
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		storeNewCalled = true
		assert.Assert(t, jti != refreshJTI, "new JTI should differ from old")
		return nil
	})

	_, _, err = s.service.RefreshToken(context.Background(), refreshToken)
	assert.NilError(t, err)
	assert.Assert(t, deleteOldCalled, "DeleteRefreshToken should have been called for old JTI")
	assert.Assert(t, storeNewCalled, "StoreRefreshToken should have been called for new token")
}
