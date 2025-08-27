package minimal

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/testing/spectest/shared/gloas/operations"
)

func TestMinimal_Gloas_Operations_ExecutionPayloadHeader(t *testing.T) {
	operations.RunExecutionPayloadHeaderTest(t, "minimal")
}
