package authService_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/vbncursed/vkr/auth/internal/models"
	"github.com/vbncursed/vkr/auth/internal/services/authService"
)

type mockStorage struct {
	users     map[string]*models.User
	createErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{users: make(map[string]*models.User)}
}

// CreateUser stores the user in the mock map.
func (m *mockStorage) CreateUser(ctx context.Context, user *models.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.Email] = user
	return nil
}

// GetUserByEmail returns the user if found, (nil, nil) otherwise.
func (m *mockStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		setupMock   func(*mockStorage)
		wantErr     bool
		wantErrType string // "validation", "duplicate", "invalid_email", ""
	}{
		{
			name:     "successful registration",
			email:    "test@example.com",
			password: "MyStr0ng!Pass99",
			wantErr:  false,
		},
		{
			name:        "invalid email no at sign",
			email:       "invalid-email",
			password:    "MyStr0ng!Pass99",
			wantErr:     true,
			wantErrType: "invalid_email",
		},
		{
			name:        "invalid email no domain",
			email:       "user@",
			password:    "MyStr0ng!Pass99",
			wantErr:     true,
			wantErrType: "invalid_email",
		},
		{
			name:        "invalid email no dot in domain",
			email:       "user@localhost",
			password:    "MyStr0ng!Pass99",
			wantErr:     true,
			wantErrType: "invalid_email",
		},
		{
			name:        "empty email",
			email:       "",
			password:    "MyStr0ng!Pass99",
			wantErr:     true,
			wantErrType: "invalid_email",
		},
		{
			name:        "weak password",
			email:       "test@example.com",
			password:    "short",
			wantErr:     true,
			wantErrType: "validation",
		},
		{
			name:     "duplicate email",
			email:    "existing@example.com",
			password: "MyStr0ng!Pass99",
			setupMock: func(m *mockStorage) {
				m.users["existing@example.com"] = &models.User{Email: "existing@example.com"}
			},
			wantErr:     true,
			wantErrType: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockStorage()
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}
			svc := authService.NewAuthService(mock, nil)
			user, err := svc.Register(context.Background(), tt.email, tt.password)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				switch tt.wantErrType {
				case "validation":
					var valErr *authService.PasswordValidationError
					if !errors.As(err, &valErr) {
						t.Fatalf("expected PasswordValidationError, got %T: %v", err, err)
					}
				case "duplicate":
					if !errors.Is(err, authService.ErrDuplicateEmail) {
						t.Fatalf("expected ErrDuplicateEmail, got: %v", err)
					}
				case "invalid_email":
					if !errors.Is(err, authService.ErrInvalidEmail) {
						t.Fatalf("expected ErrInvalidEmail, got: %v", err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user == nil {
				t.Fatal("expected user, got nil")
			}
			if user.ID == "" {
				t.Error("user ID should not be empty")
			}
			if user.Email != strings.ToLower(strings.TrimSpace(tt.email)) {
				t.Errorf("email mismatch: got %q", user.Email)
			}
			// Verify password was hashed with bcrypt
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password)); err != nil {
				t.Error("password hash does not match original password with bcrypt")
			}
		})
	}
}
