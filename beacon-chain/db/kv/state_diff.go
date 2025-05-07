package kv

import (
	"context"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/monitoring/tracing/trace"
	bolt "go.etcd.io/bbolt"
)

/*
	We use a level-based approach to save state diffs. The levels are 0-6, where each level corresponds to an exponent of 2 (exponents[lvl]).
	The data at level 0 is saved every 2**exponent[0] slots and always contains a full state snapshot that is used as a base for the delta saved at other levels.
*/

var (
	exponents   = []uint64{21, 18, 16, 13, 11, 9, 5}                      // TODO: should be taken from a config
	anchorCache = make(map[int]state.ReadOnlyBeaconState, len(exponents)) // cache full states at the last node at each level
)

// SaveStateDiff takes a state and decides between saving a full state snapshot or a diff.
func (s *Store) SaveStateDiff(ctx context.Context, st state.ReadOnlyBeaconState) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateDiff")
	defer span.End()

	slot := st.Slot()
	offset, err := loadOrInitOffset(s, slot)
	if err != nil {
		return err
	}
	if uint64(slot) < offset {
		return ErrSlotBeforeOffset
	}

	// Find the level to save the state.
	lvl := computeLevel(offset, slot)
	if lvl == -1 {
		return nil
	}

	// Save full state if level is 0.
	if lvl == 0 {
		return saveFullSnapshot(s, lvl, st)
	}

	// Get anchor state to compute the diff from.
	anchorState, err := getAnchorState(s, offset, lvl, slot)
	if err != nil {
		return err
	}

	err = saveHdiff(s, lvl, anchorState, st)
	if err != nil {
		return err
	}

	return nil
}

// StateDiff retrieves the full state for a given slot.
func (s *Store) StateDiff(ctx context.Context, slot primitives.Slot) (state.BeaconState, error) {
	offset, err := getOffset(s)
	if err != nil {
		return nil, err
	}
	if uint64(slot) < offset {
		return nil, ErrSlotBeforeOffset
	}

	snapshot, diffChain := getDiffChain(s, offset, slot)
	return nil, nil
}

// SaveHdiff computes the diff between the anchor state and the current state and saves it to the database.
func saveHdiff(s *Store, lvl int, anchor, st state.ReadOnlyBeaconState) error {
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

	// Save the full state to the cache (if not the last level).
	if lvl != len(exponents)-1 {
		anchorCache[lvl] = st
	}

	return nil
}

// SaveFullSnapshot saves the full level 0 state snapshot to the database.
func saveFullSnapshot(s *Store, lvl int, st state.ReadOnlyBeaconState) error {
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
	// Save the full state to the cache.
	anchorCache[lvl] = st
	return nil
}
