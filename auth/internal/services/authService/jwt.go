package authService

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// Claims defines the JWT claims structure for access and refresh tokens.
type Claims struct {
	jwt.RegisteredClaims
	Email       string `json:"email"`
	TokenFamily string `json:"token_family,omitempty"`
}

// GenerateAccessToken creates an RS256-signed JWT access token with standard claims.
func (s *AuthService) GenerateAccessToken(userID, email string) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			Issuer:    "mpc-2fa-auth",
		},
		Email: email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, jti, nil
}

// GenerateRefreshToken creates an RS256-signed JWT refresh token with a token_family claim.
func (s *AuthService) GenerateRefreshToken(userID, email, tokenFamily string) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			Issuer:    "mpc-2fa-auth",
		},
		Email:       email,
		TokenFamily: tokenFamily,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return tokenString, jti, nil
}

// ParseToken validates and parses a JWT token string, enforcing RS256 algorithm only.
func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Algorithm check is delegated to jwt.WithValidMethods below (SEC-01).
		return s.publicKey, nil
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer("mpc-2fa-auth"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		// Check for expiration specifically
		if s.isExpiredError(err) {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

// isExpiredError checks whether a JWT parsing error is due to token expiration.
func (s *AuthService) isExpiredError(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}

// LoadRSAKeys reads RSA private and public keys from PEM files.
func LoadRSAKeys(privatePath, publicPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateData, err := os.ReadFile(privatePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}

	publicData, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse public key: %w", err)
	}

	return privateKey, publicKey, nil
}
