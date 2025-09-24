package gloas

import (
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	state_native "github.com/OffchainLabs/prysm/v6/beacon-chain/state/state-native"
	fieldparams "github.com/OffchainLabs/prysm/v6/config/fieldparams"
	"github.com/OffchainLabs/prysm/v6/config/params"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/pkg/errors"
)

func ConvertToGloas(beaconState state.BeaconState) (state.BeaconState, error) {
	s, err := convertToGloasPB(beaconState)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert to gloas pb")
	}
	// Initialize Gloas-specific fields
	s.ExecutionPayloadAvailability = make([]byte, fieldparams.BlockRootsLength/8)
	s.BuilderPendingPayments = make([]*ethpb.BuilderPendingPayment, 2*fieldparams.SlotsPerEpoch)
	s.BuilderPendingWithdrawals = make([]*ethpb.BuilderPendingWithdrawal, 0)
	s.LatestBlockHash = make([]byte, 32)
	s.LatestWithdrawalsRoot = make([]byte, 32)

	post, err := state_native.InitializeFromProtoUnsafeGloas(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize post gloas beaconState")
	}
	return post, nil
}

func convertToGloasPB(beaconState state.BeaconState) (*ethpb.BeaconStateGloas, error) {
	currentSyncCommittee, err := beaconState.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSyncCommittee, err := beaconState.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	prevEpochParticipation, err := beaconState.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	currentEpochParticipation, err := beaconState.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	inactivityScores, err := beaconState.InactivityScores()
	if err != nil {
		return nil, err
	}
	wi, err := beaconState.NextWithdrawalIndex()
	if err != nil {
		return nil, err
	}
	vi, err := beaconState.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, err
	}
	summaries, err := beaconState.HistoricalSummaries()
	if err != nil {
		return nil, err
	}
	depositRequestsStartIndex, err := beaconState.DepositRequestsStartIndex()
	if err != nil {
		return nil, err
	}
	depositBalanceToConsume, err := beaconState.DepositBalanceToConsume()
	if err != nil {
		return nil, err
	}
	exitBalanceToConsume, err := beaconState.ExitBalanceToConsume()
	if err != nil {
		return nil, err
	}
	earliestExitEpoch, err := beaconState.EarliestExitEpoch()
	if err != nil {
		return nil, err
	}
	consolidationBalanceToConsume, err := beaconState.ConsolidationBalanceToConsume()
	if err != nil {
		return nil, err
	}
	earliestConsolidationEpoch, err := beaconState.EarliestConsolidationEpoch()
	if err != nil {
		return nil, err
	}
	pendingDeposits, err := beaconState.PendingDeposits()
	if err != nil {
		return nil, err
	}
	pendingPartialWithdrawals, err := beaconState.PendingPartialWithdrawals()
	if err != nil {
		return nil, err
	}
	pendingConsolidations, err := beaconState.PendingConsolidations()
	if err != nil {
		return nil, err
	}
	proposerLookaheadIndices, err := beaconState.ProposerLookahead()
	if err != nil {
		return nil, err
	}
	proposerLookahead := make([]uint64, len(proposerLookaheadIndices))
	for i, idx := range proposerLookaheadIndices {
		proposerLookahead[i] = uint64(idx)
	}

	s := &ethpb.BeaconStateGloas{
		GenesisTime:           uint64(beaconState.GenesisTime().Unix()),
		GenesisValidatorsRoot: beaconState.GenesisValidatorsRoot(),
		Slot:                  beaconState.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: beaconState.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().GloasForkVersion,
			Epoch:           time.CurrentEpoch(beaconState),
		},
		LatestBlockHeader:            beaconState.LatestBlockHeader(),
		BlockRoots:                   beaconState.BlockRoots(),
		StateRoots:                   beaconState.StateRoots(),
		HistoricalRoots:              beaconState.HistoricalRoots(),
		Eth1Data:                     beaconState.Eth1Data(),
		Eth1DataVotes:                beaconState.Eth1DataVotes(),
		Eth1DepositIndex:             beaconState.Eth1DepositIndex(),
		Validators:                   beaconState.Validators(),
		Balances:                     beaconState.Balances(),
		RandaoMixes:                  beaconState.RandaoMixes(),
		Slashings:                    beaconState.Slashings(),
		PreviousEpochParticipation:   prevEpochParticipation,
		CurrentEpochParticipation:    currentEpochParticipation,
		JustificationBits:            beaconState.JustificationBits(),
		PreviousJustifiedCheckpoint:  beaconState.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:   beaconState.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:          beaconState.FinalizedCheckpoint(),
		InactivityScores:             inactivityScores,
		CurrentSyncCommittee:         currentSyncCommittee,
		NextSyncCommittee:            nextSyncCommittee,
		LatestExecutionPayloadBid:    nil, // Not present in pre-Gloas states
		NextWithdrawalIndex:          wi,
		NextWithdrawalValidatorIndex: vi,
		HistoricalSummaries:          summaries,

		DepositRequestsStartIndex:     depositRequestsStartIndex,
		DepositBalanceToConsume:       depositBalanceToConsume,
		ExitBalanceToConsume:          exitBalanceToConsume,
		EarliestExitEpoch:             earliestExitEpoch,
		ConsolidationBalanceToConsume: consolidationBalanceToConsume,
		EarliestConsolidationEpoch:    earliestConsolidationEpoch,
		PendingDeposits:               pendingDeposits,
		PendingPartialWithdrawals:     pendingPartialWithdrawals,
		PendingConsolidations:         pendingConsolidations,
		ProposerLookahead:             proposerLookahead,
	}
	return s, nil
}
