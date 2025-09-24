package state_native

import (
	"slices"

	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
)

// ExecutionPayloadAvailability is a non-mutating call to the beacon state which returns the
// execution payload availability.
func (b *BeaconState) ExecutionPayloadAvailability() []byte {
	if b.version < version.Gloas {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return slices.Clone(b.executionPayloadAvailability)
}

// LatestBlockHash is a non-mutating call to the beacon state which returns the
// latest block hash.
func (b *BeaconState) LatestBlockHash() [32]byte {
	if b.version < version.Gloas {
		return [32]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	var hash [32]byte

	copy(hash[:], b.latestBlockHash)
	return hash
}

// LatestWithdrawalsRoot is a non-mutating call to the beacon state which returns the
// latest withdrawals root.
func (b *BeaconState) LatestWithdrawalsRoot() [32]byte {
	if b.version < version.Gloas {
		return [32]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	var root [32]byte
	copy(root[:], b.latestWithdrawalsRoot)
	return root
}

// BuilderPendingPayments is a non-mutating call to the beacon state which returns the
// builder pending payments.
func (b *BeaconState) BuilderPendingPayments() []*ethpb.BuilderPendingPayment {
	if b.version < version.Gloas {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.builderPendingPaymentsVal()
}

// BuilderPendingWithdrawals is a non-mutating call to the beacon state which returns the
// builder pending withdrawals.
func (b *BeaconState) BuilderPendingWithdrawals() []*ethpb.BuilderPendingWithdrawal {
	if b.version < version.Gloas {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.builderPendingWithdrawalsVal()
}

// ExecutionPayloadBid is a non-mutating call to the beacon state which returns the
// execution payload bid.
func (b *BeaconState) ExecutionPayloadBid() *ethpb.ExecutionPayloadBid {
	if b.version < version.Gloas {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.executionPayloadBidVal()
}
