package gloas

import (
	"bytes"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/crypto/bls/common"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	validatorpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1/validator-client"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	fastssz "github.com/prysmaticlabs/fastssz"
	"google.golang.org/protobuf/proto"
)

type stubBlockBody struct {
	signedBid *ethpb.SignedExecutionPayloadBid
}

func (s stubBlockBody) Version() int                                 { return version.Gloas }
func (s stubBlockBody) RandaoReveal() [96]byte                       { return [96]byte{} }
func (s stubBlockBody) Eth1Data() *ethpb.Eth1Data                    { return nil }
func (s stubBlockBody) Graffiti() [32]byte                           { return [32]byte{} }
func (s stubBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing { return nil }
func (s stubBlockBody) AttesterSlashings() []ethpb.AttSlashing       { return nil }
func (s stubBlockBody) Attestations() []ethpb.Att                    { return nil }
func (s stubBlockBody) Deposits() []*ethpb.Deposit                   { return nil }
func (s stubBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit { return nil }
func (s stubBlockBody) SyncAggregate() (*ethpb.SyncAggregate, error) { return nil, nil }
func (s stubBlockBody) IsNil() bool                                  { return s.signedBid == nil }
func (s stubBlockBody) HashTreeRoot() ([32]byte, error)              { return [32]byte{}, nil }
func (s stubBlockBody) Proto() (proto.Message, error)                { return nil, nil }
func (s stubBlockBody) Execution() (interfaces.ExecutionData, error) { return nil, nil }
func (s stubBlockBody) BLSToExecutionChanges() ([]*ethpb.SignedBLSToExecutionChange, error) {
	return nil, nil
}
func (s stubBlockBody) BlobKzgCommitments() ([][]byte, error) { return nil, nil }
func (s stubBlockBody) ExecutionRequests() (*enginev1.ExecutionRequests, error) {
	return nil, nil
}
func (s stubBlockBody) PayloadAttestations() ([]*ethpb.PayloadAttestation, error) {
	return nil, nil
}
func (s stubBlockBody) SignedExecutionPayloadBid() (*ethpb.SignedExecutionPayloadBid, error) {
	return s.signedBid, nil
}
func (s stubBlockBody) MarshalSSZ() ([]byte, error)         { return nil, nil }
func (s stubBlockBody) MarshalSSZTo([]byte) ([]byte, error) { return nil, nil }
func (s stubBlockBody) UnmarshalSSZ([]byte) error           { return nil }
func (s stubBlockBody) SizeSSZ() int                        { return 0 }

type stubBlock struct {
	slot       primitives.Slot
	proposer   primitives.ValidatorIndex
	parentRoot [32]byte
	body       stubBlockBody
	v          int
}

func (s stubBlock) Slot() primitives.Slot                    { return s.slot }
func (s stubBlock) ProposerIndex() primitives.ValidatorIndex { return s.proposer }
func (s stubBlock) ParentRoot() [32]byte                     { return s.parentRoot }
func (s stubBlock) StateRoot() [32]byte                      { return [32]byte{} }
func (s stubBlock) Body() interfaces.ReadOnlyBeaconBlockBody { return s.body }
func (s stubBlock) IsNil() bool                              { return false }
func (s stubBlock) IsBlinded() bool                          { return false }
func (s stubBlock) HashTreeRoot() ([32]byte, error)          { return [32]byte{}, nil }
func (s stubBlock) Proto() (proto.Message, error)            { return nil, nil }
func (s stubBlock) MarshalSSZ() ([]byte, error)              { return nil, nil }
func (s stubBlock) MarshalSSZTo([]byte) ([]byte, error)      { return nil, nil }
func (s stubBlock) UnmarshalSSZ([]byte) error                { return nil }
func (s stubBlock) SizeSSZ() int                             { return 0 }
func (s stubBlock) Version() int                             { return s.v }
func (s stubBlock) AsSignRequestObject() (validatorpb.SignRequestObject, error) {
	return nil, nil
}
func (s stubBlock) HashTreeRootWith(*fastssz.Hasher) error { return nil }

func buildGloasState(t *testing.T, slot primitives.Slot, builderIdx primitives.ValidatorIndex, balance uint64, randao [32]byte, latestHash [32]byte, pubkeyBytes [48]byte) *state_native.BeaconState {
	t.Helper()

	cfg := params.BeaconConfig()
	blockRoots := make([][]byte, cfg.SlotsPerHistoricalRoot)
	stateRoots := make([][]byte, cfg.SlotsPerHistoricalRoot)
	for i := range blockRoots {
		blockRoots[i] = bytes.Repeat([]byte{0xAA}, 32)
		stateRoots[i] = bytes.Repeat([]byte{0xBB}, 32)
	}
	randaoMixes := make([][]byte, cfg.EpochsPerHistoricalVector)
	for i := range randaoMixes {
		randaoMixes[i] = randao[:]
	}

	withdrawalCreds := make([]byte, 32)
	withdrawalCreds[0] = cfg.BuilderWithdrawalPrefixByte

	validatorCount := int(builderIdx) + 1
	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := range validatorCount {
		validators[i] = &ethpb.Validator{
			PublicKey:                  pubkeyBytes[:],
			WithdrawalCredentials:      withdrawalCreds,
			EffectiveBalance:           balance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 0,
			ActivationEpoch:            0,
			ExitEpoch:                  cfg.FarFutureEpoch,
			WithdrawableEpoch:          cfg.FarFutureEpoch,
		}
		balances[i] = balance
	}

	payments := make([]*ethpb.BuilderPendingPayment, cfg.SlotsPerEpoch*2)
	for i := range payments {
		payments[i] = &ethpb.BuilderPendingPayment{Withdrawal: &ethpb.BuilderPendingWithdrawal{}}
	}

	stProto := &ethpb.BeaconStateGloas{
		Slot:                  slot,
		GenesisValidatorsRoot: bytes.Repeat([]byte{0x11}, 32),
		Fork: &ethpb.Fork{
			CurrentVersion:  bytes.Repeat([]byte{0x22}, 4),
			PreviousVersion: bytes.Repeat([]byte{0x22}, 4),
			Epoch:           0,
		},
		BlockRoots:                blockRoots,
		StateRoots:                stateRoots,
		RandaoMixes:               randaoMixes,
		Validators:                validators,
		Balances:                  balances,
		LatestBlockHash:           latestHash[:],
		BuilderPendingPayments:    payments,
		BuilderPendingWithdrawals: []*ethpb.BuilderPendingWithdrawal{},
	}

	st, err := state_native.InitializeFromProtoGloas(stProto)
	require.NoError(t, err)
	return st.(*state_native.BeaconState)
}

func signBid(t *testing.T, sk common.SecretKey, bid *ethpb.ExecutionPayloadBid, fork *ethpb.Fork, genesisRoot [32]byte) [96]byte {
	t.Helper()
	epoch := slots.ToEpoch(primitives.Slot(bid.Slot))
	domain, err := signing.Domain(fork, epoch, params.BeaconConfig().DomainBeaconBuilder, genesisRoot[:])
	require.NoError(t, err)
	root, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sig := sk.Sign(root[:]).Marshal()
	var out [96]byte
	copy(out[:], sig)
	return out
}

func TestProcessExecutionPayloadBid_SelfBuildSuccess(t *testing.T) {
	slot := primitives.Slot(12)
	builderIdx := primitives.ValidatorIndex(0)
	randao := [32]byte(bytes.Repeat([]byte{0xAA}, 32))
	latestHash := [32]byte(bytes.Repeat([]byte{0xBB}, 32))
	pubKey := [48]byte{}
	state := buildGloasState(t, slot, builderIdx, params.BeaconConfig().MinActivationBalance+1000, randao, latestHash, pubKey)

	bid := &ethpb.ExecutionPayloadBid{
		ParentBlockHash:        latestHash[:],
		ParentBlockRoot:        bytes.Repeat([]byte{0xCC}, 32),
		BlockHash:              bytes.Repeat([]byte{0xDD}, 32),
		PrevRandao:             randao[:],
		GasLimit:               1,
		BuilderIndex:           builderIdx,
		Slot:                   slot,
		Value:                  0,
		ExecutionPayment:       0,
		BlobKzgCommitmentsRoot: bytes.Repeat([]byte{0xEE}, 32),
		FeeRecipient:           bytes.Repeat([]byte{0xFF}, 20),
	}
	signed := &ethpb.SignedExecutionPayloadBid{
		Message:   bid,
		Signature: common.InfiniteSignature[:],
	}

	block := stubBlock{
		slot:       slot,
		proposer:   builderIdx,
		parentRoot: bytesutil.ToBytes32(bid.ParentBlockRoot),
		body:       stubBlockBody{signedBid: signed},
		v:          version.Gloas,
	}

	require.NoError(t, ProcessExecutionPayloadBid(state, block))
	sum, err := state.PendingPaymentSum(builderIdx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), sum)
}

func TestProcessExecutionPayloadBid_SelfBuildNonZeroAmountFails(t *testing.T) {
	slot := primitives.Slot(2)
	builderIdx := primitives.ValidatorIndex(0)
	randao := [32]byte{}
	latestHash := [32]byte{1}
	state := buildGloasState(t, slot, builderIdx, params.BeaconConfig().MinActivationBalance+1000, randao, latestHash, [48]byte{})

	bid := &ethpb.ExecutionPayloadBid{
		ParentBlockHash:        latestHash[:],
		ParentBlockRoot:        bytes.Repeat([]byte{0xAA}, 32),
		BlockHash:              bytes.Repeat([]byte{0xBB}, 32),
		PrevRandao:             randao[:],
		BuilderIndex:           builderIdx,
		Slot:                   slot,
		Value:                  10,
		ExecutionPayment:       0,
		BlobKzgCommitmentsRoot: bytes.Repeat([]byte{0xCC}, 32),
		FeeRecipient:           bytes.Repeat([]byte{0xDD}, 20),
	}
	signed := &ethpb.SignedExecutionPayloadBid{
		Message:   bid,
		Signature: common.InfiniteSignature[:],
	}
	block := stubBlock{
		slot:       slot,
		proposer:   builderIdx,
		parentRoot: bytesutil.ToBytes32(bid.ParentBlockRoot),
		body:       stubBlockBody{signedBid: signed},
		v:          version.Gloas,
	}

	err := ProcessExecutionPayloadBid(state, block)
	require.ErrorContains(t, "self-build amount must be zero", err)
}

func TestProcessExecutionPayloadBid_PendingPaymentAndCacheBid(t *testing.T) {
	slot := primitives.Slot(8)
	builderIdx := primitives.ValidatorIndex(1)
	randao := [32]byte(bytes.Repeat([]byte{0xAA}, 32))
	latestHash := [32]byte(bytes.Repeat([]byte{0xBB}, 32))

	sk, err := bls.RandKey()
	require.NoError(t, err)
	pub := sk.PublicKey().Marshal()
	var pubKey [48]byte
	copy(pubKey[:], pub)

	balance := params.BeaconConfig().MinActivationBalance + 1_000_000
	state := buildGloasState(t, slot, builderIdx, balance, randao, latestHash, pubKey)

	bid := &ethpb.ExecutionPayloadBid{
		ParentBlockHash:        latestHash[:],
		ParentBlockRoot:        bytes.Repeat([]byte{0xCC}, 32),
		BlockHash:              bytes.Repeat([]byte{0xDD}, 32),
		PrevRandao:             randao[:],
		GasLimit:               1,
		BuilderIndex:           builderIdx,
		Slot:                   slot,
		Value:                  500_000,
		ExecutionPayment:       1,
		BlobKzgCommitmentsRoot: bytes.Repeat([]byte{0xEE}, 32),
		FeeRecipient:           bytes.Repeat([]byte{0xFF}, 20),
	}

	genesis := bytesutil.ToBytes32(state.GenesisValidatorsRoot())
	sig := signBid(t, sk, bid, state.Fork(), genesis)
	signed := &ethpb.SignedExecutionPayloadBid{
		Message:   bid,
		Signature: sig[:],
	}

	block := stubBlock{
		slot:       slot,
		proposer:   builderIdx + 1, // not self-build
		parentRoot: bytesutil.ToBytes32(bid.ParentBlockRoot),
		body:       stubBlockBody{signedBid: signed},
		v:          version.Gloas,
	}

	require.NoError(t, ProcessExecutionPayloadBid(state, block))

	paymentSum, err := state.PendingPaymentSum(builderIdx)
	require.NoError(t, err)
	require.Equal(t, uint64(500_000), paymentSum)

	gotBid, err := state.LatestExecutionPayloadBid()
	require.NoError(t, err)
	require.Equal(t, primitives.ValidatorIndex(1), gotBid.BuilderIndex())
	require.Equal(t, primitives.Gwei(500_000), gotBid.Value())
}
