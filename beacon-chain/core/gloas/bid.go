package gloas

import (
	"errors"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/crypto/bls/common"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// ProcessExecutionPayloadBid processes a signed execution payload bid in the Gloas fork.
// Spec v1.6.1 (pseudocode):
// process_execution_payload_bid(state: BeaconState, block: BeaconBlock):
//
//	signed_bid = block.body.signed_execution_payload_bid
//	bid = signed_bid.message
//	builder_index = bid.builder_index
//	builder = state.validators[builder_index]
//	amount = bid.value
//	if builder_index == block.proposer_index:
//	  assert amount == 0
//	  assert signed_bid.signature == G2_POINT_AT_INFINITY
//	else:
//	  assert has_builder_withdrawal_credential(builder)
//	  assert verify_execution_payload_bid_signature(state, signed_bid)
//	assert is_active_validator(builder, get_current_epoch(state))
//	assert not builder.slashed
//	assert amount == 0 or state.balances[builder_index] >= amount + pending_payments + pending_withdrawals + MIN_ACTIVATION_BALANCE
//	assert bid.slot == block.slot
//	assert bid.parent_block_hash == state.latest_block_hash
//	assert bid.parent_block_root == block.parent_root
//	assert bid.prev_randao == get_randao_mix(state, get_current_epoch(state))
//	if amount > 0:
//	  state.builder_pending_payments[...] = BuilderPendingPayment(weight=0, withdrawal=BuilderPendingWithdrawal(fee_recipient=bid.fee_recipient, amount=amount, builder_index=builder_index, withdrawable_epoch=FAR_FUTURE_EPOCH))
//	state.latest_execution_payload_bid = bid
func ProcessExecutionPayloadBid(st state.BeaconState, block interfaces.ReadOnlyBeaconBlock) error {
	signedBid, err := block.Body().SignedExecutionPayloadBid()
	if err != nil {
		return fmt.Errorf("failed to get signed execution payload bid: %w", err)
	}

	wrappedBid, err := blocks.WrappedROSignedExecutionPayloadBid(signedBid)
	if err != nil {
		return fmt.Errorf("failed to wrap signed bid: %w", err)
	}

	bid, err := wrappedBid.Bid()
	if err != nil {
		return fmt.Errorf("failed to get bid from wrapped bid: %w", err)
	}

	builderIndex := bid.BuilderIndex()
	proposerIndex := block.ProposerIndex()
	amount := bid.Value()

	builder, err := st.ValidatorAtIndex(builderIndex)
	if err != nil {
		return fmt.Errorf("failed to get builder validator: %w", err)
	}

	if builderIndex == proposerIndex {
		if amount != 0 {
			return fmt.Errorf("self-build amount must be zero, got %d", amount)
		}
		if wrappedBid.Signature() != common.InfiniteSignature {
			return errors.New("self-build signature must be point at infinity")
		}
	} else {
		if err := validateBuilderWithdrawalCredential(builder); err != nil {
			return fmt.Errorf("builder withdrawal credential validation failed: %w", err)
		}
		if err := validatePayloadBidSignature(st, wrappedBid); err != nil {
			return fmt.Errorf("bid signature validation failed: %w", err)
		}
	}

	if err := validateBuilderStatus(st, builder); err != nil {
		return fmt.Errorf("builder validation failed: %w", err)
	}

	if err := validateBuilderHasEnoughBalance(st, builderIndex, amount); err != nil {
		return fmt.Errorf("builder financial capacity validation failed: %w", err)
	}

	if err := validateBidConsistency(st, bid, block); err != nil {
		return fmt.Errorf("bid consistency validation failed: %w", err)
	}

	if amount > 0 {
		feeRecipient := bid.FeeRecipient()
		pendingPayment := &ethpb.BuilderPendingPayment{
			Weight: 0,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      feeRecipient[:],
				Amount:            amount,
				BuilderIndex:      builderIndex,
				WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			},
		}
		slotIndex := params.BeaconConfig().SlotsPerEpoch + (bid.Slot() % params.BeaconConfig().SlotsPerEpoch)
		if err := st.SetBuilderPendingPayment(slotIndex, pendingPayment); err != nil {
			return fmt.Errorf("failed to set pending payment: %w", err)
		}
	}

	if err := st.SetExecutionPayloadBid(bid); err != nil {
		return fmt.Errorf("failed to cache execution payload bid: %w", err)
	}

	return nil
}

