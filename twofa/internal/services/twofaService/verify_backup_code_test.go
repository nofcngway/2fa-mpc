package twofaService_test

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/twofa/internal/models"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService/mocks"
)

func newBackupCodeService(t *testing.T) (*twofaService.TwoFAService, *mocks.StorageMock, *mocks.SessionStorageMock) {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]twofaService.MPCClient, 3)
	for i := range 3 {
		m := mocks.NewMPCClientMock(mc)
		m.RetrieveShareMock.Optional()
		m.StoreShareMock.Optional()
		m.DeleteShareMock.Optional()
		mpcClients[i] = m
	}

	ep := mocks.NewEventProducerMock(mc)
	ep.PublishEventMock.Optional().Return(nil)
	ep.CloseMock.Optional().Return(nil)

	// Mark all storage/session mocks optional by default
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

	svc := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcClients, ep, "test-secret", 5*time.Second,
	)
	return svc, storage, sessionStorage
}

func hashCode(t *testing.T, code string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashCode: %v", err)
	}
	return string(h)
}

func TestVerifyBackupCode_Success(t *testing.T) {
	svc, storage, _ := newBackupCodeService(t)

	code := "1234-5678"
	storage.GetUnusedBackupCodeHashesMock.Return([]models.BackupCodeRow{
		{ID: "code-1", CodeHash: hashCode(t, "0000-0000")},
		{ID: "code-2", CodeHash: hashCode(t, code)},
		{ID: "code-3", CodeHash: hashCode(t, "9999-9999")},
	}, nil)
	storage.MarkBackupCodeUsedMock.Expect(minimock.AnyContext, "code-2").Return(nil)

	err := svc.VerifyBackupCode(t.Context(), "user-1", code)
	assert.NilError(t, err)
}

func TestVerifyBackupCode_InvalidCode(t *testing.T) {
	svc, storage, _ := newBackupCodeService(t)

	storage.GetUnusedBackupCodeHashesMock.Return([]models.BackupCodeRow{
		{ID: "code-1", CodeHash: hashCode(t, "1111-1111")},
	}, nil)
	storage.MarkBackupCodeUsedMock.Optional()

	err := svc.VerifyBackupCode(t.Context(), "user-1", "9999-0000")
	assert.Assert(t, errors.Is(err, twofaService.ErrInvalidBackupCode))
}

func TestVerifyBackupCode_NoCodes(t *testing.T) {
	svc, storage, _ := newBackupCodeService(t)

	storage.GetUnusedBackupCodeHashesMock.Return(nil, nil)
	storage.MarkBackupCodeUsedMock.Optional()

	err := svc.VerifyBackupCode(t.Context(), "user-1", "1234-5678")
	assert.Assert(t, errors.Is(err, twofaService.ErrInvalidBackupCode))
}

func TestVerifyBackupCode_StorageError(t *testing.T) {
	svc, storage, _ := newBackupCodeService(t)

	storage.GetUnusedBackupCodeHashesMock.Return(nil, errors.New("db down"))
	storage.MarkBackupCodeUsedMock.Optional()

	err := svc.VerifyBackupCode(t.Context(), "user-1", "1234-5678")
	assert.Assert(t, err != nil)
	assert.Assert(t, !errors.Is(err, twofaService.ErrInvalidBackupCode))
}

func TestVerify_BackupCodeIntegration(t *testing.T) {
	svc, storage, sessionStorage := newBackupCodeService(t)

	code := "5555-6666"

	storage.GetTwoFARecordMock.Return(
		&models.TwoFARecord{UserID: "user-1", IsEnabled: true}, nil,
	)
	sessionStorage.IncrementRateLimitMock.Return(1, nil)

	storage.GetUnusedBackupCodeHashesMock.Return([]models.BackupCodeRow{
		{ID: "bc-1", CodeHash: hashCode(t, code)},
	}, nil)
	storage.MarkBackupCodeUsedMock.Expect(minimock.AnyContext, "bc-1").Return(nil)

	valid, isNewlyEnabled, err := svc.Verify(t.Context(), "user-1", code)

	assert.NilError(t, err)
	assert.Assert(t, valid, "backup code should be valid")
	assert.Assert(t, !isNewlyEnabled, "backup code should not trigger newly enabled")
}
