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
