package light_client

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/blockchain"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	state_native "github.com/OffchainLabs/prysm/v6/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v6/consensus-types/blocks"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/OffchainLabs/prysm/v6/testing/require"
	"github.com/OffchainLabs/prysm/v6/testing/spectest/utils"
	"github.com/OffchainLabs/prysm/v6/testing/util"
	"github.com/golang/snappy"
)

// RunLightClientDataCollectionTests executes "light_client/data_collection/pyspec_tests/light_client_data_collection" tests.
func RunLightClientDataCollectionTests(t *testing.T, config string, v int) {
	require.NoError(t, utils.SetConfig(t, config))

	_, testsFolderPath := utils.TestFolders(t, config, version.String(v), "light_client/data_collection/pyspec_tests/")
	testTypes, err := util.BazelListDirectories(testsFolderPath)
	require.NoError(t, err)

	if len(testTypes) == 0 {
		t.Fatalf("No test types found for %s", testsFolderPath)
	}
	if testTypes[0] != "light_client_data_collection" {
		t.Fatalf("Expected test type 'light_client_data_collection', got %s", testTypes[0])
	}

	_, testsFolderPath = utils.TestFolders(t, config, version.String(v), "light_client/data_collection/pyspec_tests/light_client_data_collection")
	helpers.ClearCache()
	t.Run("data collection", func(t *testing.T) {
		runLightClientDataCollectionTest(t, testsFolderPath, v)
	})
}

func runLightClientDataCollectionTest(t *testing.T, testFolderPath string, v int) {
	stepsFile, err := util.BazelFileBytes(path.Join(testFolderPath, "steps.yaml"))
	require.NoError(t, err)

	var steps []Step
	require.NoError(t, utils.UnmarshalYaml(stepsFile, &steps))

	preBeaconStateFile, err := util.BazelFileBytes(path.Join(testFolderPath, "initial_state.ssz_snappy"))
	require.NoError(t, err)
	preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
	require.NoError(t, err, "Failed to decompress")
	beaconStateBase := &ethpb.BeaconStateAltair{}
	require.NoError(t, beaconStateBase.UnmarshalSSZ(preBeaconStateSSZ), "Failed to unmarshal")
	beaconState, err := state_native.InitializeFromProtoAltair(beaconStateBase)
	require.NoError(t, err)

	service, err := blockchain.NewService(context.Background(), blockchain.WithFinalizedStateAtStartUp(beaconState))
	require.NoError(t, err)

	for _, step := range steps {
		if step.NewBlock != nil && step.NewHead == nil {
			fmt.Println("block")
			blkFile, err := util.BazelFileBytes(path.Join(testFolderPath, step.NewBlock.Data))
			require.NoError(t, err)
			blockSSZ, err := snappy.Decode(nil /* dst */, blkFile)
			require.NoError(t, err, "Failed to decompress")
			block := &ethpb.SignedBeaconBlockAltair{}
			require.NoError(t, block.UnmarshalSSZ(blockSSZ), "Failed to unmarshal")
			wsb, err := blocks.NewSignedBeaconBlock(block)
			require.NoError(t, err)

		} else if step.NewHead != nil && step.NewBlock == nil {
			fmt.Println("head")
		} else {
			t.Fatalf("Unexpected step")
		}
	}
}
