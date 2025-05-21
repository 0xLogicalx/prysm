package light_client

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/OffchainLabs/prysm/v6/testing/require"
	"github.com/OffchainLabs/prysm/v6/testing/spectest/utils"
	"github.com/OffchainLabs/prysm/v6/testing/util"
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

func runLightClientDataCollectionTest(t *testing.T, path string, v int) {

}
