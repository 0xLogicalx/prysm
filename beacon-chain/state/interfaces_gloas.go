package state

import (
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
)

type WriteOnlyGloasFields interface {
	SetExecutionPayloadHeader(h interfaces.ROExecutionPayloadHeaderGloas) error
	SetBuilderPendingPayment(slot primitives.Slot, payment *ethpb.BuilderPendingPayment) error
	SetLatestBlockHash(hash [32]byte) error
	SetExecutionPayloadAvailability(index primitives.Slot, available bool) error
	SetBuilderPendingPayments(payments []*ethpb.BuilderPendingPayment) error
	AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error
}

type ReadOnlyGloasFields interface {
	LatestBlockHash() ([32]byte, error)
	BuilderPendingPayment(slot primitives.Slot) (*ethpb.BuilderPendingPayment, error)
	BuilderPendingPayments() ([]*ethpb.BuilderPendingPayment, error)
	BuilderPendingWithdrawals() ([]*ethpb.BuilderPendingWithdrawal, error)
	ExecutionHeader() (interfaces.ROExecutionPayloadHeaderGloas, error)
	LatestWithdrawalsRoot() ([32]byte, error)
}
