// SPDX-License-Identifier: GPL-3.0-only

package world

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestRealPartition(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	path := filepath.Join(gamedata, "main/startup/world.x0")
	data, err := os.ReadFile(path)
	assert.NilError(t, err)

	cells, objects := 0, 0
	withFloor := 0
	withOverlay := 0
	tileIDs := map[int16]int{}
	err = Walk(data, func(x, y int, c Cell) {
		cells++
		objects += int(c.ObjectCount)
		if c.FloorTileID != 0 {
			withFloor++
		}
		if c.OverlayTile != -1 {
			withOverlay++
		}
		tileIDs[c.FloorTileID]++
	})
	assert.NilError(t, err)

	t.Logf("cells=%d, objects=%d, with floor=%d, with overlay=%d, distinct tile ids=%d",
		cells, objects, withFloor, withOverlay, len(tileIDs))
	assert.Assert(t, cmp.Equal(cells > 0, true), "no cells walked")
}
