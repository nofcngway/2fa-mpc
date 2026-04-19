package authService_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/services/authService/mocks"
)

// registerSuite holds shared setup for register tests.
type registerSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	service        *authService.AuthService
}

func newRegisterSuite(t *testing.T) *registerSuite {
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
	return &registerSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		service:        service,
	}
}

func TestRegister_Success(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(nil, domain.ErrUserNotFound)
	s.storage.CreateUserMock.Set(func(_ context.Context, user *domain.User) error {
		assert.Assert(t, user.ID != "", "user ID should not be empty")
		assert.Equal(t, user.Email, "test@example.com")
		assert.Assert(t, user.PasswordHash != "", "password hash should not be empty")
		return nil
	})
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		assert.Assert(t, jti != "", "JTI should not be empty")
		assert.Assert(t, userID != "", "userID should not be empty")
		assert.Assert(t, tokenFamily != "", "token family should not be empty")
		assert.Equal(t, ttl, 168*time.Hour)
		return nil
	})

	user, accessToken, refreshToken, err := s.service.Register(t.Context(), "test@example.com", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Assert(t, user != nil, "expected user, got nil")
	assert.Assert(t, user.ID != "", "user ID should not be empty")
	assert.Equal(t, user.Email, "test@example.com")
	assert.Assert(t, accessToken != "", "access token should not be empty")
	assert.Assert(t, refreshToken != "", "refresh token should not be empty")

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("MyStr0ng!Pass99"))
	assert.NilError(t, err, "password hash does not match original password with bcrypt")
}

func TestRegister_InvalidEmail(t *testing.T) {
	cases := []struct {
		name  string
		email string
	}{
		{"no at sign", "invalid-email"},
		{"no domain", "user@"},
		{"no dot in domain", "user@localhost"},
		{"empty email", ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := newRegisterSuite(t)

			_, _, _, err := s.service.Register(t.Context(), tt.email, "MyStr0ng!Pass99")

			assert.Assert(t, err != nil, "expected error for email %q", tt.email)
			assert.Assert(t, errors.Is(err, domain.ErrInvalidEmail),
				"expected ErrInvalidEmail, got: %v", err)
		})
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	s := newRegisterSuite(t)

	_, _, _, err := s.service.Register(t.Context(), "test@example.com", "short")

	assert.Assert(t, err != nil, "expected error for weak password")

	_, ok := errors.AsType[*domain.PasswordValidationError](err)
	assert.Assert(t, ok,
		"expected PasswordValidationError, got %T: %v", err, err)
}

func TestRegister_DuplicateEmail_PreCheck(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "existing@example.com").
		Return(&domain.User{Email: "existing@example.com"}, nil)

	_, _, _, err := s.service.Register(t.Context(), "existing@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for duplicate email")
	assert.Assert(t, errors.Is(err, domain.ErrDuplicateEmail),
		"expected ErrDuplicateEmail, got: %v", err)
}

func TestRegister_DuplicateEmail_RaceCondition(t *testing.T) {
	s := newRegisterSuite(t)

	// Pre-check passes (user not found), but CreateUser hits unique constraint
	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "race@example.com").Return(nil, domain.ErrUserNotFound)
	s.storage.CreateUserMock.Return(domain.ErrDuplicateEmail)

	_, _, _, err := s.service.Register(t.Context(), "race@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for race condition duplicate")
	assert.Assert(t, errors.Is(err, domain.ErrDuplicateEmail),
		"expected ErrDuplicateEmail from race condition, got: %v", err)
}

func TestRegister_StorageError(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(nil, domain.ErrUserNotFound)
	s.storage.CreateUserMock.Return(errors.New("connection refused"))

	_, _, _, err := s.service.Register(t.Context(), "test@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for storage failure")
	assert.Assert(t, !errors.Is(err, domain.ErrDuplicateEmail),
		"storage error should not be ErrDuplicateEmail")
	assert.Assert(t, strings.Contains(err.Error(), "create user"),
		"error should be wrapped with context, got: %v", err)
}

func TestRegister_EmailNormalization(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "user@example.com").Return(nil, domain.ErrUserNotFound)
	s.storage.CreateUserMock.Set(func(_ context.Context, user *domain.User) error {
		assert.Equal(t, user.Email, "user@example.com", "email should be lowercased and trimmed")
		return nil
	})
	s.sessionStorage.StoreRefreshTokenMock.Return(nil)

	user, accessToken, refreshToken, err := s.service.Register(t.Context(), "  User@Example.COM  ", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Equal(t, user.Email, "user@example.com")
	assert.Assert(t, accessToken != "", "access token should not be empty after register")
	assert.Assert(t, refreshToken != "", "refresh token should not be empty after register")
}

func TestRegister_StoresRefreshToken(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "store@example.com").Return(nil, domain.ErrUserNotFound)
	s.storage.CreateUserMock.Return(nil)

	storeCalled := false
	s.sessionStorage.StoreRefreshTokenMock.Set(func(_ context.Context, jti, userID, tokenFamily string, ttl time.Duration) error {
		storeCalled = true
		assert.Assert(t, jti != "", "JTI should not be empty")
		assert.Assert(t, userID != "", "userID should not be empty")
		assert.Assert(t, tokenFamily != "", "token family should not be empty")
		assert.Equal(t, ttl, 168*time.Hour)
		return nil
	})

	_, _, _, err := s.service.Register(t.Context(), "store@example.com", "MyStr0ng!Pass99")
	assert.NilError(t, err)
	assert.Assert(t, storeCalled, "StoreRefreshToken should have been called after registration")
}
