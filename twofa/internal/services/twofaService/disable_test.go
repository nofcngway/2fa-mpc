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

// disableSuite holds shared setup for Disable tests.
type disableSuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	mpcClients     []*mocks.MPCClientMock
	service        *twofaService.TwoFAService
}

func newDisableSuite(t *testing.T) *disableSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]*mocks.MPCClientMock, 3)
	mpcInterfaces := make([]twofaService.MPCClient, 3)
	for i := range 3 {
		mpcClients[i] = mocks.NewMPCClientMock(mc)
		mpcInterfaces[i] = mpcClients[i]
	}

	eventProducer := mocks.NewEventProducerMock(mc)
	eventProducer.PublishEventMock.Optional().Return(nil)
	eventProducer.CloseMock.Optional().Return(nil)

	service := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcInterfaces, eventProducer, "test-secret", 5*time.Second,
	)

	return &disableSuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		mpcClients:     mpcClients,
		service:        service,
	}
}

// makeAllMocksOptional marks all storage/session mocks as optional so minimock
// does not fail on unexpected non-calls.
func (ds *disableSuite) makeAllMocksOptional() {
	ds.storage.CreateTwoFARecordMock.Optional()
	ds.storage.EnableTwoFAMock.Optional()
	ds.storage.StoreBatchBackupCodesMock.Optional()
	ds.storage.DeleteTwoFARecordMock.Optional()
	ds.storage.DeleteBackupCodesMock.Optional()
	ds.storage.GetUnusedBackupCodeHashesMock.Optional()
	ds.storage.MarkBackupCodeUsedMock.Optional()
	ds.sessionStorage.IncrementRateLimitMock.Optional()
	ds.sessionStorage.GetRateLimitMock.Optional()
	ds.sessionStorage.GetUsedOTPCounterMock.Optional()
	ds.sessionStorage.SetUsedOTPCounterMock.Optional()
	ds.sessionStorage.DeleteKeysMock.Optional()
}

func TestDisable_Success(t *testing.T) {
	ds := newDisableSuite(t)
	ds.makeAllMocksOptional()

	// Use real Shamir split to get valid shares
	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	// Record exists and is enabled
	ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)

	// MPC RetrieveShare returns valid shares
	for i := range 3 {
		data := shamirPkg[i]
		ds.mpcClients[i].RetrieveShareMock.Set(func(_ context.Context, _ string, _ int) ([]byte, error) {
			return data, nil
		})
	}

	// All 3 DeleteShare succeed
	for i := range 3 {
		ds.mpcClients[i].DeleteShareMock.Set(func(_ context.Context, userID string) error {
			assert.Equal(t, userID, "test-user")
			return nil
		})
	}

	// PG cleanup
	ds.storage.DeleteBackupCodesMock.Expect(minimock.AnyContext, "test-user").Return(nil)
	ds.storage.DeleteTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(nil)

	// OTP reuse check: no prior counter stored
	ds.sessionStorage.GetUsedOTPCounterMock.Set(func(_ context.Context, _ string) (int64, error) {
		return 0, models.ErrCounterNotFound
	})
	ds.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return nil
	})

	// Redis cleanup
	ds.sessionStorage.DeleteKeysMock.Set(func(_ context.Context, keys ...string) error {
		return nil
	})

	err := ds.service.Disable(t.Context(), "test-user", code)
	assert.NilError(t, err)
}

func TestDisable_InvalidOTP(t *testing.T) {
	ds := newDisableSuite(t)
	ds.makeAllMocksOptional()

	shamirPkg := shamirSplit(t, testSecret)

	ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)

	for i := range 3 {
		data := shamirPkg[i]
		// Optional because first-2-wins may skip the 3rd client.
		ds.mpcClients[i].RetrieveShareMock.Optional().Set(func(_ context.Context, _ string, _ int) ([]byte, error) {
			return data, nil
		})
	}

	// Should NOT reach DeleteShare
	for i := range 3 {
		ds.mpcClients[i].DeleteShareMock.Optional()
	}

	err := ds.service.Disable(t.Context(), "test-user", "000000")
	assert.Assert(t, err != nil, "expected error for invalid OTP")
	assert.Assert(t, !errors.Is(err, twofaService.ErrNotSetUp))
	assert.Assert(t, !errors.Is(err, twofaService.ErrNotEnabled))

	// Verify NO DeleteShare calls were made
	for i := range 3 {
		assert.Equal(t, ds.mpcClients[i].DeleteShareAfterCounter(), uint64(0),
			"no DeleteShare calls expected for invalid OTP")
	}
}

