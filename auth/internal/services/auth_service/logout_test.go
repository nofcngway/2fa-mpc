package auth_service_test

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
	"github.com/vbncursed/vkr/auth/internal/services/auth_service"
	"github.com/vbncursed/vkr/auth/internal/services/auth_service/mocks"
)

// logoutSuite holds shared setup for logout tests.
type logoutSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *auth_service.AuthService
	privateKey     *rsa.PrivateKey
}

func newLogoutSuite(t *testing.T) *logoutSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)
	eventProducer := mocks.NewEventPublisherMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair for test")

	service, err := auth_service.NewAuthService(auth_service.Deps{
		Storage:         storage,
		SessionStorage:  sessionStorage,
		EventPublisher:   eventProducer,
		PrivateKey:      privateKey,
		PublicKey:        &privateKey.PublicKey,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})
	assert.NilError(t, err, "failed to create auth service")
	return &logoutSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
		privateKey:     privateKey,
	}
}

func TestLogout_Success(t *testing.T) {
	s := newLogoutSuite(t)

	// Generate a valid refresh token
	refreshToken, refreshJTI, err := s.service.GenerateRefreshToken("user-123", "test@example.com", "family-abc")
	assert.NilError(t, err)

	// Mock: DeleteRefreshToken called with correct JTI
	deleteCalled := false
	s.sessionStorage.DeleteRefreshTokenMock.Set(func(_ context.Context, jti string) error {
		deleteCalled = true
		assert.Equal(t, jti, refreshJTI, "should delete the correct JTI")
		return nil
	})

	err = s.service.Logout(t.Context(), refreshToken)

	assert.NilError(t, err)
	assert.Assert(t, deleteCalled, "DeleteRefreshToken should have been called")
}

func TestLogout_InvalidToken(t *testing.T) {
	s := newLogoutSuite(t)

	err := s.service.Logout(t.Context(), "not-a-valid-token")

	assert.Assert(t, err != nil, "expected error for invalid token")
	assert.Assert(t, errors.Is(err, domain.ErrInvalidToken),
		"expected ErrInvalidToken, got: %v", err)
}
