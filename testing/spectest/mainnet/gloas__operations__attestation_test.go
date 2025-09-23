package mainnet

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/testing/spectest/shared/gloas/operations"
)

func TestMainnet_Gloas_Operations_Attestation(t *testing.T) {
	operations.RunAttestationTest(t, "mainnet")
}
