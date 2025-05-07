package kv

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/math"
	"github.com/OffchainLabs/prysm/v6/monitoring/tracing/trace"
	bolt "go.etcd.io/bbolt"
)

var (
	offsetKey           = []byte("offset")
	ErrSlotBeforeOffset = errors.New("slot is before root offset")
	exponents           = []uint64{21, 18, 16, 13, 11, 9, 5} // TODO: should be taken from a config
	EmptyNodeMarker     = []byte{0x00}
	snapshotCache       = make(map[int]state.ReadOnlyBeaconState, len(exponents)) // cache full states at the last node at each level
)

/*
	We use a level-based approach to save state diffs. The levels are 0-6, where each level corresponds to an exponent of 2 (exponents[lvl]).
	The data at level 0 is saved every 2**exponent[0] slots and always contains a full state snapshot that is used as a base for the delta saved at other levels.
*/

// SaveStateDiff takes a state and decides between saving a full state snapshot or a diff.
func (s *Store) SaveStateDiff(ctx context.Context, st state.ReadOnlyBeaconState) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateDiff")
	defer span.End()

	slot := st.Slot()
	offset, err := s.loadOrInitOffset(slot)
	if err != nil {
		return err
	}
	if uint64(slot) < offset {
		return ErrSlotBeforeOffset
	}
	rel := uint64(slot) - offset

	// Find the level to save the state.
	lvl, shouldSave := computeLevel(rel)
	if !shouldSave {
		return nil
	}

	// Save full state if level is 0.
	if lvl == 0 {
		return s.saveFullSnapshot(lvl, st)
	}

	// Get anchor state to compute the diff from.
	anchorState, err := getAnchorState(lvl, rel)
	if err != nil {
		return err
	}

	err = s.saveHdiff(lvl, anchorState, st)
	if err != nil {
		return err
	}

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
		span := math.PowerOf2(exp)
		if rel%span == 0 {
			return i, true
		}
	}
	// If rel isn’t on any of the boundaries, we should ignore saving it.
	return -1, false
}

func (s *Store) saveHdiff(lvl int, anchor, st state.ReadOnlyBeaconState) error {
	slot := uint64(st.Slot())
	key := makeKey(lvl, slot)

	// TODO: compute the diff here

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}
		// TODO: save the diff bytes
		if err := bucket.Put(key, nil); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	snapshotCache[lvl] = st
	return nil
}

func (s *Store) saveFullSnapshot(lvl int, st state.ReadOnlyBeaconState) error {
	slot := uint64(st.Slot())
	key := makeKey(lvl, slot)
	stateBytes, err := st.MarshalSSZ()
	if err != nil {
		return err
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bolt.ErrBucketNotFound
		}

		if err := bucket.Put(key, stateBytes); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	snapshotCache[lvl] = st
	return nil
}

func makeKey(level int, slot uint64) []byte {
	buf := make([]byte, 1+8)
	buf[0] = byte(level)
	binary.BigEndian.PutUint64(buf[1:], slot)
	return buf
}

func getAnchorState(lvl int, rel uint64) (state.ReadOnlyBeaconState, error) {
	if lvl == 0 {
		return nil, errors.New("no anchor for level 0")
	}

	prevExp := exponents[lvl-1]
	span := math.PowerOf2(prevExp)
	anchorRel := rel / span * span
	anchorLvl, _ := computeLevel(anchorRel)
	if anchorLvl == -1 {
		return nil, errors.New("could not compute anchor level")
	}
	// TODO: check if the anchor state is in the cache
	anchor, ok := snapshotCache[anchorLvl]
	if ok {
		return anchor, nil
	}
	anchorSlot := getAnchorSlot(rel)
}

func getAnchorSlot(rel uint64) uint64 {

}
