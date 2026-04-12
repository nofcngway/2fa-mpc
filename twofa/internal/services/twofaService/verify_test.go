package twofaService_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/gojuno/minimock/v3"

	"github.com/vbncursed/vkr/twofa/internal/crypto/shamir"
	"github.com/vbncursed/vkr/twofa/internal/crypto/totp"
	"github.com/vbncursed/vkr/twofa/internal/models"
	"github.com/vbncursed/vkr/twofa/internal/pb/mpc_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService/mocks"
	"google.golang.org/grpc"
)

// verifySuite holds shared setup for Verify tests.
type verifySuite struct {
	mc             *minimock.Controller
	storage        *mocks.StorageMock
	sessionStorage *mocks.SessionStorageMock
	mpcClients     []*mocks.MPCClientMock
	service        *twofaService.TwoFAService
}

func newVerifySuite(t *testing.T) *verifySuite {
	t.Helper()
	mc := minimock.NewController(t)
	storage := mocks.NewStorageMock(mc)
	sessionStorage := mocks.NewSessionStorageMock(mc)

	mpcClients := make([]*mocks.MPCClientMock, 3)
	mpcInterfaces := make([]twofaService.MPCClient, 3)
	for i := 0; i < 3; i++ {
		mpcClients[i] = mocks.NewMPCClientMock(mc)
		mpcInterfaces[i] = mpcClients[i]
	}

	service := twofaService.NewTwoFAService(
		storage, sessionStorage, mpcInterfaces, "test-secret", 5*time.Second,
	)

	return &verifySuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		mpcClients:     mpcClients,
		service:        service,
	}
}

// testSecret and testShareData are helpers for consistent test data.
var testSecret = []byte("12345678901234567890")

// makeValidCode generates a valid OTP code for the test secret at the given time.
func makeValidCode(unixTime int64) string {
	return totp.GenerateOTP(testSecret, unixTime)
}

// setupMPCReturnsShares configures all 3 MPC clients to return valid shares.
// Each node returns the share data for its own index (node 0 -> index 1, etc.)
// so that any 2-of-3 combination reconstructs the secret correctly.
func (vs *verifySuite) setupMPCReturnsShares(allShareData [3][]byte) {
	for i := 0; i < 3; i++ {
		data := allShareData[i]
		vs.mpcClients[i].RetrieveShareMock.Set(func(_ context.Context, req *mpc_api.RetrieveShareRequest, _ ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error) {
			return &mpc_api.RetrieveShareResponse{ShareData: data}, nil
		})
	}
}

func TestVerify_Success(t *testing.T) {
	vs := newVerifySuite(t)

	// Use real Shamir split to get valid shares
	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})
	vs.setupMPCReturnsShares(shamirPkg)
	vs.sessionStorage.GetUsedOTPCounterMock.Expect(minimock.AnyContext, "test-user").Return(0, nil)
	vs.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return nil
	})

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	valid, isNewlyEnabled, err := vs.service.Verify(context.Background(), "test-user", code)

	assert.NilError(t, err)
	assert.Assert(t, valid, "should be valid")
	assert.Assert(t, !isNewlyEnabled, "should not be newly enabled (already enabled)")
}

func TestVerify_EnablesOnFirst(t *testing.T) {
	vs := newVerifySuite(t)

	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: false}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})
	vs.setupMPCReturnsShares(shamirPkg)
	vs.sessionStorage.GetUsedOTPCounterMock.Expect(minimock.AnyContext, "test-user").Return(0, nil)
	vs.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return nil
	})
	vs.storage.EnableTwoFAMock.Expect(minimock.AnyContext, "test-user").Return(nil)

	// Make remaining mocks optional
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	valid, isNewlyEnabled, err := vs.service.Verify(context.Background(), "test-user", code)

	assert.NilError(t, err)
	assert.Assert(t, valid, "should be valid")
	assert.Assert(t, isNewlyEnabled, "should be newly enabled on first verify")
	assert.Assert(t, vs.storage.EnableTwoFAAfterCounter() >= 1, "EnableTwoFA should be called")
}

func TestVerify_RateLimit_Exceeded(t *testing.T) {
	vs := newVerifySuite(t)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 6, nil // Exceeds 5
	})

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetUsedOTPCounterMock.Optional()
	vs.sessionStorage.SetUsedOTPCounterMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	_, _, err := vs.service.Verify(context.Background(), "test-user", "123456")

	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, twofaService.ErrRateLimitExceeded),
		"expected ErrRateLimitExceeded, got: %v", err)

	// Verify NO MPC calls were made
	for i := 0; i < 3; i++ {
		assert.Equal(t, vs.mpcClients[i].RetrieveShareAfterCounter(), uint64(0),
			"no MPC calls should be made when rate limited")
	}
}

func TestVerify_RateLimit_RedisDown(t *testing.T) {
	vs := newVerifySuite(t)

	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	// Redis down - IncrementRateLimit returns error
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 0, errors.New("connection refused")
	})
	vs.setupMPCReturnsShares(shamirPkg)
	vs.sessionStorage.GetUsedOTPCounterMock.Set(func(_ context.Context, _ string) (int64, error) {
		return 0, errors.New("connection refused")
	})
	vs.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return errors.New("connection refused")
	})

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	valid, _, err := vs.service.Verify(context.Background(), "test-user", code)

	assert.NilError(t, err)
	assert.Assert(t, valid, "should still verify when Redis is down (D-07)")
}

