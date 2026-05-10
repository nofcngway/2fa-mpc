// Package twofaService_test contains explicit fault-tolerance tests for the
// 2-of-3 Shamir threshold scheme used by the TwoFA service.
//
// These tests document and verify the threshold guarantees of the system:
//
//   - Verify/Disable read path: 2-of-3 nodes responding is sufficient (the
//     remaining node may be down, slow, or unreachable).
//   - Setup write path: all 3 nodes must accept a share. A single failure
//     triggers a compensating delete on every node and surfaces an error to
//     the caller (atomic all-or-nothing).
//   - Disable cleanup path: all 3 nodes must accept the delete. A single
//     failure leaves the record intact (the user can retry).
//
// Tests reuse the existing per-flow suites (verifySuite, setupSuite,
// disableSuite). The only fault-tolerance-specific helper here is a short-
// timeout suite used by the Verify timeout scenarios — those need a sub-second
// mpcTimeout to complete quickly.
package twofaService_test

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService/mocks"
)

// newVerifySuiteWithTimeout returns a verifySuite-shaped harness configured
// with the given mpcTimeout. Use it for fault-tolerance tests that need to
// observe the per-call MPC timeout in real time without slowing the suite.
func newVerifySuiteWithTimeout(t *testing.T, mpcTimeout time.Duration) *verifySuite {
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

	service, err := twofaService.NewTwoFAService(twofaService.Deps{
		Storage:        storage,
		SessionStorage: sessionStorage,
		MPCClients:     mpcInterfaces,
		EventProducer:  eventProducer,
		MPCTimeout:     mpcTimeout,
	})
	assert.NilError(t, err, "failed to create TwoFA service")

	return &verifySuite{
		mc:             mc,
		storage:        storage,
		sessionStorage: sessionStorage,
		mpcClients:     mpcClients,
		service:        service,
	}
}

// nodeName returns a stable subtest name so failures reference a specific node
// index without being mistaken for a 1-based ordinal.
func nodeName(idx int) string {
	switch idx {
	case 0:
		return "node0"
	case 1:
		return "node1"
	case 2:
		return "node2"
	default:
		return "nodeN"
	}
}
