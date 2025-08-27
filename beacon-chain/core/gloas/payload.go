package gloas

import (
	"bytes"
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/electra"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/crypto/bls"
	"github.com/OffchainLabs/prysm/v6/encoding/ssz"
	enginev1 "github.com/OffchainLabs/prysm/v6/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/time/slots"
	"github.com/pkg/errors"
)

// ProcessExecutionPayload processes the signed execution payload envelope for Gloas fork.
// This function implements the process_execution_payload spec for Gloas.
func ProcessExecutionPayload(
	ctx context.Context,
	st state.BeaconState,
	signedEnvelope interfaces.ROSignedExecutionPayloadEnvelope,
) error {
	// Verify the signature on the signed execution payload envelope
	if err := verifyExecutionPayloadEnvelopeSignature(st, signedEnvelope); err != nil {
		return errors.Wrap(err, "signature verification failed")
	}

	envelope, err := signedEnvelope.Envelope()
	if err != nil {
		return errors.Wrap(err, "could not get envelope from signed envelope")
	}
	payload, err := envelope.Execution()
	if err != nil {
		return errors.Wrap(err, "could not get execution payload from envelope")
	}

	latestHeader := st.LatestBlockHeader()
	if len(latestHeader.StateRoot) == 0 || bytes.Equal(latestHeader.StateRoot, make([]byte, 32)) {
		previousStateRoot, err := st.HashTreeRoot(ctx)
		if err != nil {
			return errors.Wrap(err, "could not compute state root")
		}
		latestHeader.StateRoot = previousStateRoot[:]
		if err := st.SetLatestBlockHeader(latestHeader); err != nil {
			return errors.Wrap(err, "could not set latest block header")
		}
	}

	blockHeaderRoot, err := latestHeader.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute block header root")
	}
	beaconBlockRoot := envelope.BeaconBlockRoot()
	if !bytes.Equal(beaconBlockRoot[:], blockHeaderRoot[:]) {
		return errors.Errorf("envelope beacon block root does not match state latest block header root: envelope=%#x, header=%#x", beaconBlockRoot, blockHeaderRoot)
	}

	if envelope.Slot() != st.Slot() {
		return errors.Errorf("envelope slot does not match state slot: envelope=%d, state=%d", envelope.Slot(), st.Slot())
	}

	committedHeader, err := st.ExecutionHeader()
	if err != nil {
		return errors.Wrap(err, "could not get execution payload header")
	}
	if envelope.BuilderIndex() != committedHeader.BuilderIndex() {
		return errors.Errorf("envelope builder index does not match committed header builder index: envelope=%d, header=%d", envelope.BuilderIndex(), committedHeader.BuilderIndex())
	}

	envelopeBlobCommitments := envelope.BlobKzgCommitments()
	envelopeBlobRoot, err := ssz.KzgCommitmentsRoot(envelopeBlobCommitments)
	if err != nil {
		return errors.Wrap(err, "could not compute envelope blob KZG commitments root")
	}
	committedBlobRoot := committedHeader.BlobKzgCommitmentsRoot()
	if !bytes.Equal(committedBlobRoot[:], envelopeBlobRoot[:]) {
		return errors.Errorf("committed header blob KZG commitments root does not match envelope: header=%#x, envelope=%#x", committedBlobRoot, envelopeBlobRoot)
	}

	withdrawals, err := payload.Withdrawals()
	if err != nil {
		return errors.Wrap(err, "could not get withdrawals from payload")
	}
	withdrawalsRoot, err := ssz.WithdrawalSliceRoot(withdrawals, params.BeaconConfig().MaxWithdrawalsPerPayload)
	if err != nil {
		return errors.Wrap(err, "could not compute withdrawals root")
	}

	latestWithdrawalsRoot, err := st.LatestWithdrawalsRoot()
	if err != nil {
		return errors.Wrap(err, "could not get latest withdrawals root")
	}
	if !bytes.Equal(withdrawalsRoot[:], latestWithdrawalsRoot[:]) {
		return errors.Errorf("payload withdrawals root does not match state latest withdrawals root: payload=%#x, state=%#x", withdrawalsRoot, latestWithdrawalsRoot)
	}

	if committedHeader.GasLimit() != payload.GasLimit() {
		return errors.Errorf("committed header gas limit does not match payload gas limit: header=%d, payload=%d", committedHeader.GasLimit(), payload.GasLimit())
	}

	headerBlockHash := committedHeader.BlockHash()
	payloadBlockHash := payload.BlockHash()
	if !bytes.Equal(headerBlockHash[:], payloadBlockHash) {
		return errors.Errorf("committed header block hash does not match payload block hash: header=%#x, payload=%#x", headerBlockHash, payloadBlockHash)
	}

	latestBlockHash, err := st.LatestBlockHash()
	if err != nil {
		return errors.Wrap(err, "could not get latest block hash")
	}
	if !bytes.Equal(payload.ParentHash(), latestBlockHash[:]) {
		return errors.Errorf("payload parent hash does not match state latest block hash: payload=%#x, state=%#x", payload.ParentHash(), latestBlockHash)
	}

	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return errors.Wrap(err, "could not get randao mix")
	}
	if !bytes.Equal(payload.PrevRandao(), random) {
		return errors.Errorf("payload prev randao does not match expected randao mix: payload=%#x, expected=%#x", payload.PrevRandao(), random)
	}

	t, err := slots.StartTime(st.GenesisTime(), st.Slot())
	if err != nil {
		return errors.Wrap(err, "could not compute timestamp")
	}
	if payload.Timestamp() != uint64(t.Unix()) {
		return errors.Errorf("payload timestamp does not match expected timestamp: payload=%d, expected=%d", payload.Timestamp(), uint64(t.Unix()))
	}

	maxBlobsPerBlock := params.BeaconConfig().MaxBlobsPerBlock(envelope.Slot())
	if len(envelope.BlobKzgCommitments()) > maxBlobsPerBlock {
		return errors.Errorf("too many blob KZG commitments: got=%d, max=%d", len(envelope.BlobKzgCommitments()), maxBlobsPerBlock)
	}

	// Process execution requests
	if err := processExecutionRequests(ctx, st, envelope.ExecutionRequests()); err != nil {
		return errors.Wrap(err, "could not process execution requests")
	}

	// Set the latest block hash from payload
	if err := st.SetLatestBlockHash([32]byte(payload.BlockHash())); err != nil {
		return errors.Wrap(err, "could not set latest block hash")
	}

	// Set execution payload availability
	if err := st.SetExecutionPayloadAvailability(st.Slot(), true); err != nil {
		return errors.Wrap(err, "could not set execution payload availability")
	}

	// Queue the builder payment
	if err := queueBuilderPayment(ctx, st); err != nil {
		return errors.Wrap(err, "could not queue builder payment")
	}

	// Verify state root
	r, err := st.HashTreeRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get hash tree root")
	}
	if r != envelope.StateRoot() {
		return fmt.Errorf("state root mismatch: expected %#x, got %#x", envelope.StateRoot(), r)
	}

	return nil
}

