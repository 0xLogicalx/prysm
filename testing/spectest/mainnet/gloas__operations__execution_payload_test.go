package mainnet

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/testing/spectest/shared/gloas/operations"
)

func TestMainnet_Gloas_Operations_ExecutionPayloadEnvelope(t *testing.T) {
	operations.RunExecutionPayloadTest(t, "mainnet")
}
