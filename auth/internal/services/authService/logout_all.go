package authService

import "context"

// LogoutAll revokes all sessions for a user by deleting all their tokens from Redis.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessionStorage.DeleteAllUserTokens(ctx, userID)
}
