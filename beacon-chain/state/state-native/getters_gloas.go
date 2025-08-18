package state_native

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

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

// PendingPaymentSum returns the total pending payment amount for a builder.
func (b *BeaconState) PendingPaymentSum(builderIndex primitives.ValidatorIndex) (uint64, error) {
	if b.version < version.Gloas {
		return 0, errNotSupported("PendingPaymentSum", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	var total uint64
	for _, payment := range b.builderPendingPayments {
		if payment.Withdrawal.BuilderIndex == builderIndex {
			total += uint64(payment.Withdrawal.Amount)
		}
	}

	return total, nil
}

// PendingWithdrawalSum returns the total pending withdrawal amount for a builder.
func (b *BeaconState) PendingWithdrawalSum(builderIndex primitives.ValidatorIndex) (uint64, error) {
	if b.version < version.Gloas {
		return 0, errNotSupported("PendingWithdrawalSum", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	var total uint64
	for _, withdrawal := range b.builderPendingWithdrawals {
		if withdrawal.BuilderIndex == builderIndex {
			total += uint64(withdrawal.Amount)
		}
	}

	return total, nil
}

// LatestExecutionPayloadBid returns the cached latest execution payload bid for Gloas.
func (b *BeaconState) LatestExecutionPayloadBid() (interfaces.ROExecutionPayloadBid, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("LatestExecutionPayloadBid", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.latestExecutionPayloadBid == nil {
		return nil, nil
	}

	return blocks.WrappedROExecutionPayloadBid(b.latestExecutionPayloadBid.Copy())
}
