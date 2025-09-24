package hdiff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	fieldparams "github.com/OffchainLabs/prysm/v6/config/fieldparams"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/pkg/errors"
)

// readExecutionPayloadAvailability reads the execution payload availability from the data.
func (ret *stateDiff) readExecutionPayloadAvailability(data *[]byte) error {
	const executionPayloadAvailabilityLength = fieldparams.BlockRootsLength / 8
	if len(*data) < executionPayloadAvailabilityLength {
		return errors.Wrap(errDataSmall, "executionPayloadAvailability")
	}
	ret.executionPayloadAvailability = make([]byte, executionPayloadAvailabilityLength)
	copy(ret.executionPayloadAvailability, (*data)[:executionPayloadAvailabilityLength])
	*data = (*data)[executionPayloadAvailabilityLength:]
	return nil
}

// readLatestBlockHash reads the latest block hash from the data.
func (ret *stateDiff) readLatestBlockHash(data *[]byte) error {
	if len(*data) < fieldparams.RootLength {
		return errors.Wrap(errDataSmall, "latestBlockHash")
	}
	copy(ret.latestBlockHash[:], (*data)[:fieldparams.RootLength])
	*data = (*data)[fieldparams.RootLength:]
	return nil
}

// readLatestWithdrawalsRoot reads the latest withdrawals root from the data.
func (ret *stateDiff) readLatestWithdrawalsRoot(data *[]byte) error {
	if len(*data) < fieldparams.RootLength {
		return errors.Wrap(errDataSmall, "latestWithdrawalsRoot")
	}
	copy(ret.latestWithdrawalsRoot[:], (*data)[:fieldparams.RootLength])
	*data = (*data)[fieldparams.RootLength:]
	return nil
}

