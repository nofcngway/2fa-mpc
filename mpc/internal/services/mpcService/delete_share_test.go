package mpcService_test

import (
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService/mocks"
)

type deleteSuite struct {
	mc      *minimock.Controller
	storage *mocks.StorageMock
	service *mpcService.MPCService
}

func newDeleteSuite(t *testing.T) *deleteSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)
	key := []byte("01234567890123456789012345678901") // exactly 32 bytes
	service, err := mpcService.NewMPCService(mpcService.Deps{
		Storage:       storage,
		EncryptionKey: key,
		NodeID:        1,
		EventProducer: eventProducer,
	})
	assert.NilError(t, err, "failed to create MPC service")
	return &deleteSuite{mc: mc, storage: storage, service: service}
}

func TestDeleteShareHappyPath(t *testing.T) {
	s := newDeleteSuite(t)

	s.storage.DeleteSharesByUserIDMock.Expect(minimock.AnyContext, "user-123").Return(3, nil)

	count, err := s.service.DeleteShare(t.Context(), "user-123")
	assert.NilError(t, err)
	assert.Equal(t, count, int64(3))
}

func TestDeleteShareNoShares(t *testing.T) {
	s := newDeleteSuite(t)

	s.storage.DeleteSharesByUserIDMock.Expect(minimock.AnyContext, "user-123").Return(0, nil)

	count, err := s.service.DeleteShare(t.Context(), "user-123")
	assert.NilError(t, err)
	assert.Equal(t, count, int64(0))
}

func TestDeleteShareStorageError(t *testing.T) {
	s := newDeleteSuite(t)

	s.storage.DeleteSharesByUserIDMock.Expect(minimock.AnyContext, "user-123").Return(0, errors.New("connection refused"))

	_, err := s.service.DeleteShare(t.Context(), "user-123")
	assert.Assert(t, err != nil, "expected error for storage failure")
}