func TestVerify_OTPReuse(t *testing.T) {
	vs := newVerifySuite(t)

	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)
	// The counter that ValidateOTPWithCounter would match
	expectedCounter := now / 30

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})
	vs.setupMPCReturnsShares(shamirPkg)
	// Return the same counter as what would match -- OTP reuse
	vs.sessionStorage.GetUsedOTPCounterMock.Expect(minimock.AnyContext, "test-user").Return(expectedCounter, nil)

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.SetUsedOTPCounterMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	_, _, err := vs.service.Verify(context.Background(), "test-user", code)

	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, twofaService.ErrOTPReused),
		"expected ErrOTPReused, got: %v", err)
}

func TestVerify_InvalidOTP(t *testing.T) {
	vs := newVerifySuite(t)

	shamirPkg := shamirSplit(t, testSecret)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})
	vs.setupMPCReturnsShares(shamirPkg)
	vs.sessionStorage.GetUsedOTPCounterMock.Expect(minimock.AnyContext, "test-user").Return(0, nil)

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.SetUsedOTPCounterMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	valid, isNewlyEnabled, err := vs.service.Verify(context.Background(), "test-user", "000000")

	assert.NilError(t, err)
	assert.Assert(t, !valid, "should be invalid for wrong OTP")
	assert.Assert(t, !isNewlyEnabled)
}

func TestVerify_InsufficientShares(t *testing.T) {
	vs := newVerifySuite(t)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})

	// Only 1 of 3 MPC clients succeeds
	vs.mpcClients[0].RetrieveShareMock.Set(func(_ context.Context, _ *mpc_api.RetrieveShareRequest, _ ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error) {
		return &mpc_api.RetrieveShareResponse{ShareData: []byte("data")}, nil
	})
	vs.mpcClients[1].RetrieveShareMock.Set(func(_ context.Context, _ *mpc_api.RetrieveShareRequest, _ ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error) {
		return nil, errors.New("node 1 unreachable")
	})
	vs.mpcClients[2].RetrieveShareMock.Set(func(_ context.Context, _ *mpc_api.RetrieveShareRequest, _ ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error) {
		return nil, errors.New("node 2 unreachable")
	})

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetUsedOTPCounterMock.Optional()
	vs.sessionStorage.SetUsedOTPCounterMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	_, _, err := vs.service.Verify(context.Background(), "test-user", "123456")

	assert.Assert(t, err != nil, "expected error with insufficient shares")
	assert.Assert(t, errors.Is(err, twofaService.ErrInsufficientShares),
		"expected ErrInsufficientShares, got: %v", err)
}

func TestVerify_NoRecord(t *testing.T) {
	vs := newVerifySuite(t)

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(nil, nil)

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.IncrementRateLimitMock.Optional()
	vs.sessionStorage.GetUsedOTPCounterMock.Optional()
	vs.sessionStorage.SetUsedOTPCounterMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	_, _, err := vs.service.Verify(context.Background(), "test-user", "123456")

	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, twofaService.ErrNotSetUp),
		"expected ErrNotSetUp, got: %v", err)

	// No MPC calls should be made
	for i := 0; i < 3; i++ {
		assert.Equal(t, vs.mpcClients[i].RetrieveShareAfterCounter(), uint64(0),
			"no MPC calls should be made when no 2FA record")
	}
}

func TestVerify_Zeroization(t *testing.T) {
	vs := newVerifySuite(t)

	shamirPkg := shamirSplit(t, testSecret)
	now := time.Now().Unix()
	code := makeValidCode(now)

	// Create copies of share data that the mocks will return.
	// We keep references to check zeroization after Verify returns.
	var capturedShares [3][]byte
	for i := 0; i < 3; i++ {
		capturedShares[i] = make([]byte, len(shamirPkg[i]))
		copy(capturedShares[i], shamirPkg[i])
	}

	for i := 0; i < 3; i++ {
		data := capturedShares[i]
		vs.mpcClients[i].RetrieveShareMock.Set(func(_ context.Context, _ *mpc_api.RetrieveShareRequest, _ ...grpc.CallOption) (*mpc_api.RetrieveShareResponse, error) {
			return &mpc_api.RetrieveShareResponse{ShareData: data}, nil
		})
	}

	vs.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "test-user").Return(
		&models.TwoFARecord{UserID: "test-user", IsEnabled: true}, nil,
	)
	vs.sessionStorage.IncrementRateLimitMock.Set(func(_ context.Context, _ string, _ time.Duration) (int64, error) {
		return 1, nil
	})
	vs.sessionStorage.GetUsedOTPCounterMock.Expect(minimock.AnyContext, "test-user").Return(0, nil)
	vs.sessionStorage.SetUsedOTPCounterMock.Set(func(_ context.Context, _ string, _ int64, _ time.Duration) error {
		return nil
	})

	// Make remaining mocks optional
	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
	vs.sessionStorage.DeleteKeysMock.Optional()

	valid, _, err := vs.service.Verify(context.Background(), "test-user", code)

	assert.NilError(t, err)
	assert.Assert(t, valid)

	// Check that the share data slices used by retrieveShares were zeroized.
	// At least the 2 shares that were used should be zeroed.
	zeroedCount := 0
	for idx := 0; idx < 3; idx++ {
		allZero := true
		for _, b := range capturedShares[idx] {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			zeroedCount++
		}
	}
	assert.Assert(t, zeroedCount >= 2, "at least 2 share data slices should be zeroed, got %d", zeroedCount)
}

// shamirSplit creates real Shamir shares for the given secret using the crypto package.
// Returns the raw share data for each of the 3 shares (index 1, 2, 3).
func shamirSplit(t *testing.T, secret []byte) [3][]byte {
	t.Helper()
	shamirShares, err := shamir.Split(secret, 3, 2)
	if err != nil {
		t.Fatalf("shamirSplit: %v", err)
	}
	var result [3][]byte
	for i := 0; i < 3; i++ {
		result[i] = shamirShares[i].Data
	}
	return result
}
