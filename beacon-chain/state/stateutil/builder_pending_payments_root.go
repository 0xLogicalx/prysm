package stateutil

import (
	"github.com/OffchainLabs/prysm/v6/encoding/ssz"
	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
)

func BuilderPendingPaymentsRoot(slice []*ethpb.BuilderPendingPayment) ([32]byte, error) {
	roots := make([][32]byte, 64)
	for i := 0; i < len(slice) && i < 64; i++ {
		r, err := slice[i].HashTreeRoot()
		if err != nil {
			return [32]byte{}, err
		}
		roots[i] = r
	}
	return ssz.MerkleizeVector(roots, uint64(len(roots))), nil
}