func TestDisable_ShareDeletionFails(t *testing.T) {
	ds := newDisableSuite(t)
	ds.makeAllMocksOptional()

	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)

	for i := range 3 {
		data := shamirPkg[i]
		ds.mpcClients[i].RetrieveShareMock.Set(func(_ context.Context, _ string, _ int) ([]byte, error) {
			return data, nil
		})
	}

	// OTP reuse check: no prior counter stored
	ds.sessionStorage.GetUsedOTPCounterMock.Set(func(_ context.Context, _ string) (int64, error) {
		return 0, models.ErrCounterNotFound
	})
	ds.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return nil
	})

	// First 2 succeed, third fails
	ds.mpcClients[0].DeleteShareMock.Set(func(_ context.Context, _ string) error {
		return nil
	})
	ds.mpcClients[1].DeleteShareMock.Set(func(_ context.Context, _ string) error {
		return nil
	})
	ds.mpcClients[2].DeleteShareMock.Set(func(_ context.Context, _ string) error {
		return errors.New("node 2 unreachable")
	})

	err := ds.service.Disable(t.Context(), "test-user", code)
	assert.Assert(t, err != nil, "expected error when share deletion fails")

	// Assert DeleteTwoFARecord NOT called (twofa_record stays enabled per D-13)
	assert.Equal(t, ds.storage.DeleteTwoFARecordAfterCounter(), uint64(0),
		"DeleteTwoFARecord should NOT be called when share deletion fails")
	assert.Equal(t, ds.storage.DeleteBackupCodesAfterCounter(), uint64(0),
		"DeleteBackupCodes should NOT be called when share deletion fails")
}

func TestDisable_NotSetUp(t *testing.T) {
	ds := newDisableSuite(t)
	ds.makeAllMocksOptional()

	ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(nil, nil)

	// Make MPC optional
	for i := range 3 {
		ds.mpcClients[i].RetrieveShareMock.Optional()
		ds.mpcClients[i].DeleteShareMock.Optional()
		ds.mpcClients[i].StoreShareMock.Optional()
	}

	err := ds.service.Disable(t.Context(), "test-user", "123456")
	assert.Assert(t, errors.Is(err, twofaService.ErrNotSetUp),
		"expected ErrNotSetUp, got: %v", err)

	// No MPC calls
	for i := range 3 {
		assert.Equal(t, ds.mpcClients[i].RetrieveShareAfterCounter(), uint64(0))
		assert.Equal(t, ds.mpcClients[i].DeleteShareAfterCounter(), uint64(0))
	}
}

func TestDisable_NotEnabled(t *testing.T) {
	ds := newDisableSuite(t)
	ds.makeAllMocksOptional()

	ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: false}, nil,
	)

	// Make MPC optional
	for i := range 3 {
		ds.mpcClients[i].RetrieveShareMock.Optional()
		ds.mpcClients[i].DeleteShareMock.Optional()
		ds.mpcClients[i].StoreShareMock.Optional()
	}

	err := ds.service.Disable(t.Context(), "test-user", "123456")
	assert.Assert(t, errors.Is(err, twofaService.ErrNotEnabled),
		"expected ErrNotEnabled, got: %v", err)

	// No MPC calls
	for i := range 3 {
		assert.Equal(t, ds.mpcClients[i].RetrieveShareAfterCounter(), uint64(0))
		assert.Equal(t, ds.mpcClients[i].DeleteShareAfterCounter(), uint64(0))
	}
}
