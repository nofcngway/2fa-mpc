// Package authService implements authentication business logic including
// registration, login, JWT token management, and session lifecycle.
package authService

import (
	"context"
	"crypto/rsa"
	"errors"
	"time"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

//go:generate minimock -i Storage -o ./mocks/ -s _mock.go
//go:generate minimock -i SessionStorage -o ./mocks/ -s _mock.go

// Storage defines the interface for persistent data access.
type Storage interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	StoreRefreshToken(ctx context.Context, jti, userID, tokenFamily string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, jti string) (*domain.RefreshTokenData, error)
	DeleteRefreshToken(ctx context.Context, jti string) error
	DeleteTokenFamily(ctx context.Context, family, userID string) error
	DeleteAllUserTokens(ctx context.Context, userID string) error
}

// AuthService implements authentication business logic.
type AuthService struct {
	storage         Storage
	sessionStorage  SessionStorage
	eventProducer   EventProducer
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewAuthService creates a new AuthService instance. Returns an error if any required dependency is nil.
func NewAuthService(
	storage Storage,
	sessionStorage SessionStorage,
	eventProducer EventProducer,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) (*AuthService, error) {
	var errs []error
	if storage == nil {
		errs = append(errs, errors.New("storage is required"))
	}
	if sessionStorage == nil {
		errs = append(errs, errors.New("session storage is required"))
	}
	if eventProducer == nil {
		errs = append(errs, errors.New("event producer is required"))
	}
	if privateKey == nil {
		errs = append(errs, errors.New("private key is required"))
	}
	if publicKey == nil {
		errs = append(errs, errors.New("public key is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &AuthService{
		storage:         storage,
		sessionStorage:  sessionStorage,
		eventProducer:   eventProducer,
		privateKey:      privateKey,
		publicKey:       publicKey,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}
