package state_native

import (
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state/stateutil"
	fieldparams "github.com/OffchainLabs/prysm/v6/config/fieldparams"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/pkg/errors"
)

// SetExecutionPayloadAvailability is a mutating call to the beacon state which sets the
// execution payload availability.
func (b *BeaconState) SetExecutionPayloadAvailability(availability []byte) error {
	if b.version < version.Gloas {
		return errNotSupported("SetExecutionPayloadAvailability", b.version)
	}
	expectedLength := fieldparams.BlockRootsLength / 8
	if len(availability) != expectedLength {
		return errors.Errorf("invalid length for execution payload availability: got %d, expected %d", len(availability), expectedLength)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.ExecutionPayloadAvailability].MinusRef()
	b.sharedFieldReferences[types.ExecutionPayloadAvailability] = stateutil.NewRef(1)

	b.executionPayloadAvailability = availability

	b.markFieldAsDirty(types.ExecutionPayloadAvailability)
	return nil
}

// SetBuilderPendingPayments is a mutating call to the beacon state which sets the
// builder pending payments.
func (b *BeaconState) SetBuilderPendingPayments(payments []*ethpb.BuilderPendingPayment) error {
	if b.version < version.Gloas {
		return errNotSupported("SetBuilderPendingPayments", b.version)
	}
	expectedLength := 2 * fieldparams.SlotsPerEpoch
	if len(payments) != expectedLength {
		return errors.Errorf("invalid length for builder pending payments: got %d, expected %d", len(payments), expectedLength)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.BuilderPendingPayments].MinusRef()
	b.sharedFieldReferences[types.BuilderPendingPayments] = stateutil.NewRef(1)

	// Create deep copies of the payments
	b.builderPendingPayments = make([]*ethpb.BuilderPendingPayment, len(payments))
	for i, payment := range payments {
		if payment != nil {
			b.builderPendingPayments[i] = payment.Copy()
		}
	}

	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}

// SetBuilderPendingWithdrawals is a mutating call to the beacon state which sets the
// builder pending withdrawals.
func (b *BeaconState) SetBuilderPendingWithdrawals(withdrawals []*ethpb.BuilderPendingWithdrawal) error {
	if b.version < version.Gloas {
		return errNotSupported("SetBuilderPendingWithdrawals", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.BuilderPendingWithdrawals].MinusRef()
	b.sharedFieldReferences[types.BuilderPendingWithdrawals] = stateutil.NewRef(1)

	// Create deep copies of the withdrawals
	b.builderPendingWithdrawals = make([]*ethpb.BuilderPendingWithdrawal, len(withdrawals))
	for i, withdrawal := range withdrawals {
		if withdrawal != nil {
			b.builderPendingWithdrawals[i] = withdrawal.Copy()
		}
	}

	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	return nil
}

// SetLatestBlockHash is a mutating call to the beacon state which sets the
// latest block hash.
func (b *BeaconState) SetLatestBlockHash(hash [32]byte) error {
	if b.version < version.Gloas {
		return errNotSupported("SetLatestBlockHash", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.LatestBlockHash].MinusRef()
	b.sharedFieldReferences[types.LatestBlockHash] = stateutil.NewRef(1)

	b.latestBlockHash = hash[:]

	b.markFieldAsDirty(types.LatestBlockHash)
	return nil
}

// SetLatestWithdrawalsRoot is a mutating call to the beacon state which sets the
// latest withdrawals root.
func (b *BeaconState) SetLatestWithdrawalsRoot(root [32]byte) error {
	if b.version < version.Gloas {
		return errNotSupported("SetLatestWithdrawalsRoot", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.LatestWithdrawalsRoot].MinusRef()
	b.sharedFieldReferences[types.LatestWithdrawalsRoot] = stateutil.NewRef(1)

	b.latestWithdrawalsRoot = root[:]

	b.markFieldAsDirty(types.LatestWithdrawalsRoot)
	return nil
}

// SetExecutionPayloadBid is a mutating call to the beacon state which sets the
// execution payload bid.
func (b *BeaconState) SetExecutionPayloadBid(bid *ethpb.ExecutionPayloadBid) error {
	if b.version < version.Gloas {
		return errNotSupported("SetExecutionPayloadBid", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.sharedFieldReferences[types.ExecutionPayloadBid].MinusRef()
	b.sharedFieldReferences[types.ExecutionPayloadBid] = stateutil.NewRef(1)

	if bid == nil {
		b.executionPayloadbid = nil
	} else {
		b.executionPayloadbid = bid.Copy()
	}

	b.markFieldAsDirty(types.ExecutionPayloadBid)
	return nil
}