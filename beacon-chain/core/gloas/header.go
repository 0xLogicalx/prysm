package gloas

import (
	"fmt"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/crypto/bls"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/time/slots"
)

// ProcessExecutionPayloadHeader processes a signed execution payload header in the Gloas fork.
// It follows the official Ethereum spec for validating builder bids and managing payments.
func ProcessExecutionPayloadHeader(st state.BeaconState, block interfaces.ReadOnlyBeaconBlock) error {
	signedHeader, err := block.Body().SignedExecutionPayloadHeader()
	if err != nil {
		return fmt.Errorf("failed to get signed execution payload header: %w", err)
	}

	wrappedHeader, err := blocks.WrappedROSignedExecutionPayloadHeader(signedHeader)
	if err != nil {
		return fmt.Errorf("failed to wrap signed header: %w", err)
	}

	if err := validatePayloadHeaderSignature(st, wrappedHeader); err != nil {
		return fmt.Errorf("header signature validation failed: %w", err)
	}

	header, err := wrappedHeader.Header()
	if err != nil {
		return fmt.Errorf("failed to get header from wrapped header: %w", err)
	}

	if err := validateBuilder(st, header, block.ProposerIndex()); err != nil {
		return fmt.Errorf("builder validation failed: %w", err)
	}

	if err := validateHeaderConsistency(st, header, block); err != nil {
		return fmt.Errorf("header consistency validation failed: %w", err)
	}

	// Cache the pending payment
	feeRecipient := header.FeeRecipient()
	pendingPayment := &ethpb.BuilderPendingPayment{
		Weight: 0,
		Withdrawal: &ethpb.BuilderPendingWithdrawal{
			FeeRecipient: feeRecipient[:],
			Amount:       header.Value(),
			BuilderIndex: header.BuilderIndex(),
		},
	}
	slotIndex := params.BeaconConfig().SlotsPerEpoch + (header.Slot() % params.BeaconConfig().SlotsPerEpoch)
	if err := st.SetBuilderPendingPayment(slotIndex, pendingPayment); err != nil {
		return fmt.Errorf("failed to set pending payment: %w", err)
	}

	// Cache the signed execution payload header
	if err := st.SetExecutionPayloadHeader(header); err != nil {
		return fmt.Errorf("failed to cache execution payload header: %w", err)
	}

	return nil
}

// validateBuilder checks if the builder is eligible to submit execution payload headers.
// This includes self-build validation, withdrawal credential checks, and enough balance.
func validateBuilder(st state.BeaconState, header interfaces.ROExecutionPayloadHeaderGloas, proposerIndex primitives.ValidatorIndex) error {
	builderIndex := header.BuilderIndex()
	builder, err := st.ValidatorAtIndex(builderIndex)
	if err != nil {
		return fmt.Errorf("failed to get builder validator: %w", err)
	}

	currentEpoch := slots.ToEpoch(st.Slot())
	if !helpers.IsActiveValidator(builder, currentEpoch) {
		return fmt.Errorf("builder %d is not active in epoch %d", builderIndex, currentEpoch)
	}

	if builder.Slashed {
		return fmt.Errorf("builder %d is slashed", builderIndex)
	}

	amount := header.Value()

	// Self-build validation: amount must be zero when builder == proposer
	fmt.Println(builderIndex, proposerIndex, amount)
	if builderIndex == proposerIndex {
		if amount != 0 {
			return fmt.Errorf("self-build amount must be zero, got %d", amount)
		}
	} else {
		// Non-self builds require builder withdrawal credential
		if err := validateBuilderWithdrawalCredential(builder); err != nil {
			return fmt.Errorf("builder withdrawal credential validation failed: %w", err)
		}
	}

	if err := validateBuilderHasEnoughBalance(st, builderIndex, amount); err != nil {
		return fmt.Errorf("builder financial capacity validation failed: %w", err)
	}

	return nil
}

// validateHeaderConsistency checks that the header is consistent with the current beacon state.
func validateHeaderConsistency(st state.BeaconState, header interfaces.ROExecutionPayloadHeaderGloas, block interfaces.ReadOnlyBeaconBlock) error {
	// Verify that the bid is for the current slot
	if header.Slot() != block.Slot() {
		return fmt.Errorf("header slot %d does not match block slot %d", header.Slot(), block.Slot())
	}

	// Verify that the bid is for the right parent block hash
	latestBlockHash, err := st.LatestBlockHash()
	if err != nil {
		return fmt.Errorf("failed to get latest block hash: %w", err)
	}
	if header.ParentBlockHash() != latestBlockHash {
		return fmt.Errorf("header parent block hash mismatch: got %x, expected %x",
			header.ParentBlockHash(), latestBlockHash)
	}

	// Verify that the bid is for the right parent block root
	if header.ParentBlockRoot() != block.ParentRoot() {
		return fmt.Errorf("header parent block root mismatch: got %x, expected %x",
			header.ParentBlockRoot(), block.ParentRoot())
	}

	return nil
}