// processExecutionRequests processes deposits, withdrawals, and consolidations from execution requests.
func processExecutionRequests(ctx context.Context, st state.BeaconState, requests *enginev1.ExecutionRequests) error {
	var err error
	st, err = electra.ProcessDepositRequests(ctx, st, requests.Deposits)
	if err != nil {
		return errors.Wrap(err, "could not process deposit requests")
	}

	st, err = electra.ProcessWithdrawalRequests(ctx, st, requests.Withdrawals)
	if err != nil {
		return errors.Wrap(err, "could not process withdrawal requests")
	}
	err = electra.ProcessConsolidationRequests(ctx, st, requests.Consolidations)
	if err != nil {
		return errors.Wrap(err, "could not process consolidation requests")
	}
	return nil
}

// queueBuilderPayment implements the builder payment queuing logic for Gloas.
func queueBuilderPayment(ctx context.Context, st state.BeaconState) error {
	payment, err := st.BuilderPendingPayment(st.Slot())
	if err != nil {
		return errors.Wrap(err, "could not get builder pending payment")
	}

	exitQueueEpoch, err := st.ExitEpochAndUpdateChurn(payment.Withdrawal.Amount)
	if err != nil {
		return errors.Wrap(err, "could not compute exit epoch and update churn")
	}

	minValidatorWithdrawabilityDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
	payment.Withdrawal.WithdrawableEpoch = exitQueueEpoch + minValidatorWithdrawabilityDelay

	if err := st.AppendBuilderPendingWithdrawal(payment.Withdrawal); err != nil {
		return errors.Wrap(err, "could not append builder pending withdrawal")
	}

	// Clear the payment by setting an empty BuilderPendingPayment
	emptyPayment := &ethpb.BuilderPendingPayment{
		Withdrawal: &ethpb.BuilderPendingWithdrawal{
			FeeRecipient: make([]byte, 20), // Initialize with zero bytes
		},
	}
	if err := st.SetBuilderPendingPayment(st.Slot(), emptyPayment); err != nil {
		return errors.Wrap(err, "could not set builder pending payment")
	}

	return nil
}

// verifyExecutionPayloadEnvelopeSignature verifies the BLS signature on a signed execution payload envelope.
// It validates that the signature was created by the builder specified in the envelope
// using the appropriate domain for the beacon builder.
func verifyExecutionPayloadEnvelopeSignature(st state.BeaconState, signedEnvelope interfaces.ROSignedExecutionPayloadEnvelope) error {
	envelope, err := signedEnvelope.Envelope()
	if err != nil {
		return fmt.Errorf("failed to get envelope: %w", err)
	}

	builderPubkey := st.PubkeyAtIndex(envelope.BuilderIndex())
	publicKey, err := bls.PublicKeyFromBytes(builderPubkey[:])
	if err != nil {
		return fmt.Errorf("invalid builder public key: %w", err)
	}

	signatureBytes := signedEnvelope.Signature()
	signature, err := bls.SignatureFromBytes(signatureBytes[:])
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	currentEpoch := slots.ToEpoch(envelope.Slot())
	domain, err := signing.Domain(
		st.Fork(),
		currentEpoch,
		params.BeaconConfig().DomainBeaconBuilder,
		st.GenesisValidatorsRoot(),
	)
	if err != nil {
		return fmt.Errorf("failed to compute signing domain: %w", err)
	}

	signingRoot, err := signedEnvelope.SigningRoot(domain)
	if err != nil {
		return fmt.Errorf("failed to compute signing root: %w", err)
	}

	if !signature.Verify(publicKey, signingRoot[:]) {
		return fmt.Errorf("signature verification failed: %w", signing.ErrSigFailedToVerify)
	}

	return nil
}
