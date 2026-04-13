package redisstorage

import (
	"context"
	"time"

	"github.com/vbncursed/vkr/twofa/internal/models"
)

// NoOpSessionStorage is a no-op implementation of twofaService.SessionStorage.
// Used as fallback when Redis is unavailable — rate limiting and OTP reuse
// prevention are disabled, but the service does not panic on nil receiver.
type NoOpSessionStorage struct{}

func (n *NoOpSessionStorage) IncrementRateLimit(_ context.Context, _ string, _ time.Duration) (int64, error) {
	return 0, nil
}

func (n *NoOpSessionStorage) GetRateLimit(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (n *NoOpSessionStorage) SetUsedOTPCounter(_ context.Context, _ string, _ int64, _ time.Duration) error {
	return nil
}

func (n *NoOpSessionStorage) GetUsedOTPCounter(_ context.Context, _ string) (int64, error) {
	return 0, models.ErrCounterNotFound
}

func (n *NoOpSessionStorage) DeleteKeys(_ context.Context, _ ...string) error {
	return nil
}
