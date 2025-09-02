// Package gloas implements the Gloas network upgrade functionality for Ethereum beacon chain.
package gloas

import (
	"bytes"
	"context"
	"encoding/binary"
	"slices"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	fieldparams "github.com/OffchainLabs/prysm/v6/config/fieldparams"
	"github.com/OffchainLabs/prysm/v6/config/params"
	consensus_types "github.com/OffchainLabs/prysm/v6/consensus-types"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/crypto/bls"
	"github.com/OffchainLabs/prysm/v6/crypto/hash"
	"github.com/OffchainLabs/prysm/v6/encoding/bytesutil"
	eth "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/OffchainLabs/prysm/v6/time/slots"
	"github.com/pkg/errors"
)

var (
	// ErrGloasNotActive is returned when Gloas fork functionality is used before fork activation.
	ErrGloasNotActive = errors.New("payload attestations require Gloas fork activation")
)

// ProcessPayloadAttestations processes payload attestations from a beacon block body.
// It validates each attestation's beacon block root, slot, and signature according to the Gloas specification.
func ProcessPayloadAttestations(ctx context.Context, state state.BeaconState, body interfaces.ReadOnlyBeaconBlockBody) error {
	if state.Version() < version.Gloas {
		return ErrGloasNotActive
	}

	atts, err := body.PayloadAttestations()
	if err != nil {
		return errors.Wrap(err, "could not get payload attestations from block body")
	}
	if len(atts) == 0 {
		return nil
	}

	latestBlockHeader := state.LatestBlockHeader()
	for i, att := range atts {
		data := att.Data
		if !bytes.Equal(data.BeaconBlockRoot, latestBlockHeader.ParentRoot) {
			return errors.Errorf("payload attestation %d: beacon block root %x does not match parent root %x",
				i, data.BeaconBlockRoot, latestBlockHeader.ParentRoot)
		}
		if data.Slot+1 != state.Slot() {
			return errors.Errorf("payload attestation %d: slot %d is not one less than state slot %d",
				i, data.Slot, state.Slot())
		}
		indexed, err := getIndexedPayloadAttestation(ctx, state, data.Slot, att)
		if err != nil {
			return errors.Wrapf(err, "could not get indexed payload attestation %d", i)
		}
		valid, err := isValidIndexedPayloadAttestation(state, indexed)
		if err != nil {
			return errors.Wrapf(err, "could not validate indexed payload attestation %d", i)
		}
		if !valid {
			return errors.Errorf("payload attestation %d: signature verification failed", i)
		}
	}
	return nil
}

// getIndexedPayloadAttestation converts a payload attestation to an indexed payload attestation.
// It extracts the attesting validator indices from the aggregation bits and sorts them.
func getIndexedPayloadAttestation(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot, att *eth.PayloadAttestation) (*consensus_types.IndexedPayloadAttestation, error) {
	if state.Version() < version.Gloas {
		return nil, ErrGloasNotActive
	}

	attestingIndices, err := getPayloadAttestingIndices(ctx, state, slot, att)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}

	slices.Sort(attestingIndices)

	return &consensus_types.IndexedPayloadAttestation{
		AttestingIndices: attestingIndices,
		Data:             att.Data,
		Signature:        att.Signature,
	}, nil
}

// getPayloadAttestingIndices extracts the validator indices that are attesting in a payload attestation.
// It maps the aggregation bits to the corresponding validators in the payload timeliness committee.
func getPayloadAttestingIndices(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot, att *eth.PayloadAttestation) ([]primitives.ValidatorIndex, error) {
	if state.Version() < version.Gloas {
		return nil, ErrGloasNotActive
	}

	committee, err := getPayloadTimelinessCommittee(ctx, state, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload timeliness committee")
	}

	var attestingIndices []primitives.ValidatorIndex
	for i, validatorIndex := range committee {
		if att.AggregationBits.BitAt(uint64(i)) {
			attestingIndices = append(attestingIndices, validatorIndex)
		}
	}

	return attestingIndices, nil
}

// getPayloadTimelinessCommittee returns the payload timeliness committee for a given slot.
// It aggregates all beacon committees for the slot and applies balance-weighted selection
// to choose PTCSize validators according to the Gloas specification.
func getPayloadTimelinessCommittee(ctx context.Context, state state.ReadOnlyBeaconState, slot primitives.Slot) ([]primitives.ValidatorIndex, error) {
	if state.Version() < version.Gloas {
		return nil, ErrGloasNotActive
	}

	epoch := slots.ToEpoch(slot)
	seed, err := computePTCSeed(state, epoch, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute PTC seed")
	}

	committeesPerSlot := helpers.SlotCommitteeCount(uint64(state.NumValidators()))
	s := uint64(state.NumValidators()) / committeesPerSlot
	candidateIndices := make([]primitives.ValidatorIndex, 0, committeesPerSlot*s)

	for i := primitives.CommitteeIndex(0); i < primitives.CommitteeIndex(committeesPerSlot); i++ {
		committee, err := helpers.BeaconCommitteeFromState(ctx, state, slot, i)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get beacon committee %d", i)
		}
		candidateIndices = append(candidateIndices, committee...)
	}

	return computeBalanceWeightedSelection(state, candidateIndices, seed, fieldparams.PTCSize)
}

