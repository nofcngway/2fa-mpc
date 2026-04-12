package authService

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/vbncursed/vkr/auth/internal/models"
)

//go:generate minimock -i Storage -o ./mocks/ -s _mock.go
//go:generate minimock -i SessionStorage -o ./mocks/ -s _mock.go

// Storage defines the interface for persistent data access.
type Storage interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

// SessionStorage defines the interface for session/cache operations.
type SessionStorage interface {
	StoreRefreshToken(ctx context.Context, jti, userID, tokenFamily string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, jti string) (*RefreshTokenData, error)
	DeleteRefreshToken(ctx context.Context, jti string) error
	DeleteTokenFamily(ctx context.Context, family string) error
	DeleteAllUserTokens(ctx context.Context, userID string) error
}

// RefreshTokenData holds the data associated with a stored refresh token.
type RefreshTokenData struct {
	UserID      string `json:"user_id"`
	TokenFamily string `json:"token_family"`
	IssuedAt    string `json:"issued_at"`
}

// AuthService implements authentication business logic.
type AuthService struct {
	storage         Storage
	sessionStorage  SessionStorage
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(
	storage Storage,
	sessionStorage SessionStorage,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		storage:         storage,
		sessionStorage:  sessionStorage,
		privateKey:      privateKey,
		publicKey:       publicKey,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}
