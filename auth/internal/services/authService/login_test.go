package authService_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/services/authService/mocks"
)

// loginSuite holds shared setup for login tests.
type loginSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *authService.AuthService
}

func newLoginSuite(t *testing.T) *loginSuite {
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
	return &loginSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
	}
}

func TestLogin_Success(t *testing.T) {
	s := newLoginSuite(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("MyStr0ng!Pass99"), 4)
	assert.NilError(t, err)

	existingUser := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(existingUser, nil)
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		assert.Assert(t, jti != "", "JTI should not be empty")
		assert.Equal(t, userID, "user-123")
		assert.Assert(t, tokenFamily != "", "token family should not be empty")
		assert.Equal(t, ttl, 168*time.Hour)
		return nil
	})

	user, accessToken, refreshToken, err := s.service.Login(t.Context(), "test@example.com", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Assert(t, user != nil, "expected user, got nil")
	assert.Equal(t, user.ID, "user-123")
	assert.Assert(t, accessToken != "", "access token should not be empty")
	assert.Assert(t, refreshToken != "", "refresh token should not be empty")
}

func TestLogin_NonExistentEmail(t *testing.T) {
	s := newLoginSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "nobody@example.com").Return(nil, domain.ErrUserNotFound)

	_, _, _, err := s.service.Login(t.Context(), "nobody@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for non-existent email")
	assert.Assert(t, errors.Is(err, domain.ErrInvalidCredentials),
		"expected ErrInvalidCredentials, got: %v", err)
}

func TestLogin_WrongPassword(t *testing.T) {
	s := newLoginSuite(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("MyStr0ng!Pass99"), 4)
	assert.NilError(t, err)

	existingUser := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(hash),
	}

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(existingUser, nil)

	_, _, _, err = s.service.Login(t.Context(), "test@example.com", "WrongPassword!1")

	assert.Assert(t, err != nil, "expected error for wrong password")
	assert.Assert(t, errors.Is(err, domain.ErrInvalidCredentials),
		"expected ErrInvalidCredentials (same as non-existent), got: %v", err)
}

func TestLogin_StoresRefreshToken(t *testing.T) {
	s := newLoginSuite(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("MyStr0ng!Pass99"), 4)
	assert.NilError(t, err)

	existingUser := &domain.User{
		ID:           "user-456",
		Email:        "store@example.com",
		PasswordHash: string(hash),
	}

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "store@example.com").Return(existingUser, nil)

	storeCalled := false
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		storeCalled = true
		assert.Equal(t, userID, "user-456")
		assert.Equal(t, ttl, 168*time.Hour)
		assert.Assert(t, tokenFamily != "", "token family UUID must be set")
		return nil
	})

	_, _, _, err = s.service.Login(t.Context(), "store@example.com", "MyStr0ng!Pass99")
	assert.NilError(t, err)
	assert.Assert(t, storeCalled, "StoreRefreshToken should have been called")
}

func TestLogin_EmailNormalization(t *testing.T) {
	s := newLoginSuite(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("MyStr0ng!Pass99"), 4)
	assert.NilError(t, err)

	existingUser := &domain.User{
		ID:           "user-789",
		Email:        "user@example.com",
		PasswordHash: string(hash),
	}

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "user@example.com").Return(existingUser, nil)
	s.sessionStorage.StoreRefreshTokenMock.Return(nil)

	user, _, _, err := s.service.Login(t.Context(), "  User@Example.COM  ", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Assert(t, user != nil, "expected user, got nil")
}
