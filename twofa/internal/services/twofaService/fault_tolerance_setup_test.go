package twofaService_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"
)

// TestFaultTolerance_Setup_OneNodeDown_AllOrNothing asserts that Setup is
// strictly atomic across all 3 MPC nodes: a single failure aborts the
// operation and a compensating DeleteShare is invoked on every node. The
// twofa_record and backup codes must NOT be persisted, so the system never
// retains a half-written share set that could be reconstructed without the
// user's knowledge.
func TestFaultTolerance_Setup_OneNodeDown_AllOrNothing(t *testing.T) {
	for downNode := range 3 {
		t.Run("nodeDown="+nodeName(downNode), func(t *testing.T) {
			s := newSetupSuite(t)
			s.storage.GetTwoFARecordMock.Expect(minimock.AnyContext, "user-fault-setup").
				Return(nil, nil)

			for i := range 3 {
				if i == downNode {
					s.mpcClients[i].StoreShareMock.Set(
						func(_ context.Context, _ string, _ int, _ []byte) error {
							return errors.New("node down")
						},
					)
				} else {
					s.mpcClients[i].StoreShareMock.Set(
						func(_ context.Context, _ string, _ int, _ []byte) error { return nil },
					)
				}
				s.mpcClients[i].DeleteShareMock.Set(
					func(_ context.Context, _ string) error { return nil },
				)
			}

			s.storage.CreateTwoFARecordMock.Optional()
			s.storage.StoreBatchBackupCodesMock.Optional()

			_, _, err := s.service.Setup(t.Context(), "user-fault-setup", "u@example.com")
			assert.Assert(t, err != nil, "Setup must fail when any node rejects the share")

			// All 3 nodes must receive a compensating DeleteShare — including
			// the failed node (best-effort, idempotent cleanup).
			for i := range 3 {
				assert.Assert(t, s.mpcClients[i].DeleteShareAfterCounter() >= 1,
					"compensating DeleteShare missing on node %d", i)
			}

			// twofa_record + backup codes must not be persisted.
			assert.Equal(t, s.storage.CreateTwoFARecordAfterCounter(), uint64(0),
				"twofa_record must not be created after share-distribution failure")
			assert.Equal(t, s.storage.StoreBatchBackupCodesAfterCounter(), uint64(0),
				"backup codes must not be stored after share-distribution failure")
		})
	}
}
