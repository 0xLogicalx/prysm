package state

import (
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
)

// ReadOnlyBuilderPendingPayments defines methods for reading builder pending payments from the beacon state.
type ReadOnlyBuilderPendingPayments interface {
	BuilderPendingPayments() ([]*ethpb.BuilderPendingPayment, error)
}

// WriteOnlyBuilderPendingPayments defines methods for writing builder pending payments to the beacon state.
type WriteOnlyBuilderPendingPayments interface {
	RotateBuilderPendingPayments(uint64) error
}

// WriteOnlyBuilderPendingWithdrawals defines methods for writing builder pending withdrawals to the beacon state.
type WriteOnlyBuilderPendingWithdrawals interface {
	AppendBuilderPendingWithdrawal(*ethpb.BuilderPendingWithdrawal) error
}
