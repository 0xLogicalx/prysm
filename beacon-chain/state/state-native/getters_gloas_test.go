package state_native_test

import (
	"bytes"
	"testing"

	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
)

func TestLatestBlockHash(t *testing.T) {
	t.Run("returns error before gloas", func(t *testing.T) {
		st, _ := util.DeterministicGenesisState(t, 1)
		_, err := st.LatestBlockHash()
		require.ErrorContains(t, "is not supported", err)
	})

	t.Run("returns zero hash when unset", func(t *testing.T) {
		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{})
		require.NoError(t, err)

		got, err := st.LatestBlockHash()
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("returns configured hash", func(t *testing.T) {
		hashBytes := bytes.Repeat([]byte{0xAB}, 32)
		var want [32]byte
		copy(want[:], hashBytes)

		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			LatestBlockHash: hashBytes,
		})
		require.NoError(t, err)

		got, err := st.LatestBlockHash()
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestPendingPaymentSum(t *testing.T) {
	t.Run("returns error before gloas", func(t *testing.T) {
		st, _ := util.DeterministicGenesisState(t, 1)
		_, err := st.PendingPaymentSum(1)
		require.ErrorContains(t, "is not supported", err)
	})

	t.Run("sums matching builder payments", func(t *testing.T) {
		const builder primitives.ValidatorIndex = 7
		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			BuilderPendingPayments: []*ethpb.BuilderPendingPayment{
				{
					Weight: 1,
					Withdrawal: &ethpb.BuilderPendingWithdrawal{
						Amount:       10,
						BuilderIndex: builder,
					},
				},
				{
					Weight: 1,
					Withdrawal: &ethpb.BuilderPendingWithdrawal{
						Amount:       20,
						BuilderIndex: 999,
					},
				},
				{
					Weight: 1,
					Withdrawal: &ethpb.BuilderPendingWithdrawal{
						Amount:       30,
						BuilderIndex: builder,
					},
				},
			},
		})
		require.NoError(t, err)

		got, err := st.PendingPaymentSum(builder)
		require.NoError(t, err)
		require.Equal(t, uint64(40), got)

		// No matching builder should return zero.
		got, err = st.PendingPaymentSum(primitives.ValidatorIndex(12345))
		require.NoError(t, err)
		require.Equal(t, uint64(0), got)
	})
}

func TestPendingWithdrawalSum(t *testing.T) {
	t.Run("returns error before gloas", func(t *testing.T) {
		st, _ := util.DeterministicGenesisState(t, 1)
		_, err := st.PendingWithdrawalSum(1)
		require.ErrorContains(t, "is not supported", err)
	})

	t.Run("sums matching builder withdrawals", func(t *testing.T) {
		const builder primitives.ValidatorIndex = 3
		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			BuilderPendingWithdrawals: []*ethpb.BuilderPendingWithdrawal{
				{
					Amount:       5,
					BuilderIndex: builder,
				},
				{
					Amount:       7,
					BuilderIndex: 99,
				},
				{
					Amount:       11,
					BuilderIndex: builder,
				},
			},
		})
		require.NoError(t, err)

		got, err := st.PendingWithdrawalSum(builder)
		require.NoError(t, err)
		require.Equal(t, uint64(16), got)

		got, err = st.PendingWithdrawalSum(primitives.ValidatorIndex(9999))
		require.NoError(t, err)
		require.Equal(t, uint64(0), got)
	})
}

func TestLatestExecutionPayloadBid(t *testing.T) {
	t.Run("returns error before gloas", func(t *testing.T) {
		st, _ := util.DeterministicGenesisState(t, 1)
		_, err := st.LatestExecutionPayloadBid()
		require.ErrorContains(t, "is not supported", err)
	})

	t.Run("nil when unset", func(t *testing.T) {
		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{})
		require.NoError(t, err)

		got, err := st.LatestExecutionPayloadBid()
		require.NoError(t, err)
		require.Equal(t, true, got == nil)
	})

	t.Run("wraps stored bid", func(t *testing.T) {
		bid := &ethpb.ExecutionPayloadBid{
			ParentBlockHash:        bytes.Repeat([]byte{0x01}, 32),
			ParentBlockRoot:        bytes.Repeat([]byte{0x02}, 32),
			BlockHash:              bytes.Repeat([]byte{0x03}, 32),
			PrevRandao:             bytes.Repeat([]byte{0x04}, 32),
			GasLimit:               1,
			BuilderIndex:           7,
			Slot:                   9,
			Value:                  11,
			ExecutionPayment:       12,
			BlobKzgCommitmentsRoot: bytes.Repeat([]byte{0x05}, 32),
			FeeRecipient:           bytes.Repeat([]byte{0x06}, 20),
		}
		st, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			LatestExecutionPayloadBid: bid,
		})
		require.NoError(t, err)

		got, err := st.LatestExecutionPayloadBid()
		require.NoError(t, err)
		require.Equal(t, primitives.ValidatorIndex(7), got.BuilderIndex())
		require.Equal(t, primitives.Gwei(11), got.Value())
		require.Equal(t, primitives.Gwei(12), got.ExecutionPayment())
	})
}