// validateBuilderStatus checks if the builder is active and not slashed.
func validateBuilderStatus(st state.BeaconState, builder *ethpb.Validator) error {
	currentEpoch := slots.ToEpoch(st.Slot())
	if !helpers.IsActiveValidator(builder, currentEpoch) {
		return fmt.Errorf("builder is not active in epoch %d", currentEpoch)
	}

	if builder.Slashed {
		return errors.New("builder is slashed")
	}

	return nil
}

// validateBidConsistency checks that the bid is consistent with the current beacon state.
func validateBidConsistency(st state.BeaconState, bid interfaces.ROExecutionPayloadBid, block interfaces.ReadOnlyBeaconBlock) error {
	if bid.Slot() != block.Slot() {
		return fmt.Errorf("bid slot %d does not match block slot %d", bid.Slot(), block.Slot())
	}

	latestBlockHash, err := st.LatestBlockHash()
	if err != nil {
		return fmt.Errorf("failed to get latest block hash: %w", err)
	}
	if bid.ParentBlockHash() != latestBlockHash {
		return fmt.Errorf("bid parent block hash mismatch: got %x, expected %x",
			bid.ParentBlockHash(), latestBlockHash)
	}

	if bid.ParentBlockRoot() != block.ParentRoot() {
		return fmt.Errorf("bid parent block root mismatch: got %x, expected %x",
			bid.ParentBlockRoot(), block.ParentRoot())
	}

	randaoMix, err := helpers.RandaoMix(st, slots.ToEpoch(st.Slot()))
	if err != nil {
		return fmt.Errorf("failed to get randao mix: %w", err)
	}
	if bid.PrevRandao() != [32]byte(randaoMix) {
		return fmt.Errorf("bid prev randao mismatch: got %x, expected %x", bid.PrevRandao(), randaoMix)
	}

	return nil
}

// validateBuilderWithdrawalCredential checks if the builder has the correct withdrawal credential prefix.
func validateBuilderWithdrawalCredential(validator *ethpb.Validator) error {
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
		return nil
	}

	builderBalance, err := st.BalanceAtIndex(builderIndex)
	if err != nil {
		return fmt.Errorf("failed to get builder balance: %w", err)
	}

	pendingPayments, err := st.PendingPaymentSum(builderIndex)
	if err != nil {
		return fmt.Errorf("failed to calculate pending payments: %w", err)
	}

	pendingWithdrawals, err := st.PendingWithdrawalSum(builderIndex)
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

// validatePayloadBidSignature verifies the BLS signature on a signed execution payload bid.
// It validates that the signature was created by the builder specified in the bid
// using the appropriate domain for the beacon builder.
func validatePayloadBidSignature(st state.ReadOnlyBeaconState, signedBid interfaces.ROSignedExecutionPayloadBid) error {
	bid, err := signedBid.Bid()
	if err != nil {
		return fmt.Errorf("failed to get bid: %w", err)
	}

	builderPubkey := st.PubkeyAtIndex(bid.BuilderIndex())
	publicKey, err := bls.PublicKeyFromBytes(builderPubkey[:])
	if err != nil {
		return fmt.Errorf("invalid builder public key: %w", err)
	}

	signatureBytes := signedBid.Signature()
	signature, err := bls.SignatureFromBytes(signatureBytes[:])
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	currentEpoch := slots.ToEpoch(bid.Slot())
	domain, err := signing.Domain(
		st.Fork(),
		currentEpoch,
		params.BeaconConfig().DomainBeaconBuilder,
		st.GenesisValidatorsRoot(),
	)
	if err != nil {
		return fmt.Errorf("failed to compute signing domain: %w", err)
	}

	signingRoot, err := signedBid.SigningRoot(domain)
	if err != nil {
		return fmt.Errorf("failed to compute signing root: %w", err)
	}

	if !signature.Verify(publicKey, signingRoot[:]) {
		return fmt.Errorf("signature verification failed: %w", signing.ErrSigFailedToVerify)
	}

	return nil
}
