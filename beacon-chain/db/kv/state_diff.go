package kv

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/consensus-types/hdiff"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	bolt "go.etcd.io/bbolt"
)

var (
	offsetKey           = []byte("offset")
	ErrSlotBeforeOffset = errors.New("slot is before root offset")
	exponents           = []uint64{21, 18, 16, 13, 11, 9, 5} // TODO: should be taken from a config
	EmptyNodeMarker     = []byte{0x00}
)

/*
	We use a level-based approach to save state diffs. The levels are 0-6, where each level corresponds to an exponent of 2 (exponents[lvl]).
	The data at level 0 is saved every 2**exponent[0] slots and always contains a full state snapshot that is used as a base for the delta saved at other levels.
	We save a full tree, meaning that every slot has an entry in the 6th level. for this, sometimes we need to save nil entries on higher levels.
*/

// SaveStateDiff takes a state and decides between saving a full state snapshot or a diff.
func (s *Store) SaveStateDiff(ctx context.Context, state state.ReadOnlyBeaconState) error {
	slot := state.Slot()
	offset, err := s.loadOrInitOffset(slot)
	if err != nil {
		return err
	}
	if uint64(slot) < offset {
		return ErrSlotBeforeOffset
	}
	rel := uint64(slot) - offset

	lvl, shouldSave := computeLevel(rel)
	if !shouldSave {
		return nil
	}
	if lvl == 0 {
		if err = s.saveFullSnapshot(lvl, state); err != nil {
			return err
		}
	}
	// save diff

	return nil
}

// StateDiff retrieves the full state for a given slot.
func (s *Store) StateDiff(ctx context.Context, slot primitives.Slot) (state.BeaconState, error) {

	return nil, nil
}

func (s *Store) loadOrInitOffset(slot primitives.Slot) (uint64, error) {
	var offset uint64
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}

		offsetBytes := bucket.Get(offsetKey)
		if offsetBytes != nil {
			offset = binary.BigEndian.Uint64(offsetBytes)
			return nil
		}
		return ErrNotFound
	})

	if err == nil {
		return offset, nil
	}

	if !errors.Is(err, ErrNotFound) {
		return 0, err
	}

	// If the offset is not found, initialize it to the current slot.
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}

		offset = uint64(slot)
		offsetBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(offsetBytes, offset)
		if err := bucket.Put(offsetKey, offsetBytes); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return 0, err
	}
	return offset, nil
}

func computeLevel(rel uint64) (int, bool) {
	for i, exp := range exponents {
		span := uint64(1) << exp // 2^exp slots per interval at this level
		if rel%span == 0 {
			return i, true
		}
	}
	// If rel isn’t on any of the boundaries, we should ignore saving it.
	return -1, false
}

func (s *Store) saveHdiff(ctx context.Context, lvl int, hdiff hdiff.Hdiff) error { return nil }

func (s *Store) saveFullSnapshot(lvl int, state state.ReadOnlyBeaconState) error {
	slot := uint64(state.Slot())
	key := makeKey(lvl, slot)
	stateBytes, err := state.MarshalSSZ()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		if err := bucket.Put(key, stateBytes); err != nil {
			return err
		}

		// Save nil entries for higher levels
		for i := lvl + 1; i < len(exponents); i++ {
			key = makeKey(i, slot)
			if err = bucket.Put(key, EmptyNodeMarker); err != nil {
				return err
			}
		}

		return nil
	})
}

func makeKey(level int, slot uint64) []byte {
	buf := make([]byte, 1+8)
	buf[0] = byte(level)
	binary.BigEndian.PutUint64(buf[1:], slot)
	return buf
}
