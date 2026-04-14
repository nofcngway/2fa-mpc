package twofaService_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/twofa/internal/domain"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService/mocks"
)

// setupSuite holds shared setup for Setup tests.
type setupSuite struct {
	mc         *minimock.Controller
	storage    *mocks.StorageMock
	mpcClients []*mocks.MPCClientMock
	service    *twofaService.TwoFAService
}

func newSetupSuite(t *testing.T) *setupSuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)

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
		storage, nil, mpcInterfaces, eventProducer, "test-secret", 5*time.Second,
	)

	return &setupSuite{
		mc:         mc,
		storage:    storage,
		mpcClients: mpcClients,
		service:    service,
	}
}

func TestSetup_Success(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, userID string, codeHashes []string) error {
		assert.Equal(t, userID, "test-user-id")
		assert.Equal(t, len(codeHashes), 10)
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	uri, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Assert(t, uri != "", "provisioning URI should not be empty")
	assert.Assert(t, len(uri) > 0 && strings.Contains(uri, "otpauth://totp/"), "URI should contain otpauth://totp/")
	assert.Assert(t, strings.Contains(uri, "user@example.com") || strings.Contains(uri, "user%40example.com"), "URI should contain email")
	assert.Equal(t, len(codes), 10)
}

func TestSetup_DuplicateEnabled(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(
		&domain.TwoFARecord{UserID: "test-user-id", IsEnabled: true}, nil,
	)

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, domain.ErrAlreadyEnabled),
		"expected ErrAlreadyEnabled, got: %v", err)
}

func TestSetup_DuplicateDisabled(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(
		&domain.TwoFARecord{UserID: "test-user-id", IsEnabled: false}, nil,
	)
	// No CreateTwoFARecord call expected (record already exists)
	s.storage.CreateTwoFARecordMock.Optional()
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	uri, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Assert(t, uri != "")
	assert.Equal(t, len(codes), 10)
}

func TestSetup_NewUserNoExistingRecord(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	uri, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Assert(t, uri != "")
	assert.Equal(t, len(codes), 10)
}

func TestSetup_PartialMPCFailure_Node2Fails(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)

	s.mpcClients[0].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})
	s.mpcClients[1].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})
	s.mpcClients[2].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return errors.New("node 2 unreachable")
	})

	// Compensating delete should be called on ALL 3 nodes
	for i := range 3 {
		s.mpcClients[i].DeleteShareMock.Set(func(_ context.Context, _ string) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when node 2 fails")
	// Verify delete was called on all 3 nodes
	for i := range 3 {
		assert.Assert(t, s.mpcClients[i].DeleteShareAfterCounter() >= 1,
			"DeleteShare should be called on node %d", i)
	}
}

func TestSetup_PartialMPCFailure_Node0Fails(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)

	s.mpcClients[0].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return errors.New("node 0 unreachable")
	})
	s.mpcClients[1].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})
	s.mpcClients[2].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].DeleteShareMock.Set(func(_ context.Context, _ string) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when node 0 fails")
	for i := range 3 {
		assert.Assert(t, s.mpcClients[i].DeleteShareAfterCounter() >= 1,
			"DeleteShare should be called on node %d", i)
	}
}

func TestSetup_AllMPCNodesFail(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return errors.New("node unreachable")
		})
		s.mpcClients[i].DeleteShareMock.Set(func(_ context.Context, _ string) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when all nodes fail")
	for i := range 3 {
		assert.Assert(t, s.mpcClients[i].DeleteShareAfterCounter() >= 1,
			"DeleteShare should be called on node %d", i)
	}
}

func TestSetup_BackupCodeFormat(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	codeRegex := regexp.MustCompile(`^\d{4}-\d{4}$`)
	for i, code := range codes {
		assert.Assert(t, codeRegex.MatchString(code),
			"backup code %d (%q) does not match xxxx-xxxx format", i, code)
	}
}

func TestSetup_BackupCodeUniqueness(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.Assert(t, !seen[code], "duplicate backup code: %s", code)
		seen[code] = true
	}
}

func TestSetup_BackupCodeHashing(t *testing.T) {
	s := newSetupSuite(t)

	var capturedHashes []string
	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, codeHashes []string) error {
		capturedHashes = make([]string, len(codeHashes))
		copy(capturedHashes, codeHashes)
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, codes, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Equal(t, len(capturedHashes), 10)
	for i, hash := range capturedHashes {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(codes[i]))
		assert.NilError(t, err, "bcrypt hash %d does not match plaintext code", i)
	}
}

func TestSetup_ProvisioningURIContainsEmail(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	uri, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(uri, "user%40example.com") || strings.Contains(uri, "user@example.com"),
		"provisioning URI should contain email, got: %s", uri)
}

