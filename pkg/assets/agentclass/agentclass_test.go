// SPDX-License-Identifier: GPL-3.0-only

package agentclass

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestReadHero(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	f, err := os.Open(filepath.Join(gamedata, "main/startup/data.000"))
	assert.NilError(t, err)
	defer f.Close()

	c, err := ReadHero(f)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(c.Name, "Hero"))
	want := [numAnimSlots]uint8{
		20, 20, 20, 20, 5, 20, 20, 0, 20, 20, 0, 20, 20, 20, 0, 0, 20, 20, 20,
	}
	assert.Check(t, cmp.DeepEqual(c.DirCounts, want))
}
