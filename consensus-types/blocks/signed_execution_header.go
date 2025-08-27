package blocks

import (
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	consensus_types "github.com/OffchainLabs/prysm/v6/consensus-types"
	"github.com/OffchainLabs/prysm/v6/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// signedExecutionPayloadHeader wraps the protobuf signed execution payload header
// and implements the ROSignedExecutionPayloadHeader interface.
type signedExecutionPayloadHeader struct {
	header *ethpb.SignedExecutionPayloadHeader
}

// executionPayloadHeaderGloas wraps the protobuf execution payload header for Gloas fork
// and implements the ROExecutionPayloadHeaderGloas interface.
type executionPayloadHeaderGloas struct {
	payload *ethpb.ExecutionPayloadHeaderGloas
}

// IsNil checks if the signed execution payload header is nil or invalid.
func (s signedExecutionPayloadHeader) IsNil() bool {
	if s.header == nil {
		return true
	}

	// Check if the message can be wrapped as a valid header
	if _, err := WrappedROExecutionPayloadHeaderGloas(s.header.Message); err != nil {
		return true
	}

	// Signature must be exactly 96 bytes
	return len(s.header.Signature) != 96
}

// IsNil checks if the execution payload header is nil or has invalid fields.
func (h executionPayloadHeaderGloas) IsNil() bool {
	if h.payload == nil {
		return true
	}

	// All hash fields must be exactly 32 bytes
	if len(h.payload.ParentBlockHash) != 32 ||
		len(h.payload.ParentBlockRoot) != 32 ||
		len(h.payload.BlockHash) != 32 ||
		len(h.payload.BlobKzgCommitmentsRoot) != 32 {
		return true
	}

	return false
}

// WrappedROSignedExecutionPayloadHeader creates a new read-only signed execution payload header
// wrapper from the given protobuf message.
func WrappedROSignedExecutionPayloadHeader(pb *ethpb.SignedExecutionPayloadHeader) (interfaces.ROSignedExecutionPayloadHeader, error) {
	wrapper := signedExecutionPayloadHeader{header: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// WrappedROExecutionPayloadHeaderGloas creates a new read-only execution payload header
// wrapper for the Gloas fork from the given protobuf message.
func WrappedROExecutionPayloadHeaderGloas(pb *ethpb.ExecutionPayloadHeaderGloas) (interfaces.ROExecutionPayloadHeaderGloas, error) {
	wrapper := executionPayloadHeaderGloas{payload: pb}
	if wrapper.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return wrapper, nil
}

// Header returns the execution payload header as a read-only interface.
func (s signedExecutionPayloadHeader) Header() (interfaces.ROExecutionPayloadHeaderGloas, error) {
	return WrappedROExecutionPayloadHeaderGloas(s.header.Message)
}

// SigningRoot computes the signing root for the execution payload header with the given domain.
func (s signedExecutionPayloadHeader) SigningRoot(domain []byte) ([32]byte, error) {
	return signing.ComputeSigningRoot(s.header.Message, domain)
}

// Signature returns the BLS signature as a 96-byte array.
func (s signedExecutionPayloadHeader) Signature() [96]byte {
	return [96]byte(s.header.Signature)
}

// Execution Payload Header Gloas accessor methods

// ParentBlockHash returns the hash of the parent execution block.
func (h executionPayloadHeaderGloas) ParentBlockHash() [32]byte {
	return [32]byte(h.payload.ParentBlockHash)
}

// ParentBlockRoot returns the beacon block root of the parent block.
func (h executionPayloadHeaderGloas) ParentBlockRoot() [32]byte {
	return [32]byte(h.payload.ParentBlockRoot)
}

// BlockHash returns the hash of the execution block.
func (h executionPayloadHeaderGloas) BlockHash() [32]byte {
	return [32]byte(h.payload.BlockHash)
}

// GasLimit returns the gas limit for the execution block.
func (h executionPayloadHeaderGloas) GasLimit() uint64 {
	return h.payload.GasLimit
}

// BuilderIndex returns the validator index of the builder who created this header.
func (h executionPayloadHeaderGloas) BuilderIndex() primitives.ValidatorIndex {
	return h.payload.BuilderIndex
}

// Slot returns the beacon chain slot for which this header was created.
func (h executionPayloadHeaderGloas) Slot() primitives.Slot {
	return h.payload.Slot
}

// Value returns the payment value offered by the builder in Gwei.
func (h executionPayloadHeaderGloas) Value() primitives.Gwei {
	return primitives.Gwei(h.payload.Value)
}

// BlobKzgCommitmentsRoot returns the root of the KZG commitments for blobs.
func (h executionPayloadHeaderGloas) BlobKzgCommitmentsRoot() [32]byte {
	return [32]byte(h.payload.BlobKzgCommitmentsRoot)
}

// FeeRecipient returns the execution address that will receive the builder payment.
func (h executionPayloadHeaderGloas) FeeRecipient() [20]byte {
	return [20]byte(h.payload.FeeRecipient)
}

// Proto returns the underlying protobuf message.
func (h executionPayloadHeaderGloas) Proto() proto.Message {
	return h.payload
}
