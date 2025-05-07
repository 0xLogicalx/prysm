package kv

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/math"
	"go.etcd.io/bbolt"
)

var (
	offsetKey           = []byte("offset")
	ErrSlotBeforeOffset = errors.New("slot is before root offset")
)

func makeKey(level int, slot uint64) []byte {
	buf := make([]byte, 1+8)
	buf[0] = byte(level)
	binary.BigEndian.PutUint64(buf[1:], slot)
	return buf
}

func getAnchorState(s *Store, offset uint64, lvl int, slot primitives.Slot) (anchor state.ReadOnlyBeaconState, err error) {
	if lvl == 0 {
		return nil, errors.New("no anchor for level 0")
	}

	relSlot := uint64(slot) - offset
	prevExp := exponents[lvl-1]
	span := math.PowerOf2(prevExp)
	anchorSlot := primitives.Slot((relSlot / span * span) + offset)

	anchorLvl := computeLevel(offset, anchorSlot)
	if anchorLvl == -1 {
		return nil, errors.New("could not compute anchor level")
	}

	// Check if we have the anchor in cache.
	anchor, ok := anchorCache[anchorLvl]
	if ok {
		return anchor, nil
	}

	// If not, load it from the database.
	anchor, err = s.StateDiff(context.Background(), anchorSlot)
	if err != nil {
		return nil, err
	}

	// Save it in the cache.
	anchorCache[anchorLvl] = anchor
	return anchor, nil
}

// ComputeLevel computes the level in the diff tree. Returns -1 in case slot should not be in tree.
func computeLevel(offset uint64, slot primitives.Slot) int {
	rel := uint64(slot) - offset
	for i, exp := range exponents {
		span := math.PowerOf2(exp)
		if rel%span == 0 {
			return i
		}
	}
	// If rel isn’t on any of the boundaries, we should ignore saving it.
	return -1
}

func loadOrInitOffset(s *Store, slot primitives.Slot) (offset uint64, err error) {
	return offset, s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bbolt.ErrBucketNotFound
		}

		offsetBytes := bucket.Get(offsetKey)
		if offsetBytes != nil {
			offset = binary.BigEndian.Uint64(offsetBytes)
			return nil
		}

		offset = uint64(slot)
		offsetBytes = make([]byte, 8)
		binary.BigEndian.PutUint64(offsetBytes, offset)
		if err := bucket.Put(offsetKey, offsetBytes); err != nil {
			return err
		}
		return nil
	})
}

func getOffset(s *Store) (offset uint64, err error) {
	return offset, s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bbolt.ErrBucketNotFound
		}

		offsetBytes := bucket.Get(offsetKey)
		if offsetBytes != nil {
			offset = binary.BigEndian.Uint64(offsetBytes)
			return nil
		}
		return bbolt.ErrIncompatibleValue
	})
}
