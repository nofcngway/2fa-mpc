package auth_service_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/services/auth_service"
	"github.com/vbncursed/vkr/auth/internal/services/auth_service/mocks"
)

// logoutAllSuite holds shared setup for logout-all tests.
type logoutAllSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *auth_service.AuthService
}

func newLogoutAllSuite(t *testing.T) *logoutAllSuite {
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
	return &logoutAllSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
	}
}

func TestLogoutAll_Success(t *testing.T) {
	s := newLogoutAllSuite(t)

	deleteCalled := false
	s.sessionStorage.DeleteAllUserTokensMock.Set(func(_ context.Context, userID string) error {
		deleteCalled = true
		assert.Equal(t, userID, "user-123", "should delete tokens for correct user")
		return nil
	})

	err := s.service.LogoutAll(t.Context(), "user-123")

	assert.NilError(t, err)
	assert.Assert(t, deleteCalled, "DeleteAllUserTokens should have been called")
}

