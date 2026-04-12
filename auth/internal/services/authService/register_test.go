package authService_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/auth/internal/domain"
	"github.com/vbncursed/vkr/auth/internal/models"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
	"github.com/vbncursed/vkr/auth/internal/services/authService/mocks"
)

// registerSuite holds shared setup for register tests.
type registerSuite struct {
	mc      *minimock.Controller
	storage *mocks.StorageMock
	service *authService.AuthService
}

func newRegisterSuite(t *testing.T) *registerSuite {
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	service := authService.NewAuthService(storage, nil)
	return &registerSuite{
		mc:      mc,
		storage: storage,
		service: service,
	}
}

func TestRegister_Success(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(nil, nil)
	s.storage.CreateUserMock.Set(func(_ context.Context, user *models.User) error {
		assert.Assert(t, user.ID != "", "user ID should not be empty")
		assert.Equal(t, user.Email, "test@example.com")
		assert.Assert(t, user.PasswordHash != "", "password hash should not be empty")
		return nil
	})

	user, err := s.service.Register(context.Background(), "test@example.com", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Assert(t, user != nil, "expected user, got nil")
	assert.Assert(t, user.ID != "", "user ID should not be empty")
	assert.Equal(t, user.Email, "test@example.com")

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

			_, err := s.service.Register(context.Background(), tt.email, "MyStr0ng!Pass99")

			assert.Assert(t, err != nil, "expected error for email %q", tt.email)
			assert.Assert(t, errors.Is(err, domain.ErrInvalidEmail),
				"expected ErrInvalidEmail, got: %v", err)
		})
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	s := newRegisterSuite(t)

	_, err := s.service.Register(context.Background(), "test@example.com", "short")

	assert.Assert(t, err != nil, "expected error for weak password")

	var valErr *domain.PasswordValidationError
	assert.Assert(t, errors.As(err, &valErr),
		"expected PasswordValidationError, got %T: %v", err, err)
}

func TestRegister_DuplicateEmail_PreCheck(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "existing@example.com").
		Return(&models.User{Email: "existing@example.com"}, nil)

	_, err := s.service.Register(context.Background(), "existing@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for duplicate email")
	assert.Assert(t, errors.Is(err, domain.ErrDuplicateEmail),
		"expected ErrDuplicateEmail, got: %v", err)
}

func TestRegister_DuplicateEmail_RaceCondition(t *testing.T) {
	s := newRegisterSuite(t)

	// Pre-check passes (user not found), but CreateUser hits unique constraint
	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "race@example.com").Return(nil, nil)
	s.storage.CreateUserMock.Return(domain.ErrDuplicateEmail)

	_, err := s.service.Register(context.Background(), "race@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for race condition duplicate")
	assert.Assert(t, errors.Is(err, domain.ErrDuplicateEmail),
		"expected ErrDuplicateEmail from race condition, got: %v", err)
}

func TestRegister_StorageError(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "test@example.com").Return(nil, nil)
	s.storage.CreateUserMock.Return(errors.New("connection refused"))

	_, err := s.service.Register(context.Background(), "test@example.com", "MyStr0ng!Pass99")

	assert.Assert(t, err != nil, "expected error for storage failure")
	assert.Assert(t, !errors.Is(err, domain.ErrDuplicateEmail),
		"storage error should not be ErrDuplicateEmail")
	assert.Assert(t, strings.Contains(err.Error(), "create user"),
		"error should be wrapped with context, got: %v", err)
}

func TestRegister_EmailNormalization(t *testing.T) {
	s := newRegisterSuite(t)

	s.storage.GetUserByEmailMock.Expect(minimock.AnyContext, "user@example.com").Return(nil, nil)
	s.storage.CreateUserMock.Set(func(_ context.Context, user *models.User) error {
		assert.Equal(t, user.Email, "user@example.com", "email should be lowercased and trimmed")
		return nil
	})

	user, err := s.service.Register(context.Background(), "  User@Example.COM  ", "MyStr0ng!Pass99")

	assert.NilError(t, err)
	assert.Equal(t, user.Email, "user@example.com")
}
