package operations

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/runtime/version"
	common "github.com/OffchainLabs/prysm/v6/testing/spectest/shared/common/operations"
)

func RunPayloadAttestationTest(t *testing.T, config string) {
	// Pass nil since payload attestation tests don't use the block parameter
	common.RunPayloadAttestationTest(t, config, version.String(version.Gloas), nil, sszToState)
}
