package authService_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/services/authService/mocks"
)

// logoutAllSuite holds shared setup for logout-all tests.
type logoutAllSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *authService.AuthService
}

func newLogoutAllSuite(t *testing.T) *logoutAllSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)
	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err, "failed to generate RSA key pair for test")

	service := authService.NewAuthService(
		storage, sessionStorage, eventProducer,
		privateKey, &privateKey.PublicKey,
		15*time.Minute, 168*time.Hour,
	)
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

	err := s.service.LogoutAll(context.Background(), "user-123")

	assert.NilError(t, err)
	assert.Assert(t, deleteCalled, "DeleteAllUserTokens should have been called")
}

func TestLogoutAll_ReturnsNilOnSuccess(t *testing.T) {
	s := newLogoutAllSuite(t)

	s.sessionStorage.DeleteAllUserTokensMock.Expect(minimock.AnyContext, "user-456").Return(nil)

	err := s.service.LogoutAll(context.Background(), "user-456")

	assert.NilError(t, err)
}
