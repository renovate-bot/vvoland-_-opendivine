// SPDX-License-Identifier: GPL-3.0-only

package collide

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

var expected = []struct {
	name      string // on-disk filename under static/imagelists/
	count     int
	cpackedID int // -1 = not paired with a CPacked list
}{
	{"Collide.0", 7208, 0},
	{"Collide.1", 78853, 1},
	{"Collide.2", 0, -1},
	{"Collide.3", 1336, 3},
	{"Collide.4", 383, 4},
	{"Collide.5", 734, -1}, // 5 entries differ from CPacked.5's 262
	{"Collide.6", 55, -1},
	{"Collide.7", 4838, 7},
	{"Collide.8", 667, -1},
	{"Collide.9", 9, 9},
	{"Collide.10", 78266, -1},
	{"Collide.11", 131, 11},
	{"Collide.12", 1387, 12},
}

func TestAllCollideFiles(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	dir := filepath.Join(gamedata, "static/imagelists")
	entries, err := os.ReadDir(dir)
	assert.NilError(t, err)

	available := map[string]string{} // lower-case name -> on-disk name
	for _, e := range entries {
		available[strings.ToLower(e.Name())] = e.Name()
	}

	for _, w := range expected {
		t.Run(w.name, func(t *testing.T) {
			onDisk, ok := available[strings.ToLower(w.name)]
			if !ok {
				t.Skipf("not present in this install: %s", w.name)
			}
			path := filepath.Join(dir, onDisk)
			data, err := os.ReadFile(path)
			assert.NilError(t, err)

			f, err := Decode(bytes.NewReader(data))
			assert.NilError(t, err)
			assert.Check(t, cmp.Len(f.Records, w.count))
			assert.Check(t, cmp.Equal(len(f.Records)*RecordSize, len(data)))
		})
	}
}
