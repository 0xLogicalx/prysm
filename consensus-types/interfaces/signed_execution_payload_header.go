package interfaces

import (
	field_params "github.com/OffchainLabs/prysm/v6/config/fieldparams"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"google.golang.org/protobuf/proto"
)

type ROSignedExecutionPayloadHeader interface {
	Header() (ROExecutionPayloadHeaderGloas, error)
	Signature() [field_params.BLSSignatureLength]byte
	SigningRoot([]byte) ([32]byte, error)
	IsNil() bool
}

type ROExecutionPayloadHeaderGloas interface {
	ParentBlockHash() [32]byte
	ParentBlockRoot() [32]byte
	BlockHash() [32]byte
	GasLimit() uint64
	BuilderIndex() primitives.ValidatorIndex
	Slot() primitives.Slot
	Value() primitives.Gwei
	BlobKzgCommitmentsRoot() [32]byte
	FeeRecipient() [20]byte
	IsNil() bool
	Proto() proto.Message
}
