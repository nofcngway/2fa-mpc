package twofaService_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/twofa/internal/models"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService/mocks"
)

func TestGetStatus_Found(t *testing.T) {
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]twofaService.MPCClient, 3)
	for i := 0; i < 3; i++ {
		m := mocks.NewMPCClientMock(mc)
		m.RetrieveShareMock.Optional()
		m.StoreShareMock.Optional()
		m.DeleteShareMock.Optional()
		mpcClients[i] = m
	}

	service := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcClients, "test-secret", 5*time.Second,
	)

	// Make all optional
	storage.CreateTwoFARecordMock.Optional()
	storage.EnableTwoFAMock.Optional()
	storage.StoreBatchBackupCodesMock.Optional()
	storage.DeleteTwoFARecordMock.Optional()
	storage.DeleteBackupCodesMock.Optional()
	sessionStorage.IncrementRateLimitMock.Optional()
	sessionStorage.GetRateLimitMock.Optional()
	sessionStorage.GetUsedOTPCounterMock.Optional()
	sessionStorage.SetUsedOTPCounterMock.Optional()
	sessionStorage.DeleteKeysMock.Optional()

	createdAt := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	expected := &models.TwoFARecord{
		UserID:    "test-user",
		IsEnabled: true,
		CreatedAt: createdAt,
	}

	storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(expected, nil)

	record, err := service.GetStatus(context.Background(), "test-user")
	assert.NilError(t, err)
	assert.Assert(t, record != nil)
	assert.Equal(t, record.UserID, "test-user")
	assert.Assert(t, record.IsEnabled)
	assert.Equal(t, record.CreatedAt, createdAt)
}

func TestGetStatus_NotFound(t *testing.T) {
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]twofaService.MPCClient, 3)
	for i := 0; i < 3; i++ {
		m := mocks.NewMPCClientMock(mc)
		m.RetrieveShareMock.Optional()
		m.StoreShareMock.Optional()
		m.DeleteShareMock.Optional()
		mpcClients[i] = m
	}

	service := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcClients, "test-secret", 5*time.Second,
	)

	storage.CreateTwoFARecordMock.Optional()
	storage.EnableTwoFAMock.Optional()
	storage.StoreBatchBackupCodesMock.Optional()
	storage.DeleteTwoFARecordMock.Optional()
	storage.DeleteBackupCodesMock.Optional()
	sessionStorage.IncrementRateLimitMock.Optional()
	sessionStorage.GetRateLimitMock.Optional()
	sessionStorage.GetUsedOTPCounterMock.Optional()
	sessionStorage.SetUsedOTPCounterMock.Optional()
	sessionStorage.DeleteKeysMock.Optional()

	storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(nil, nil)

	record, err := service.GetStatus(context.Background(), "test-user")
	assert.NilError(t, err)
	assert.Assert(t, record == nil)
}

func TestGetStatus_Error(t *testing.T) {
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]twofaService.MPCClient, 3)
	for i := 0; i < 3; i++ {
		m := mocks.NewMPCClientMock(mc)
		m.RetrieveShareMock.Optional()
		m.StoreShareMock.Optional()
		m.DeleteShareMock.Optional()
		mpcClients[i] = m
	}

	service := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcClients, "test-secret", 5*time.Second,
	)

	storage.CreateTwoFARecordMock.Optional()
	storage.EnableTwoFAMock.Optional()
	storage.StoreBatchBackupCodesMock.Optional()
	storage.DeleteTwoFARecordMock.Optional()
	storage.DeleteBackupCodesMock.Optional()
	sessionStorage.IncrementRateLimitMock.Optional()
	sessionStorage.GetRateLimitMock.Optional()
	sessionStorage.GetUsedOTPCounterMock.Optional()
	sessionStorage.SetUsedOTPCounterMock.Optional()
	sessionStorage.DeleteKeysMock.Optional()

	dbErr := errors.New("database connection failed")
	storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(nil, dbErr)

	record, err := service.GetStatus(context.Background(), "test-user")
	assert.Assert(t, err != nil)
	assert.Assert(t, record == nil)
	assert.Assert(t, errors.Is(err, dbErr), "expected wrapped dbErr, got: %v", err)
}