// readBuilderPendingPayments reads the builder pending payments diff from the data.
func (ret *stateDiff) readBuilderPendingPayments(data *[]byte) error {
	if len(*data) < 16 {
		return errors.Wrap(errDataSmall, "builderPendingPayments header")
	}

	ret.builderPendingPaymentsStartIndex = binary.LittleEndian.Uint64((*data)[:8])
	paymentCount := int(binary.LittleEndian.Uint64((*data)[8:16]))

	const paymentSize = 8 + 20 + 8 + 8 + 8 // weight + feeRecipient + amount + builderIndex + withdrawableEpoch
	totalSize := 16 + paymentCount*paymentSize

	if len(*data) < totalSize {
		return errors.Wrap(errDataSmall, "builderPendingPaymentsDiff data")
	}

	ret.builderPendingPaymentsDiff = make([]*ethpb.BuilderPendingPayment, paymentCount)
	cursor := 16
	for i := 0; i < paymentCount; i++ {
		payment := &ethpb.BuilderPendingPayment{
			Withdrawal: &ethpb.BuilderPendingWithdrawal{},
		}

		payment.Weight = primitives.Gwei(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		payment.Withdrawal.FeeRecipient = make([]byte, 20)
		copy(payment.Withdrawal.FeeRecipient, (*data)[cursor:cursor+20])
		cursor += 20

		payment.Withdrawal.Amount = primitives.Gwei(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		payment.Withdrawal.BuilderIndex = primitives.ValidatorIndex(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		payment.Withdrawal.WithdrawableEpoch = primitives.Epoch(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		ret.builderPendingPaymentsDiff[i] = payment
	}

	*data = (*data)[cursor:]
	return nil
}

// readBuilderPendingWithdrawals reads the builder pending withdrawals diff from the data.
func (ret *stateDiff) readBuilderPendingWithdrawals(data *[]byte) error {
	if len(*data) < 16 {
		return errors.Wrap(errDataSmall, "builderPendingWithdrawals header")
	}

	ret.builderPendingWithdrawalsStartIndex = binary.LittleEndian.Uint64((*data)[:8])
	withdrawalCount := int(binary.LittleEndian.Uint64((*data)[8:16]))

	totalSize := 16 + withdrawalCount*builderPendingWithdrawalLength

	if len(*data) < totalSize {
		return errors.Wrap(errDataSmall, "builderPendingWithdrawalsDiff data")
	}

	ret.builderPendingWithdrawalsDiff = make([]*ethpb.BuilderPendingWithdrawal, withdrawalCount)
	cursor := 16
	for i := 0; i < withdrawalCount; i++ {
		withdrawal := &ethpb.BuilderPendingWithdrawal{}

		withdrawal.FeeRecipient = make([]byte, 20)
		copy(withdrawal.FeeRecipient, (*data)[cursor:cursor+20])
		cursor += 20

		withdrawal.Amount = primitives.Gwei(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		withdrawal.BuilderIndex = primitives.ValidatorIndex(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		withdrawal.WithdrawableEpoch = primitives.Epoch(binary.LittleEndian.Uint64((*data)[cursor : cursor+8]))
		cursor += 8

		ret.builderPendingWithdrawalsDiff[i] = withdrawal
	}

	*data = (*data)[cursor:]
	return nil
}

// readExecutionPayloadBid reads the execution payload bid from the data.
func (ret *stateDiff) readExecutionPayloadBid(data *[]byte) error {
	if len(*data) < 1 {
		return errors.Wrap(errDataSmall, "executionPayloadBid marker")
	}
	if (*data)[0] == nilMarker {
		*data = (*data)[1:]
		return nil
	}
	if len(*data) < 5 { // 1 byte marker + at least 4 bytes for size
		return errors.Wrap(errDataSmall, "executionPayloadBid size header")
	}
	bidSize := binary.LittleEndian.Uint32((*data)[1:5])
	totalSize := 1 + 4 + int(bidSize)
	if len(*data) < totalSize {
		return errors.Wrap(errDataSmall, "executionPayloadBid data")
	}

	ret.executionpayloadBid = &ethpb.ExecutionPayloadBid{}
	if err := ret.executionpayloadBid.UnmarshalSSZ((*data)[5:totalSize]); err != nil {
		return errors.Wrap(err, "failed to unmarshal ExecutionPayloadBid")
	}
	*data = (*data)[totalSize:]
	return nil
}

// writeBuilderPendingPayments writes the builder pending payments diff to the data.
// Format: startIndex(8) + payment data from startIndex to end of buffer
func (diff *stateDiff) writeBuilderPendingPayments(data *[]byte, startIndex uint64) error {
	const headerSize = 8
	numPayments := len(diff.builderPendingPaymentsDiff)
	totalSize := headerSize + numPayments*builderPendingPaymentLength

	// Grow the slice if needed
	oldLen := len(*data)
	*data = append(*data, make([]byte, totalSize)...)
	cursor := oldLen

	// Write start index
	binary.LittleEndian.PutUint64((*data)[cursor:cursor+headerSize], startIndex)
	cursor += headerSize

	// Write payment data
	for _, payment := range diff.builderPendingPaymentsDiff {
		paymentBytes, err := payment.MarshalSSZ()
		if err != nil {
			return errors.Wrap(err, "failed to marshal BuilderPendingPayment")
		}
		if len(paymentBytes) != builderPendingPaymentLength {
			return errors.Errorf("unexpected payment size: got %d, expected %d", len(paymentBytes), builderPendingPaymentLength)
		}
		copy((*data)[cursor:cursor+builderPendingPaymentLength], paymentBytes)
		cursor += builderPendingPaymentLength
	}

	return nil
}

// diffBuilderPendingPayments computes the diff for builder pending payments circular buffer
func diffBuilderPendingPayments(diff *stateDiff, source, target state.ReadOnlyBeaconState) error {
	tgt := target.BuilderPendingPayments()
	src := source.BuilderPendingPayments()
	paymentLength := 2 * fieldparams.SlotsPerEpoch
	if len(tgt) != paymentLength || len(src) != paymentLength {
		return errors.Errorf("unexpected builder pending payments length: got src %d, tgt %d, expected %d", len(src), len(tgt), paymentLength)
	}

	startIndex := paymentLength

	for i := 0; i < paymentLength; i++ {
		if !builderPendingPaymentEqual(src[i], tgt[i]) {
			startIndex = i
			break
		}
	}

	if startIndex == paymentLength {
		diff.builderPendingPaymentsStartIndex = uint64(paymentLength)
		diff.builderPendingPaymentsDiff = make([]*ethpb.BuilderPendingPayment, 0)
		return nil
	}
	diff.builderPendingPaymentsStartIndex = uint64(startIndex)
	diff.builderPendingPaymentsDiff = make([]*ethpb.BuilderPendingPayment, paymentLength-startIndex)

	for i := startIndex; i < paymentLength; i++ {
		diff.builderPendingPaymentsDiff[i-startIndex] = &ethpb.BuilderPendingPayment{
			Weight: tgt[i].Weight,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      slices.Clone(tgt[i].Withdrawal.FeeRecipient),
				Amount:            tgt[i].Withdrawal.Amount,
				BuilderIndex:      tgt[i].Withdrawal.BuilderIndex,
				WithdrawableEpoch: tgt[i].Withdrawal.WithdrawableEpoch,
			},
		}

	}
	return nil
}

// diffBuilderPendingWithdrawals computes the diff for builder pending withdrawals variable-length list
func diffBuilderPendingWithdrawals(diff *stateDiff, source, target state.ReadOnlyBeaconState) error {
	tPendingWithdrawals := target.BuilderPendingWithdrawals()
	tlen := len(tPendingWithdrawals)
	var sPendingWithdrawals []*ethpb.BuilderPendingWithdrawal
	if source.Version() >= version.Gloas {
		sPendingWithdrawals = source.BuilderPendingWithdrawals()
	}

	// Find the optimal starting index using simple prefix matching
	index := 0
	minLen := min(len(sPendingWithdrawals), tlen)
	for i := 0; i < minLen; i++ {
		if !builderPendingWithdrawalEqual(sPendingWithdrawals[i], tPendingWithdrawals[i]) {
			break
		}
		index = i + 1
	}

	diff.builderPendingWithdrawalsStartIndex = uint64(index)
	diff.builderPendingWithdrawalsDiff = make([]*ethpb.BuilderPendingWithdrawal, tlen-index)
	for i, withdrawal := range tPendingWithdrawals[index:] {
		diff.builderPendingWithdrawalsDiff[i] = &ethpb.BuilderPendingWithdrawal{
			FeeRecipient:      slices.Clone(withdrawal.FeeRecipient),
			Amount:            withdrawal.Amount,
			BuilderIndex:      withdrawal.BuilderIndex,
			WithdrawableEpoch: withdrawal.WithdrawableEpoch,
		}
	}
	return nil
}

// builderPendingWithdrawalEqual compares two BuilderPendingWithdrawal structs for equality
func builderPendingWithdrawalEqual(a, b *ethpb.BuilderPendingWithdrawal) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return bytes.Equal(a.FeeRecipient, b.FeeRecipient) &&
		a.Amount == b.Amount &&
		a.BuilderIndex == b.BuilderIndex &&
		a.WithdrawableEpoch == b.WithdrawableEpoch
}

// builderPendingPaymentEqual compares two BuilderPendingPayment structs for equality
func builderPendingPaymentEqual(a, b *ethpb.BuilderPendingPayment) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Weight != b.Weight {
		return false
	}
	if a.Withdrawal == nil && b.Withdrawal == nil {
		return true
	}
	if a.Withdrawal == nil || b.Withdrawal == nil {
		return false
	}
	return bytes.Equal(a.Withdrawal.FeeRecipient, b.Withdrawal.FeeRecipient) &&
		a.Withdrawal.Amount == b.Withdrawal.Amount &&
		a.Withdrawal.BuilderIndex == b.Withdrawal.BuilderIndex &&
		a.Withdrawal.WithdrawableEpoch == b.Withdrawal.WithdrawableEpoch
}

// applyGloasDiff applies the Gloas fields diff to the source state in place.
func applyGloasDiff(source state.BeaconState, diff *stateDiff) error {
	if err := source.SetExecutionPayloadAvailability(diff.executionPayloadAvailability); err != nil {
		return errors.Wrap(err, "failed to set execution payload availability")
	}

	if err := applyBuilderPendingPaymentsDiff(source, diff); err != nil {
		return errors.Wrap(err, "failed to apply builder pending payments diff")
	}

	if err := applyBuilderPendingWithdrawalsDiff(source, diff); err != nil {
		return errors.Wrap(err, "failed to apply builder pending withdrawals diff")
	}

	if err := source.SetLatestBlockHash(diff.latestBlockHash); err != nil {
		return errors.Wrap(err, "failed to set latest block hash")
	}
	if err := source.SetLatestWithdrawalsRoot(diff.latestWithdrawalsRoot); err != nil {
		return errors.Wrap(err, "failed to set latest withdrawals root")
	}
	return nil
}

// applyBuilderPendingPaymentsDiff applies the builder pending payments diff to the source state
func applyBuilderPendingPaymentsDiff(source state.BeaconState, diff *stateDiff) error {
	if len(diff.builderPendingPaymentsDiff) == 0 {
		return nil
	}

	pendingPayments := source.BuilderPendingPayments()
	if len(pendingPayments) != 2*fieldparams.SlotsPerEpoch {
		return fmt.Errorf("incorrect builder pending payments length: got %d, expected %d", len(pendingPayments), 2*fieldparams.SlotsPerEpoch)
	}

	// Apply diff from startIndex to end
	startIndex := int(diff.builderPendingPaymentsStartIndex)
	for i, payment := range diff.builderPendingPaymentsDiff {
		pendingPayments[startIndex+i] = payment
	}

	// Set the updated payments back to the state
	return source.SetBuilderPendingPayments(pendingPayments)
}

// applyBuilderPendingWithdrawalsDiff applies the builder pending withdrawals diff to the source state
func applyBuilderPendingWithdrawalsDiff(source state.BeaconState, diff *stateDiff) error {
	pendingWithdrawals := source.BuilderPendingWithdrawals()
	pendingWithdrawals = pendingWithdrawals[:int(diff.builderPendingWithdrawalsStartIndex)]
	for _, withdrawal := range diff.builderPendingWithdrawalsDiff {
		pendingWithdrawals = append(pendingWithdrawals, &ethpb.BuilderPendingWithdrawal{
			FeeRecipient:      slices.Clone(withdrawal.FeeRecipient),
			Amount:            withdrawal.Amount,
			BuilderIndex:      withdrawal.BuilderIndex,
			WithdrawableEpoch: withdrawal.WithdrawableEpoch,
		})
	}
	return source.SetBuilderPendingWithdrawals(pendingWithdrawals)
}
