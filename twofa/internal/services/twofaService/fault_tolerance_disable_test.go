package twofaService_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/twofa/internal/domain"
)

// TestFaultTolerance_Disable_OneNodeDown_PreservesRecord asserts that Disable
// is strict: a single MPC delete failure aborts the operation and leaves the
// twofa_record + backup codes intact. The user is still considered to have
// 2FA enabled and can retry until cleanup completes (per D-13).
func TestFaultTolerance_Disable_OneNodeDown_PreservesRecord(t *testing.T) {
	for downNode := range 3 {
		t.Run("nodeDown="+nodeName(downNode), func(t *testing.T) {
			ds := newDisableSuite(t)
			ds.makeAllMocksOptional()

			ds.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "user-fault-dis").Return(
				&domain.TwoFARecord{UserID: "user-fault-dis", IsEnabled: true}, nil,
			)
			// IncrementRateLimit default (returns 1, nil) is supplied by
			// makeAllMocksOptional. Override only the OTP counter behavior.
			ds.sessionStorage.GetUsedOTPCounterMock.Set(
				func(_ context.Context, _ string) (int64, error) {
					return 0, domain.ErrCounterNotFound
				},
			)
			// SetUsedOTPCounter fires after a valid OTP — provide a return so
			// the call is satisfied even though it is conceptually optional.
			ds.sessionStorage.SetUsedOTPCounterMock.Set(
				func(_ context.Context, _ string, _ int64, _ time.Duration) error {
					return nil
				},
			)

			shareData := shamirSplit(t, testSecret)
			for i := range 3 {
				data := shareData[i]
				ds.mpcClients[i].RetrieveShareMock.Optional().Set(
					func(_ context.Context, _ string, _ int) ([]byte, error) { return data, nil },
				)

				// DeleteShare is optional on every node: errgroup cancellation
				// on the first failure may pre-empt sibling goroutines before
				// their mock is invoked.
				if i == downNode {
					ds.mpcClients[i].DeleteShareMock.Optional().Set(
						func(_ context.Context, _ string) error {
							return errors.New("node down on delete")
						},
					)
				} else {
					ds.mpcClients[i].DeleteShareMock.Optional().Set(
						func(_ context.Context, _ string) error { return nil },
					)
				}
			}

			code := makeValidCode(time.Now().Unix())
			err := ds.service.Disable(t.Context(), "user-fault-dis", code)
			assert.Assert(t, err != nil, "Disable must fail when any node refuses the delete")

			// Record + backup codes must NOT be removed: 2FA stays active for
			// the user until all 3 nodes confirm deletion.
			assert.Equal(t, ds.storage.DeleteTwoFARecordAfterCounter(), uint64(0),
				"twofa_record must not be deleted on partial MPC failure")
			assert.Equal(t, ds.storage.DeleteBackupCodesAfterCounter(), uint64(0),
				"backup codes must not be deleted on partial MPC failure")
		})
	}
}
