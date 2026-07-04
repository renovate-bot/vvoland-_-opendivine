// SPDX-License-Identifier: GPL-3.0-only

package mover_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"grono.dev/opendivine/internal/game/collision"
	"grono.dev/opendivine/internal/game/mover"
	"grono.dev/opendivine/internal/testutils"
	"grono.dev/opendivine/pkg/assets/collide"
	"grono.dev/opendivine/pkg/assets/location"
	"grono.dev/opendivine/pkg/assets/objects"
	"grono.dev/opendivine/pkg/assets/world"
)

// TestWalkShippedWorld drives the leg stepper across the real region-0
// start room and asserts the two invariants that matter for the bug this
// replaced: the mover makes progress, and it NEVER occupies a blocked
// cell (no tunnelling through the walls it now collides with — including
// the type-0 stone walls just south of the spawn).
func TestWalkShippedWorld(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	read := func(rel string) []byte {
		b, err := os.ReadFile(filepath.Join(gamedata, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return b
	}
	cat, err := objects.Decode(bytes.NewReader(read("static/objects.000")))
	if err != nil {
		t.Fatal(err)
	}
	col, err := collide.Decode(bytes.NewReader(read("static/imagelists/Collide.0")))
	if err != nil {
		t.Fatal(err)
	}
	locs, err := location.Decode(bytes.NewReader(read("global/location.000")))
	if err != nil {
		t.Fatal(err)
	}

	grid := collision.NewGrid()
	err = world.Walk(read("main/startup/world.x0"), func(cx, cy int, c world.Cell) {
		for _, o := range c.Objects {
			id := int(o.CatalogueID)
			if id >= len(col.Records) || id >= len(cat.Entries) {
				continue
			}
			cr := col.Records[id]
			e := cat.Entries[id]
			door := e.HasS(objects.SDoor)
			mask := collision.ObjectMask(collision.ObjectState{
				PlayerBlock:   e.HasS(objects.SPlayerBlock),
				WalkThrough:   e.HasS(objects.SWalkThrough),
				Door:          door,
				Closed:        door && e.HasS(objects.SClosed),
				Locked:        e.HasS(objects.SLocked),
				Light:         e.HasS(objects.SLight),
				Lever:         e.HasS(objects.SLever),
				WalkOn:        e.HasSB(objects.SBWalkOn),
				NoLookThrough: e.HasSB(objects.SBNoLookThrough),
			})
			if mask == 0 {
				continue
			}
			grid.Stamp(collision.Cube{
				X:       cx + int(o.SubX) + int(cr.AnchorX),
				Y:       cy + int(o.SubY) + int(cr.AnchorY),
				XExtent: int(cr.XExtent),
				Width:   int(cr.Width),
				Mask:    mask,
			})
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	var sx, sy int
	for _, r := range locs.Records {
		if r.Name == "stps_hero" {
			sx, sy = int(r.V0), int(r.V1)
		}
	}
	const mask = collision.MoverMask | collision.MaskDoorClosed

	// Head in eight compass directions in turn; each is a fresh mover from
	// the spawn so a wall in one direction cannot mask progress in another.
	dirs := []struct {
		name   string
		gx, gy float64
	}{
		{"E", float64(sx) + 4096, float64(sy)},
		{"W", float64(sx) - 4096, float64(sy)},
		{"S", float64(sx), float64(sy) + 4096},
		{"N", float64(sx), float64(sy) - 4096},
	}
	movedSomewhere := false
	for _, d := range dirs {
		m := mover.New(grid, mask, float64(sx), float64(sy), 2)
		start := m.X + m.Y
		for f := range 400 {
			m.Update(d.gx, d.gy, true)
			if grid.Blocked(int(m.X), int(m.Y), mask) {
				t.Fatalf("dir %s: mover tunnelled into a blocked cell at (%.0f,%.0f) frame %d",
					d.name, m.X, m.Y, f)
			}
		}
		if m.X+m.Y != start {
			movedSomewhere = true
		}
		t.Logf("dir %s: spawn (%d,%d) -> (%.0f,%.0f)", d.name, sx, sy, m.X, m.Y)
	}
	if !movedSomewhere {
		t.Error("mover made no progress in any direction from the spawn")
	}
}
