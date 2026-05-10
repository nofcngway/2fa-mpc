package twofaService_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/twofa/internal/domain"
)

// armVerifyFaultMocks wires up storage/session mocks so an individual fault
// scenario only needs to configure the MPC clients.
func armVerifyFaultMocks(vs *verifySuite, userID string) {
	vs.storage.GetTwoFARecordMock.Set(func(_ context.Context, _ string) (*domain.TwoFARecord, error) {
		return &domain.TwoFARecord{UserID: userID, IsEnabled: true}, nil
	})
	vs.sessionStorage.IncrementRateLimitMock.Set(
		func(_ context.Context, _ string, _ time.Duration) (int64, error) { return 1, nil },
	)
	vs.sessionStorage.GetUsedOTPCounterMock.Optional().Return(0, domain.ErrCounterNotFound)
	vs.sessionStorage.SetUsedOTPCounterMock.Optional().Return(nil)
	vs.sessionStorage.DeleteKeysMock.Optional().Return(nil)

	vs.storage.EnableTwoFAMock.Optional()
	vs.storage.CreateTwoFARecordMock.Optional()
	vs.storage.StoreBatchBackupCodesMock.Optional()
	vs.storage.DeleteTwoFARecordMock.Optional()
	vs.storage.DeleteBackupCodesMock.Optional()
	vs.storage.GetUnusedBackupCodeHashesMock.Optional()
	vs.storage.MarkBackupCodeUsedMock.Optional()
	vs.sessionStorage.GetRateLimitMock.Optional()
}

// TestFaultTolerance_Verify_OneNodeDown_Succeeds asserts the central guarantee
// of the system: with one MPC node permanently down, Verify still completes
// because the surviving 2 nodes meet the Shamir reconstruction threshold.
func TestFaultTolerance_Verify_OneNodeDown_Succeeds(t *testing.T) {
	for downNode := range 3 {
		t.Run("nodeDown="+nodeName(downNode), func(t *testing.T) {
			vs := newVerifySuite(t)
			armVerifyFaultMocks(vs, "user-fault-1")

			shareData := shamirSplit(t, testSecret)
			for i := range 3 {
				if i == downNode {
					vs.mpcClients[i].RetrieveShareMock.Optional().Set(
						func(_ context.Context, _ string, _ int) ([]byte, error) {
							return nil, errors.New("connection refused")
						},
					)
					continue
				}
				data := shareData[i]
				vs.mpcClients[i].RetrieveShareMock.Optional().Set(
					func(_ context.Context, _ string, _ int) ([]byte, error) {
						return data, nil
					},
				)
			}

			code := makeValidCode(time.Now().Unix())
			valid, _, err := vs.service.Verify(t.Context(), "user-fault-1", code)
			assert.NilError(t, err, "Verify should tolerate a single down node")
			assert.Assert(t, valid, "OTP must validate via the 2 surviving nodes")
		})
	}
}

// TestFaultTolerance_Verify_SlowNodeIgnored_FirstTwoWins asserts that when one
// node responds noticeably later than the other two, Verify returns as soon as
// 2 shares are in hand and the slow response is discarded.
func TestFaultTolerance_Verify_SlowNodeIgnored_FirstTwoWins(t *testing.T) {
	vs := newVerifySuite(t)
	armVerifyFaultMocks(vs, "user-fault-2")

	shareData := shamirSplit(t, testSecret)
	const slowNode = 2

	for i := range 3 {
		data := shareData[i]
		idx := i
		vs.mpcClients[i].RetrieveShareMock.Optional().Set(
			func(ctx context.Context, _ string, _ int) ([]byte, error) {
				if idx == slowNode {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(2 * time.Second):
						return data, nil
					}
				}
				return data, nil
			},
		)
	}

	start := time.Now()
	code := makeValidCode(time.Now().Unix())
	valid, _, err := vs.service.Verify(t.Context(), "user-fault-2", code)
	elapsed := time.Since(start)

	assert.NilError(t, err)
	assert.Assert(t, valid, "Verify must succeed via the 2 fast nodes")
	assert.Assert(t, elapsed < 1*time.Second,
		"first-2-wins: must not wait for the slow node. got %s", elapsed)
}

// TestFaultTolerance_Verify_OneNodeTimeout_OneNodeDown_Fails: one node hard-
// down + one node hangs past mpcTimeout. Only 1 node remains — below the
// 2-of-3 threshold — so Verify must return ErrInsufficientShares promptly.
func TestFaultTolerance_Verify_OneNodeTimeout_OneNodeDown_Fails(t *testing.T) {
	vs := newVerifySuiteWithTimeout(t, 200*time.Millisecond)
	armVerifyFaultMocks(vs, "user-fault-3")

	shareData := shamirSplit(t, testSecret)

	vs.mpcClients[0].RetrieveShareMock.Optional().Set(
		func(_ context.Context, _ string, _ int) ([]byte, error) { return shareData[0], nil },
	)
	vs.mpcClients[1].RetrieveShareMock.Optional().Set(
		func(ctx context.Context, _ string, _ int) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	)
	vs.mpcClients[2].RetrieveShareMock.Optional().Set(
		func(_ context.Context, _ string, _ int) ([]byte, error) {
			return nil, errors.New("node 2 unreachable")
		},
	)

	start := time.Now()
	_, _, err := vs.service.Verify(t.Context(), "user-fault-3", "123456")
	elapsed := time.Since(start)

	assert.Assert(t, err != nil, "Verify must fail when only 1 node responds")
	assert.Assert(t, errors.Is(err, domain.ErrInsufficientShares),
		"expected ErrInsufficientShares, got: %v", err)
	assert.Assert(t, elapsed < 2*time.Second,
		"Verify must time out promptly via mpcTimeout. got %s", elapsed)
}

// TestFaultTolerance_Verify_AllNodesTimeout_Fails asserts that when all 3 nodes
// hang, the service returns ErrInsufficientShares without leaking goroutines or
// blocking the caller forever.
func TestFaultTolerance_Verify_AllNodesTimeout_Fails(t *testing.T) {
	vs := newVerifySuiteWithTimeout(t, 200*time.Millisecond)
	armVerifyFaultMocks(vs, "user-fault-4")

	for i := range 3 {
		vs.mpcClients[i].RetrieveShareMock.Optional().Set(
			func(ctx context.Context, _ string, _ int) ([]byte, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		)
	}

	start := time.Now()
	_, _, err := vs.service.Verify(t.Context(), "user-fault-4", "123456")
	elapsed := time.Since(start)

	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, domain.ErrInsufficientShares),
		"expected ErrInsufficientShares, got: %v", err)
	assert.Assert(t, elapsed < 2*time.Second,
		"all-timeout case must return promptly. got %s", elapsed)
}
