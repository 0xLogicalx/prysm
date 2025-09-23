package state

import (
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
)

type writeOnlyGloasFields interface {
	SetBuilderPendingPayment(index uint64, payment *ethpb.BuilderPendingPayment) error
}

type readOnlyGloasFields interface {
	BuilderPendingPayment(index uint64) (*ethpb.BuilderPendingPayment, error)
	ExecutionPayloadAvailability(slot primitives.Slot) (uint64, error)
}
