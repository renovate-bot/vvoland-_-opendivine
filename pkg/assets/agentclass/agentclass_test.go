// SPDX-License-Identifier: GPL-3.0-only

package agentclass

import (
	"os"
	"path/filepath"
	"testing"

	"grono.dev/opendivine/internal/testutils"
)

// TestReadHero parses the player class out of the shipped data.000
// and asserts the expected per-anim-slot direction counts.  These
// values come straight from the shipped Steam build of Divine
// Divinity; if Larian re-issues a patched data.000 with different
// counts the test should be updated.
//
// Reads from $TEST_GAMEDATA_PATH/main/startup/data.000.  Skips
// when the env var is unset.
func TestReadHero(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	f, err := os.Open(filepath.Join(gamedata, "main/startup/data.000"))
	if err != nil {
		t.Fatalf("open data.000: %v", err)
	}
	defer f.Close()
	c, err := ReadHero(f)
	if err != nil {
		t.Fatalf("ReadHero: %v", err)
	}
	if c.Name != "Hero" {
		t.Errorf("name = %q, want %q", c.Name, "Hero")
	}
	want := [numAnimSlots]uint8{
		20, 20, 20, 20, 5, 20, 20, 0, 20, 20, 0, 20, 20, 20, 0, 0, 20, 20, 20,
	}
	if c.DirCounts != want {
		t.Errorf("dirCounts = %v, want %v", c.DirCounts, want)
	}
}
