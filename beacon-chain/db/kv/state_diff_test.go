package kv

import (
	"testing"

	"github.com/OffchainLabs/prysm/v6/math"
	"github.com/OffchainLabs/prysm/v6/testing/require"
)

func TestStateDiff_LoadOrInitOffset(t *testing.T) {
	db := setupDB(t)

	offset, rel, err := db.loadOrInitOffset(10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), offset)
	require.Equal(t, uint64(0), rel)

	offset, rel, err = db.loadOrInitOffset(20)
	require.NoError(t, err)
	require.Equal(t, uint64(10), offset)
	require.Equal(t, uint64(10), rel)

	offset, rel, err = db.loadOrInitOffset(5)
	require.ErrorIs(t, ErrSlotBeforeOffset, err)

	offset, rel, err = db.loadOrInitOffset(10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), offset)
	require.Equal(t, uint64(0), rel)
}

func TestStateDiff_ComputeLevel(t *testing.T) {
	db := setupDB(t)

	offset, rel, err := db.loadOrInitOffset(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), offset)
	require.Equal(t, uint64(0), rel)

	// 2 ** 21
	lvl, shouldSave := computeLevel(math.PowerOf2(21))
	require.Equal(t, 0, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 21 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(21) * 3)
	require.Equal(t, 0, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 18
	lvl, shouldSave = computeLevel(math.PowerOf2(18))
	require.Equal(t, 1, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 18 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(18) * 3)
	require.Equal(t, 1, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 16
	lvl, shouldSave = computeLevel(math.PowerOf2(16))
	require.Equal(t, 2, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 16 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(16) * 3)
	require.Equal(t, 2, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 13
	lvl, shouldSave = computeLevel(math.PowerOf2(13))
	require.Equal(t, 3, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 13 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(13) * 3)
	require.Equal(t, 3, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 11
	lvl, shouldSave = computeLevel(math.PowerOf2(11))
	require.Equal(t, 4, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 11 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(11) * 3)
	require.Equal(t, 4, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 9
	lvl, shouldSave = computeLevel(math.PowerOf2(9))
	require.Equal(t, 5, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 9 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(9) * 3)
	require.Equal(t, 5, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 5
	lvl, shouldSave = computeLevel(math.PowerOf2(5))
	require.Equal(t, 6, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 5 * 3
	lvl, shouldSave = computeLevel(math.PowerOf2(5) * 3)
	require.Equal(t, 6, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 7
	lvl, shouldSave = computeLevel(math.PowerOf2(7))
	require.Equal(t, 6, lvl)
	require.Equal(t, true, shouldSave)

	// 2 ** 5 + 1
	lvl, shouldSave = computeLevel(math.PowerOf2(5) + 1)
	require.Equal(t, false, shouldSave)

	// 2 ** 5 + 16
	lvl, shouldSave = computeLevel(math.PowerOf2(5) + 16)
	require.Equal(t, false, shouldSave)

	// 2 ** 5 + 32
	lvl, shouldSave = computeLevel(math.PowerOf2(5) + 32)
	require.Equal(t, true, shouldSave)
	require.Equal(t, 6, lvl)

}