// validateBuilderWithdrawalCredential checks if the builder has the correct withdrawal credential prefix.
func validateBuilderWithdrawalCredential(validator *ethpb.Validator) error {
	// Check if withdrawal credential has the builder prefix (0x02)
	if len(validator.WithdrawalCredentials) != 32 {
		return fmt.Errorf("invalid withdrawal credential length: %d", len(validator.WithdrawalCredentials))
	}

	if validator.WithdrawalCredentials[0] != params.BeaconConfig().BuilderWithdrawalPrefixByte {
		return fmt.Errorf("builder must have withdrawal credential prefix 0x%02x, got 0x%02x",
			params.BeaconConfig().BuilderWithdrawalPrefixByte,
			validator.WithdrawalCredentials[0])
	}

	return nil
}

// validateBuilderHasEnoughBalance checks if the builder has sufficient funds for the bid.
func validateBuilderHasEnoughBalance(st state.BeaconState, builderIndex primitives.ValidatorIndex, amount primitives.Gwei) error {
	if amount == 0 {
		return nil // No payment required
	}

	builderBalance, err := st.BalanceAtIndex(builderIndex)
	if err != nil {
		return fmt.Errorf("failed to get builder balance: %w", err)
	}

	// Sum pending payments for this builder
	pendingPayments, err := calculatePendingPayments(st, builderIndex)
	if err != nil {
		return fmt.Errorf("failed to calculate pending payments: %w", err)
	}

	// Sum pending withdrawals for this builder
	pendingWithdrawals, err := calculatePendingWithdrawals(st, builderIndex)
	if err != nil {
		return fmt.Errorf("failed to calculate pending withdrawals: %w", err)
	}

	minActivationBalance := params.BeaconConfig().MinActivationBalance
	requiredBalance := uint64(amount) + pendingPayments + pendingWithdrawals + minActivationBalance

	if builderBalance < requiredBalance {
		return fmt.Errorf("builder %d has insufficient balance: has %d, needs %d (amount=%d, pending_payments=%d, pending_withdrawals=%d, min_activation=%d)",
			builderIndex, builderBalance, requiredBalance, amount, pendingPayments, pendingWithdrawals, minActivationBalance)
	}

	return nil
}

// calculatePendingPayments sums all pending payments for a given builder.
func calculatePendingPayments(st state.BeaconState, builderIndex primitives.ValidatorIndex) (uint64, error) {
	pendingPayments, err := st.BuilderPendingPayments()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending payments: %w", err)
	}

	var total uint64
	for _, payment := range pendingPayments {
		if payment.Withdrawal.BuilderIndex == builderIndex {
			total += uint64(payment.Withdrawal.Amount)
		}
	}

	return total, nil
}

// calculatePendingWithdrawals sums all pending withdrawals for a given builder.
func calculatePendingWithdrawals(st state.BeaconState, builderIndex primitives.ValidatorIndex) (uint64, error) {
	pendingWithdrawals, err := st.BuilderPendingWithdrawals()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending withdrawals: %w", err)
	}

	var total uint64
	for _, withdrawal := range pendingWithdrawals {
		if withdrawal.BuilderIndex == builderIndex {
			total += uint64(withdrawal.Amount)
		}
	}

	return total, nil
}

// validatePayloadHeaderSignature verifies the BLS signature on a signed execution payload header.
// It validates that the signature was created by the builder specified in the header
// using the appropriate domain for the beacon builder.
func validatePayloadHeaderSignature(st state.ReadOnlyBeaconState, signedHeader interfaces.ROSignedExecutionPayloadHeader) error {
	header, err := signedHeader.Header()
	if err != nil {
		return fmt.Errorf("failed to get header: %w", err)
	}

	builderPubkey := st.PubkeyAtIndex(header.BuilderIndex())
	publicKey, err := bls.PublicKeyFromBytes(builderPubkey[:])
	if err != nil {
		return fmt.Errorf("invalid builder public key: %w", err)
	}

	signatureBytes := signedHeader.Signature()
	signature, err := bls.SignatureFromBytes(signatureBytes[:])
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	currentEpoch := slots.ToEpoch(header.Slot())
	domain, err := signing.Domain(
		st.Fork(),
		currentEpoch,
		params.BeaconConfig().DomainBeaconBuilder,
		st.GenesisValidatorsRoot(),
	)
	if err != nil {
		return fmt.Errorf("failed to compute signing domain: %w", err)
	}

	signingRoot, err := signedHeader.SigningRoot(domain)
	if err != nil {
		return fmt.Errorf("failed to compute signing root: %w", err)
	}

	if !signature.Verify(publicKey, signingRoot[:]) {
		return fmt.Errorf("signature verification failed: %w", signing.ErrSigFailedToVerify)
	}

	return nil
}
