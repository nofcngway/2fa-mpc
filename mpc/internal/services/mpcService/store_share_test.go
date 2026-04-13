package mpcService_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/mpc/internal/models"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService/mocks"
)

type storeSuite struct {
	mc      *minimock.Controller
	storage *mocks.StorageMock
	service *mpcService.MPCService
}

func newStoreSuite(t *testing.T) *storeSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)
	key := []byte("01234567890123456789012345678901") // exactly 32 bytes
	service := mpcService.NewMPCService(storage, key, 1, eventProducer)
	return &storeSuite{mc: mc, storage: storage, service: service}
}

func TestStoreShareHappyPath(t *testing.T) {
	s := newStoreSuite(t)

	s.storage.CreateShareMock.Set(func(_ context.Context, share *models.Share) error {
		assert.Assert(t, share.ID != "", "share ID should be generated")
		assert.Equal(t, share.UserID, "user-123")
		assert.Equal(t, share.ShareIndex, 0)
		assert.Assert(t, len(share.EncryptedData) > 0, "encrypted data should not be empty")
		assert.Assert(t, len(share.Nonce) == 12, "nonce should be 12 bytes")
		return nil
	})

	shareID, err := s.service.StoreShare(t.Context(), "user-123", 0, []byte("share-data"))
	assert.NilError(t, err)
	assert.Assert(t, shareID != "", "share ID should be returned")
}

func TestStoreShareDuplicate(t *testing.T) {
	s := newStoreSuite(t)

	s.storage.CreateShareMock.Return(models.ErrDuplicateShare)

	_, err := s.service.StoreShare(t.Context(), "user-123", 0, []byte("share-data"))
	assert.Assert(t, err != nil, "expected error for duplicate share")
	assert.Assert(t, errors.Is(err, models.ErrDuplicateShare),
		"expected ErrDuplicateShare, got: %v", err)
}

func TestStoreShareStorageError(t *testing.T) {
	s := newStoreSuite(t)

	s.storage.CreateShareMock.Return(errors.New("connection refused"))

	_, err := s.service.StoreShare(t.Context(), "user-123", 0, []byte("share-data"))
	assert.Assert(t, err != nil, "expected error for storage failure")
	assert.Assert(t, !errors.Is(err, models.ErrDuplicateShare),
		"generic error should not be ErrDuplicateShare")
}

func TestStoreShareEmptyData(t *testing.T) {
	s := newStoreSuite(t)

	s.storage.CreateShareMock.Set(func(_ context.Context, share *models.Share) error {
		// Empty share data should still produce encrypted data (GCM tag).
		assert.Assert(t, len(share.EncryptedData) > 0, "encrypted data for empty input should contain GCM tag")
		return nil
	})

	shareID, err := s.service.StoreShare(t.Context(), "user-123", 0, []byte{})
	assert.NilError(t, err)
	assert.Assert(t, shareID != "", "share ID should be returned for empty data")
}
