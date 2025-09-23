package gloas

import (
	"bytes"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/OffchainLabs/prysm/v6/time/slots"
)

// SameSlotAttestation checks if the attestation is for the same slot as the block root in the state.
func SameSlotAttestation(state state.ReadOnlyBeaconState, blockRoot [32]byte, slot primitives.Slot) (bool, error) {
	if slot == 0 {
		return true, nil
	}

	blockRootAtSlot, err := helpers.BlockRootAtSlot(state, slot)
	if err != nil {
		return false, err
	}

	isMatchingBlockRoot := bytes.Equal(blockRoot[:], blockRootAtSlot)

	blockRootAtPrevSlot, err := helpers.BlockRootAtSlot(state, slot-1)
	if err != nil {
		return false, err
	}

	isCurrentBlockRoot := !bytes.Equal(blockRoot[:], blockRootAtPrevSlot)

	return isMatchingBlockRoot && isCurrentBlockRoot, nil
}

// UpdatePendingPaymentWeight updates the builder pending payment weight based on attestation participation.
func UpdatePendingPaymentWeight(beaconState state.BeaconState, att ethpb.Att, indices []uint64, participatedFlags map[uint8]bool) (state.BeaconState, error) {
	if beaconState.Version() < version.Gloas {
		return beaconState, nil
	}

	data := att.GetData()
	currentEpoch := slots.ToEpoch(beaconState.Slot())

	isSameSlot, err := SameSlotAttestation(beaconState, [32]byte(data.BeaconBlockRoot), data.Slot)
	if err != nil {
		return nil, err
	}
	if !isSameSlot {
		return beaconState, nil
	}

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	var paymentSlot primitives.Slot
	if data.Target.Epoch == currentEpoch {
		paymentSlot = slotsPerEpoch + (data.Slot % slotsPerEpoch)
	} else {
		paymentSlot = data.Slot % slotsPerEpoch
	}

	payment, err := beaconState.BuilderPendingPayment(uint64(paymentSlot))
	if err != nil {
		return nil, err
	}

	var epochParticipation []byte
	if data.Target.Epoch == currentEpoch {
		epochParticipation, err = beaconState.CurrentEpochParticipation()
	} else {
		epochParticipation, err = beaconState.PreviousEpochParticipation()
	}
	if err != nil {
		return nil, err
	}

	cfg := params.BeaconConfig()
	flagIndices := []uint8{cfg.TimelySourceFlagIndex, cfg.TimelyTargetFlagIndex, cfg.TimelyHeadFlagIndex}
	for _, index := range indices {
		willSetNewFlag := false
		for _, flagIndex := range flagIndices {
			if participatedFlags[flagIndex] {
				has := ((epochParticipation[index] >> flagIndex) & 1) == 1
				if !has {
					willSetNewFlag = true
					break
				}
			}
		}

		if willSetNewFlag {
			validator, err := beaconState.ValidatorAtIndex(primitives.ValidatorIndex(index))
			if err != nil {
				return nil, err
			}
			payment.Weight += primitives.Gwei(validator.EffectiveBalance)
		}
	}

	if err := beaconState.SetBuilderPendingPayment(uint64(paymentSlot), payment); err != nil {
		return nil, err
	}

	return beaconState, nil
}