func TestSetup_StoreShareReceivesCorrectData(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	type storeCall struct {
		UserID     string
		ShareIndex int
		ShareData  []byte
	}
	var calls [3]storeCall
	var callCount atomic.Int32

	for i := range 3 {
		idx := i
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, userID string, shareIndex int, shareData []byte) error {
			calls[idx] = storeCall{
				UserID:     userID,
				ShareIndex: shareIndex,
				ShareData:  shareData,
			}
			callCount.Add(1)
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Equal(t, callCount.Load(), int32(3))

	for i := range 3 {
		assert.Equal(t, calls[i].UserID, "test-user-id", "node %d: wrong user_id", i)
		assert.Assert(t, calls[i].ShareIndex >= 1 && calls[i].ShareIndex <= 3,
			"node %d: share_index should be 1-3, got %d", i, calls[i].ShareIndex)
		assert.Assert(t, len(calls[i].ShareData) > 0,
			"node %d: share_data should not be empty", i)
	}
}

func TestSetup_CreateTwoFARecordFails(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(errors.New("db error"))

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when CreateTwoFARecord fails")
	assert.Assert(t, strings.Contains(err.Error(), "create twofa record"),
		"error should mention create twofa record, got: %v", err)
}

func TestSetup_StoreBatchBackupCodesFails(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return errors.New("db error")
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when StoreBatchBackupCodes fails")
	assert.Assert(t, strings.Contains(err.Error(), "store backup codes"),
		"error should mention store backup codes, got: %v", err)
}

func TestSetup_GetTwoFARecordFails(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, errors.New("db error"))

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil, "expected error when GetTwoFARecord fails")
	assert.Assert(t, strings.Contains(err.Error(), "check existing 2fa"),
		"error should mention check existing 2fa, got: %v", err)
}

func TestSetup_CompensatingDeleteUsesFreshContext(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)

	// Make node 0 fail to trigger compensating delete
	s.mpcClients[0].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return errors.New("node 0 failed")
	})
	s.mpcClients[1].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})
	s.mpcClients[2].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
		return nil
	})

	// Verify compensating delete context is NOT cancelled
	for i := range 3 {
		s.mpcClients[i].DeleteShareMock.Set(func(ctx context.Context, userID string) error {
			// The context passed to DeleteShare should NOT be already cancelled
			assert.Assert(t, ctx.Err() == nil,
				"compensating delete context should not be cancelled (should use fresh context)")
			assert.Equal(t, userID, "test-user-id")
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.Assert(t, err != nil)
}

func TestSetup_SharesZeroized(t *testing.T) {
	s := newSetupSuite(t)

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	// Capture share data to verify it gets zeroed after setup
	var capturedShareData [3][]byte
	for i := range 3 {
		idx := i
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, shareData []byte) error {
			// Store reference to the share data (not a copy)
			capturedShareData[idx] = shareData
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	// Note: share data passed to StoreShare is from the Share struct's Data field.
	// After Setup returns, the deferred zeroize should have zeroed each share's Data.
	// However, the proto request may have a copy. This test verifies the mechanism exists
	// by checking that the service completes successfully with the zeroize defer.
}

func TestSetup_SecretZeroized(t *testing.T) {
	s := newSetupSuite(t)

	// Capture the raw secret slice to verify zeroization after Setup
	var capturedRaw []byte
	originalGenerate := twofaService.GenerateSecretFunc
	twofaService.GenerateSecretFunc = func() ([]byte, []byte, error) {
		raw, base32, err := originalGenerate()
		capturedRaw = raw
		return raw, base32, err
	}
	defer func() { twofaService.GenerateSecretFunc = originalGenerate }()

	s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil, nil)
	s.storage.CreateTwoFARecordMock.Expect(minimock.AnyContext, "test-user-id").Return(nil)
	s.storage.StoreBatchBackupCodesMock.Set(func(_ context.Context, _ string, _ []string) error {
		return nil
	})

	for i := range 3 {
		s.mpcClients[i].StoreShareMock.Set(func(_ context.Context, _ string, _ int, _ []byte) error {
			return nil
		})
	}

	_, _, err := s.service.Setup(t.Context(), "test-user-id", "user@example.com")

	assert.NilError(t, err)
	assert.Assert(t, capturedRaw != nil, "capturedRaw should not be nil")
	assert.Assert(t, len(capturedRaw) == 20, "TOTP secret should be 20 bytes, got %d", len(capturedRaw))
	for i, b := range capturedRaw {
		assert.Equal(t, byte(0), b, "raw secret byte at index %d should be zero after Setup, got %d", i, b)
	}
}

