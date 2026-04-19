package authService_test

import (
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

// validateSuite holds shared setup for validate token tests.
type validateSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *authService.AuthService
	privateKey     *rsa.PrivateKey
}

func newValidateSuite(t *testing.T) *validateSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)
	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair for test")

	service, err := authService.NewAuthService(authService.Deps{
		Storage:         storage,
		SessionStorage:  sessionStorage,
		EventProducer:   eventProducer,
		PrivateKey:      privateKey,
		PublicKey:        &privateKey.PublicKey,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})
	assert.NilError(t, err, "failed to create auth service")
	return &validateSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
		privateKey:     privateKey,
	}
}

func TestValidateToken_Success(t *testing.T) {
	s := newValidateSuite(t)

	accessToken, _, err := s.service.GenerateAccessToken("user-123", "test@example.com")
	assert.NilError(t, err)

	userID, email, err := s.service.ValidateToken(t.Context(), accessToken)

	assert.NilError(t, err)
	assert.Equal(t, userID, "user-123")
	assert.Equal(t, email, "test@example.com")
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	s := newValidateSuite(t)

	// Create service with very short access TTL
	shortEventProducer := mocks.NewEventProducerMock(s.mc)
	shortEventProducer.PublishEventMock.Optional().Return(nil)
	shortEventProducer.CloseMock.Optional().Return(nil)
	shortService, err := authService.NewAuthService(authService.Deps{
		Storage:         s.storage,
		SessionStorage:  s.sessionStorage,
		EventProducer:   shortEventProducer,
		PrivateKey:      s.privateKey,
		PublicKey:        &s.privateKey.PublicKey,
		AccessTokenTTL:  1 * time.Nanosecond,
		RefreshTokenTTL: 168 * time.Hour,
	})
	assert.NilError(t, err, "failed to create auth service")

	accessToken, _, err := shortService.GenerateAccessToken("user-123", "test@example.com")
	assert.NilError(t, err)

	time.Sleep(2 * time.Millisecond)

	_, _, err = s.service.ValidateToken(t.Context(), accessToken)

	assert.Assert(t, err != nil, "expected error for expired token")
	assert.Assert(t, errors.Is(err, domain.ErrTokenExpired),
		"expected ErrTokenExpired, got: %v", err)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	s := newValidateSuite(t)

	_, _, err := s.service.ValidateToken(t.Context(), "not-a-valid-token")

	assert.Assert(t, err != nil, "expected error for invalid token")
	assert.Assert(t, errors.Is(err, domain.ErrInvalidToken),
		"expected ErrInvalidToken, got: %v", err)
}
