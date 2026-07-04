// SPDX-License-Identifier: GPL-3.0-only

package objects

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestRealCatalog(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	path := filepath.Join(gamedata, "static/objects.000")
	data, err := os.ReadFile(path)
	assert.NilError(t, err)

	cat, err := Decode(bytes.NewReader(data))
	assert.NilError(t, err)
	assert.Check(t, cmp.Len(cat.Entries, 7208))

	// Spot check known entries (verified against CSV exporter output).
	for _, want := range []struct {
		idx  int
		name string
	}{
		{0, "Dead bush"},
		{100, "Rock wall"},
		{156, "Tree"},
		{274, "Metal Shield"},
		{1000, "Rocks"},
	} {
		got := cat.Entries[want.idx]
		assert.Check(t, cmp.Equal(got.ID, uint32(want.idx)), "entry %d ID mismatch", want.idx)
		assert.Check(t, cmp.Equal(got.Name, want.name), "entry %d name mismatch", want.idx)
	}

	// Count entries with sb_force_floor set — these are the floor objects.
	floorCount := 0
	for _, o := range cat.Entries {
		if o.HasSB(SBForceFloor) {
			floorCount++
		}
	}
	t.Logf("entries with sb_force_floor: %d", floorCount)
}

// TestInstanceStateFlags locks down the S* bit positions of FlagsA against
// known shipped entries. The bit map is the instance flags word from
// re_docs/object-interaction.md; this guards the game's interaction
// classifier, which derives door/chest/lever kinds and the initial
// closed/locked state from these bits (and against regressing to the old
// collide-Type==2 heuristic that had no doc basis).
func TestInstanceStateFlags(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	data, err := os.ReadFile(filepath.Join(gamedata, "static/objects.000"))
	assert.NilError(t, err)
	cat, err := Decode(bytes.NewReader(data))
	assert.NilError(t, err)

	for _, want := range []struct {
		idx                        int
		name                       string
		door, chest, lever, closed bool
	}{
		{64, "Door", true, false, false, true},
		{65, "Door", true, false, false, true},
		{176, "Chest", false, true, false, true},
		{697, "Lever", false, false, true, false},
		// A door-named decor entry without kind bits: must NOT classify
		// as interactive.
		{66, "Door", false, false, false, false},
	} {
		got := cat.Entries[want.idx]
		assert.Check(t, cmp.Equal(got.Name, want.name), "entry %d name", want.idx)
		assert.Check(t, cmp.Equal(got.HasS(SDoor), want.door), "entry %d sb_door", want.idx)
		assert.Check(t, cmp.Equal(got.HasS(SChest), want.chest), "entry %d sb_chest", want.idx)
		assert.Check(t, cmp.Equal(got.HasS(SLever), want.lever), "entry %d sb_lever", want.idx)
		assert.Check(t, cmp.Equal(got.HasS(SClosed), want.closed), "entry %d sb_closed", want.idx)
	}
}
