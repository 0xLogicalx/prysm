package operations

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	common "github.com/OffchainLabs/prysm/v6/testing/spectest/shared/common/operations"
)

func blockWithSignedExecutionPayloadHeader(blockSSZ []byte) (interfaces.SignedBeaconBlock, error) {
	var block ethpb.BeaconBlockGloas
	if err := block.UnmarshalSSZ(blockSSZ); err != nil {
		return nil, err
	}
	return blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockGloas{Block: &block})
}

func RunExecutionPayloadHeaderTest(t *testing.T, config string) {
	common.RunExecutionPayloadHeaderTest(t, config, version.String(version.Gloas), blockWithSignedExecutionPayloadHeader, sszToState)
}
