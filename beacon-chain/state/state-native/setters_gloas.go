package state_native

import (
	"errors"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state/stateutil"
	"github.com/OffchainLabs/prysm/v6/config/params"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
)

// RotateBuilderPendingPayments rotates builder pending payments by removing n from the beginning
// and adding n new empty payments at the end.
// This implements: state.builder_pending_payments = state.builder_pending_payments[n:] + [BuilderPendingPayment() for _ in range(n)]
func (b *BeaconState) RotateBuilderPendingPayments(n uint64) error {
	if b.version < version.Gloas {
		return errNotSupported("RotateBuilderPendingPayments", b.version)
	}

	if n > uint64(len(b.builderPendingPayments)) {
		return errors.New("cannot rotate more payments than are in the queue")
	}

	if n == 0 {
		return nil
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	// Create new slice with remaining payments + n empty payments
	oldPayments := b.builderPendingPayments
	newPayments := make([]*ethpb.BuilderPendingPayment, 2*params.BeaconConfig().SlotsPerEpoch)

	// Copy remaining payments (skip first n)
	copy(newPayments, oldPayments[n:])

	// Add n empty payments at the end
	for i := uint64(len(oldPayments)) - n; i < uint64(len(newPayments)); i++ {
		newPayments[i] = &ethpb.BuilderPendingPayment{
			Weight: 0,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      make([]byte, 20),
				Amount:            0,
				BuilderIndex:      0,
				WithdrawableEpoch: 0,
			},
		}
	}

	// Handle shared field references
	if b.sharedFieldReferences[types.BuilderPendingPayments].Refs() > 1 {
		b.sharedFieldReferences[types.BuilderPendingPayments].MinusRef()
		b.sharedFieldReferences[types.BuilderPendingPayments] = stateutil.NewRef(1)
	}

	b.builderPendingPayments = newPayments
	b.markFieldAsDirty(types.BuilderPendingPayments)
	b.rebuildTrie[types.BuilderPendingPayments] = true
	return nil
}

// AppendBuilderPendingWithdrawal appends a builder pending withdrawal to the beacon state.
func (b *BeaconState) AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error {
	if b.version < version.Gloas {
		return errNotSupported("AppendBuilderPendingWithdrawal", b.version)
	}

	if withdrawal == nil {
		return errors.New("cannot append nil builder pending withdrawal")
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	withdrawals := b.builderPendingWithdrawals
	if b.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs() > 1 {
		withdrawals = make([]*ethpb.BuilderPendingWithdrawal, 0, len(b.builderPendingWithdrawals)+1)
		withdrawals = append(withdrawals, b.builderPendingWithdrawals...)
		b.sharedFieldReferences[types.BuilderPendingWithdrawals].MinusRef()
		b.sharedFieldReferences[types.BuilderPendingWithdrawals] = stateutil.NewRef(1)
	}

	b.builderPendingWithdrawals = append(withdrawals, withdrawal)
	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	return nil
}