// computePTCSeed generates the seed for payload timeliness committee selection.
// It combines the epoch seed with the slot number to ensure different committees per slot.
func computePTCSeed(state state.ReadOnlyBeaconState, epoch primitives.Epoch, slot primitives.Slot) ([32]byte, error) {
	baseSeed, err := helpers.Seed(state, epoch, params.BeaconConfig().DomainPTCAttester)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute base seed")
	}

	return hash.Hash(append(baseSeed[:], bytesutil.Bytes8(uint64(slot))...)), nil
}

// computeBalanceWeightedSelection implements the balance-weighted selection algorithm from the Gloas spec.
// It selects validators with probability proportional to their effective balance.
func computeBalanceWeightedSelection(
	state state.ReadOnlyBeaconState,
	candidates []primitives.ValidatorIndex,
	seed [32]byte,
	targetSize uint64,
) ([]primitives.ValidatorIndex, error) {
	total := uint64(len(candidates))
	if total == 0 {
		return nil, errors.Errorf("cannot compute balance with %d candidates", total)
	}

	selected := make([]primitives.ValidatorIndex, 0, targetSize)

	hashFunc := hash.CustomSHA256Hasher()
	seedBuffer := make([]byte, len(seed)+8)
	copy(seedBuffer, seed[:])
	maxEffectiveBalance := params.BeaconConfig().MaxEffectiveBalanceElectra

	i := uint64(0)
	for uint64(len(selected)) < targetSize {
		candidatePos := i % total
		candidateIndex := candidates[candidatePos]

		accepted, err := isValidatorAccepted(state, candidateIndex, seedBuffer, hashFunc, maxEffectiveBalance, i)
		if err != nil {
			return nil, errors.Wrapf(err, "could not check acceptance for validator %d", candidateIndex)
		}

		if accepted {
			selected = append(selected, candidateIndex)
		}
		i++
	}

	return selected, nil
}

// isValidatorAccepted determines if a validator should be accepted for the PTC
// based on their effective balance using the balance-weighted acceptance algorithm.
// Optimized version that can reuse pre-allocated buffers when called from loops.
func isValidatorAccepted(
	state state.ReadOnlyBeaconState,
	validatorIndex primitives.ValidatorIndex,
	seedBuffer []byte,
	hashFunc func([]byte) [32]byte,
	maxEffectiveBalance uint64,
	round uint64,
) (bool, error) {
	// Reuse the pre-allocated seed buffer, just update the round part
	binary.LittleEndian.PutUint64(seedBuffer[len(seedBuffer)-8:], round/16)
	randomByte := hashFunc(seedBuffer)

	offset := (round % 16) * 2
	randomValue := uint64(randomByte[offset]) | uint64(randomByte[offset+1])<<8

	validator, err := state.ValidatorAtIndex(validatorIndex)
	if err != nil {
		return false, errors.Wrapf(err, "could not get validator at index %d", validatorIndex)
	}

	effectiveBalance := validator.EffectiveBalance
	return effectiveBalance*fieldparams.MaxRandomValueElectra >= maxEffectiveBalance*randomValue, nil
}

// isValidIndexedPayloadAttestation validates an indexed payload attestation's signature.
// It checks that the attesting indices are sorted and verifies the aggregate signature
// against the public keys of the attesting validators.
func isValidIndexedPayloadAttestation(state state.ReadOnlyBeaconState, att *consensus_types.IndexedPayloadAttestation) (bool, error) {
	if state.Version() < version.Gloas {
		return false, ErrGloasNotActive
	}

	// Verify that indices are sorted and non-empty.
	indices := att.AttestingIndices
	if len(indices) == 0 {
		return false, nil
	}
	sortedIndices := make([]primitives.ValidatorIndex, len(indices))
	copy(sortedIndices, indices)
	slices.Sort(sortedIndices)
	if !slices.Equal(att.AttestingIndices, sortedIndices) {
		return false, nil
	}

	// Collect public keys for all attesting validators.
	publicKeys := make([]bls.PublicKey, len(indices))
	for i, validatorIndex := range indices {
		validator, err := state.ValidatorAtIndexReadOnly(validatorIndex)
		if err != nil {
			return false, errors.Wrapf(err, "could not get validator at index %d", validatorIndex)
		}

		publicKeyBytes := validator.PublicKey()
		publicKey, err := bls.PublicKeyFromBytes(publicKeyBytes[:])
		if err != nil {
			return false, errors.Wrapf(err, "could not parse public key for validator %d", validatorIndex)
		}

		publicKeys[i] = publicKey
	}

	// Compute the signing domain and root for verification.
	domain, err := signing.Domain(
		state.Fork(),
		slots.ToEpoch(state.Slot()),
		params.BeaconConfig().DomainPTCAttester,
		state.GenesisValidatorsRoot(),
	)
	if err != nil {
		return false, errors.Wrap(err, "could not compute signing domain")
	}

	signingRoot, err := signing.ComputeSigningRoot(att.Data, domain)
	if err != nil {
		return false, errors.Wrap(err, "could not compute signing root")
	}

	signature, err := bls.SignatureFromBytes(att.Signature)
	if err != nil {
		return false, errors.Wrap(err, "could not parse BLS signature")
	}

	return signature.FastAggregateVerify(publicKeys, signingRoot), nil
}
