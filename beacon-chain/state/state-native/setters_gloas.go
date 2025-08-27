package state_native

import (
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
)

func (b *BeaconState) SetExecutionPayloadHeader(h interfaces.ROExecutionPayloadHeaderGloas) error {
	if b.version < version.Gloas {
		return errNotSupported("SetExecutionPayloadHeader", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	parentBlockHash := h.ParentBlockHash()
	parentBlockRoot := h.ParentBlockRoot()
	blockHash := h.BlockHash()
	blobKzgCommitmentsRoot := h.BlobKzgCommitmentsRoot()
	feeRecipient := h.FeeRecipient()
	b.executionPayloadHeader = &ethpb.ExecutionPayloadHeaderGloas{
		ParentBlockHash:        parentBlockHash[:],
		ParentBlockRoot:        parentBlockRoot[:],
		BlockHash:              blockHash[:],
		GasLimit:               h.GasLimit(),
		BuilderIndex:           h.BuilderIndex(),
		Slot:                   h.Slot(),
		Value:                  uint64(h.Value()),
		BlobKzgCommitmentsRoot: blobKzgCommitmentsRoot[:],
		FeeRecipient:           feeRecipient[:],
	}
	b.markFieldAsDirty(types.ExecutionPayloadHeader)

	return nil
}

// SetBuilderPendingPayment sets a builder pending payment for the specified slot.
func (b *BeaconState) SetBuilderPendingPayment(slot primitives.Slot, payment *ethpb.BuilderPendingPayment) error {
	if b.version < version.Gloas {
		return errNotSupported("SetBuilderPendingPayment", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	paymentIndex := slotsPerEpoch + (slot % slotsPerEpoch)

	b.builderPendingPayments[paymentIndex] = ethpb.CopyBuilderPendingPayment(payment)

	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}

// SetLatestBlockHash sets the latest execution block hash.
func (b *BeaconState) SetLatestBlockHash(hash [32]byte) error {
	if b.version < version.Gloas {
		return errNotSupported("SetLatestBlockHash", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestBlockHash = hash[:]
	b.markFieldAsDirty(types.LatestBlockHash)
	return nil
}

// SetExecutionPayloadAvailability sets the execution payload availability bit for a specific slot.
func (b *BeaconState) SetExecutionPayloadAvailability(index primitives.Slot, available bool) error {
	if b.version < version.Gloas {
		return errNotSupported("SetExecutionPayloadAvailability", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	bitIndex := index % params.BeaconConfig().SlotsPerHistoricalRoot
	byteIndex := bitIndex / 8
	bitPosition := bitIndex % 8

	// Set or clear the bit
	if available {
		b.executionPayloadAvailability[byteIndex] |= 1 << bitPosition
	} else {
		b.executionPayloadAvailability[byteIndex] &^= 1 << bitPosition
	}

	b.markFieldAsDirty(types.ExecutionPayloadAvailability)
	return nil
}

// SetBuilderPendingPayments sets the entire builder pending payments array.
func (b *BeaconState) SetBuilderPendingPayments(payments []*ethpb.BuilderPendingPayment) error {
	if b.version < version.Gloas {
		return errNotSupported("SetBuilderPendingPayments", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.builderPendingPayments = ethpb.CopyBuilderPendingPaymentSlice(payments)
	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}

// AppendBuilderPendingWithdrawal appends a builder pending withdrawal to the list.
func (b *BeaconState) AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error {
	if b.version < version.Gloas {
		return errNotSupported("AppendBuilderPendingWithdrawal", b.version)
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.builderPendingWithdrawals = append(b.builderPendingWithdrawals, ethpb.CopyBuilderPendingWithdrawal(withdrawal))
	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	return nil
}
