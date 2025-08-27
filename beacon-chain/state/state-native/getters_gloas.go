package state_native

import (
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
)

// LatestWithdrawalsRoot returns the root of the latest withdrawals.
func (b *BeaconState) LatestWithdrawalsRoot() ([32]byte, error) {
	if b.version < version.Gloas {
		return [32]byte{}, errNotSupported("LatestWithdrawalRoot", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.latestWithdrawalsRoot == nil {
		return [32]byte{}, nil
	}

	return [32]byte(b.latestWithdrawalsRoot), nil
}

// LatestBlockHash returns the hash of the latest execution block.
func (b *BeaconState) LatestBlockHash() ([32]byte, error) {
	if b.version < version.Gloas {
		return [32]byte{}, errNotSupported("LatestBlockHash", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.latestBlockHash == nil {
		return [32]byte{}, nil
	}

	return [32]byte(b.latestBlockHash), nil
}

// BuilderPendingPayment returns a specific builder pending payment for the given slot.
func (b *BeaconState) BuilderPendingPayment(slot primitives.Slot) (*ethpb.BuilderPendingPayment, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("BuilderPendingPayment", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	paymentIndex := slotsPerEpoch + (slot % slotsPerEpoch)

	return ethpb.CopyBuilderPendingPayment(b.builderPendingPayments[paymentIndex]), nil
}

// BuilderPendingPayments returns the builder pending payments.
func (b *BeaconState) BuilderPendingPayments() ([]*ethpb.BuilderPendingPayment, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("BuilderPendingPayments", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.builderPendingPayments == nil {
		return make([]*ethpb.BuilderPendingPayment, 0), nil
	}

	return ethpb.CopyBuilderPendingPaymentSlice(b.builderPendingPayments), nil
}

// BuilderPendingWithdrawals returns the builder pending withdrawals.
func (b *BeaconState) BuilderPendingWithdrawals() ([]*ethpb.BuilderPendingWithdrawal, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("BuilderPendingWithdrawals", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.builderPendingWithdrawals == nil {
		return make([]*ethpb.BuilderPendingWithdrawal, 0), nil
	}

	return ethpb.CopyBuilderPendingWithdrawalSlice(b.builderPendingWithdrawals), nil
}

// ExecutionHeader returns the execution payload header.
func (b *BeaconState) ExecutionHeader() (interfaces.ROExecutionPayloadHeaderGloas, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("ExecutionHeader", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return blocks.WrappedROExecutionPayloadHeaderGloas(b.executionPayloadHeader.Copy())
}
