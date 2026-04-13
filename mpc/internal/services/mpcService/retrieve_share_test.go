package mpcService_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/mpc/internal/models"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService"
	"github.com/vbncursed/vkr/mpc/internal/services/mpcService/mocks"
)

type retrieveSuite struct {
	mc      *minimock.Controller
	storage *mocks.StorageMock
	service *mpcService.MPCService
	key     []byte
}

func newRetrieveSuite(t *testing.T) *retrieveSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)
	key := []byte("01234567890123456789012345678901") // exactly 32 bytes
	service := mpcService.NewMPCService(storage, key, 1, eventProducer)
	return &retrieveSuite{mc: mc, storage: storage, service: service, key: key}
}

// testEncrypt encrypts plaintext with the given key using AES-256-GCM for test setup.
func testEncrypt(t *testing.T, key, plaintext []byte) (ciphertext, nonce []byte) {
	t.Helper()
	block, err := aes.NewCipher(key)
	assert.NilError(t, err)
	gcm, err := cipher.NewGCM(block)
	assert.NilError(t, err)
	nonce = make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	assert.NilError(t, err)
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce
}

func TestRetrieveShareHappyPath(t *testing.T) {
	s := newRetrieveSuite(t)
	plaintext := []byte("original share data")

	encData, nonce := testEncrypt(t, s.key, plaintext)

	s.storage.GetShareMock.Expect(minimock.AnyContext, "user-123", 0).Return(&models.Share{
		ID:            "share-id-1",
		UserID:        "user-123",
		ShareIndex:    0,
		EncryptedData: encData,
		Nonce:         nonce,
		CreatedAt:     time.Now(),
	}, nil)

	result, err := s.service.RetrieveShare(t.Context(), "user-123", 0)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, plaintext)
}

func TestRetrieveShareNotFound(t *testing.T) {
	s := newRetrieveSuite(t)

	s.storage.GetShareMock.Expect(minimock.AnyContext, "user-123", 0).Return(nil, models.ErrShareNotFound)

	_, err := s.service.RetrieveShare(t.Context(), "user-123", 0)
	assert.Assert(t, err != nil, "expected error for not found share")
	assert.Assert(t, errors.Is(err, models.ErrShareNotFound),
		"expected ErrShareNotFound, got: %v", err)
}

func TestRetrieveShareDecryptFailure(t *testing.T) {
	s := newRetrieveSuite(t)

	// Return corrupted encrypted data that cannot be decrypted.
	s.storage.GetShareMock.Expect(minimock.AnyContext, "user-123", 0).Return(&models.Share{
		ID:            "share-id-1",
		UserID:        "user-123",
		ShareIndex:    0,
		EncryptedData: []byte("corrupted-data-that-cannot-be-decrypted"),
		Nonce:         make([]byte, 12),
		CreatedAt:     time.Now(),
	}, nil)

	_, err := s.service.RetrieveShare(t.Context(), "user-123", 0)
	assert.Assert(t, err != nil, "expected error for decryption failure")
}

func TestRetrieveShareStorageError(t *testing.T) {
	s := newRetrieveSuite(t)

	s.storage.GetShareMock.Expect(minimock.AnyContext, "user-123", 0).Return(nil, errors.New("connection refused"))

	_, err := s.service.RetrieveShare(t.Context(), "user-123", 0)
	assert.Assert(t, err != nil, "expected error for storage failure")
	assert.Assert(t, !errors.Is(err, models.ErrShareNotFound),
		"generic error should not be ErrShareNotFound")
}
