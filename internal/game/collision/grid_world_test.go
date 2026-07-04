// SPDX-License-Identifier: GPL-3.0-only

package collision_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"grono.dev/opendivine/internal/game/collision"
	"grono.dev/opendivine/internal/testutils"
	"grono.dev/opendivine/pkg/assets/collide"
	"grono.dev/opendivine/pkg/assets/location"
	"grono.dev/opendivine/pkg/assets/objects"
	"grono.dev/opendivine/pkg/assets/world"
)

// TestShippedWorldWalkable builds the walkability grid from the real
// world.x0 the same way the game does and sanity-checks it: the hero
// spawn cell must be free and most of its neighbourhood walkable —
// guarding against mask or anchor regressions that would wall the
// player in (cell blocking is coarse, so a systematic offset error
// shows up here immediately).
func TestShippedWorldWalkable(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	mustRead := func(rel string) []byte {
		b, err := os.ReadFile(filepath.Join(gamedata, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return b
	}
	cat, err := objects.Decode(bytes.NewReader(mustRead("static/objects.000")))
	if err != nil {
		t.Fatal(err)
	}
	col, err := collide.Decode(bytes.NewReader(mustRead("static/imagelists/Collide.0")))
	if err != nil {
		t.Fatal(err)
	}
	locs, err := location.Decode(bytes.NewReader(mustRead("global/location.000")))
	if err != nil {
		t.Fatal(err)
	}

	grid := collision.NewGrid()
	stamped := 0
	err = world.Walk(mustRead("main/startup/world.x0"), func(cx, cy int, c world.Cell) {
		for _, o := range c.Objects {
			id := int(o.CatalogueID)
			if id >= len(col.Records) || id >= len(cat.Entries) {
				continue
			}
			cr := col.Records[id]
			if cr.Type == 0 || cr.ZHeight <= 0 || cr.Width <= 0 {
				continue
			}
			e := cat.Entries[id]
			door := e.HasS(objects.SDoor)
			grid.Stamp(collision.Cube{
				X:       cx + int(o.SubX) + int(cr.AnchorX),
				Y:       cy + int(o.SubY) + int(cr.AnchorY),
				XExtent: int(cr.XExtent),
				Width:   int(cr.Width),
				Mask: collision.ObjectMask(collision.ObjectState{
					PlayerBlock:   e.HasS(objects.SPlayerBlock),
					WalkThrough:   e.HasS(objects.SWalkThrough),
					Door:          door,
					Closed:        door && e.HasS(objects.SClosed),
					Locked:        e.HasS(objects.SLocked),
					Light:         e.HasS(objects.SLight),
					Lever:         e.HasS(objects.SLever),
					WalkOn:        e.HasSB(objects.SBWalkOn),
					NoLookThrough: e.HasSB(objects.SBNoLookThrough),
				}),
			})
			stamped++
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stamped %d cubes", stamped)

	var sx, sy int
	for _, r := range locs.Records {
		if r.Name == "stps_hero" {
			sx, sy = int(r.V0), int(r.V1)
		}
	}
	if sx == 0 && sy == 0 {
		t.Fatal("no stps_hero spawn record")
	}
	mask := collision.MoverMask | collision.MaskDoorClosed
	if grid.Blocked(sx, sy, mask) {
		t.Errorf("hero spawn cell (%d,%d) is blocked", sx, sy)
	}
	free := 0
	for dy := -160; dy <= 160; dy += 32 {
		for dx := -160; dx <= 160; dx += 32 {
			if !grid.Blocked(sx+dx, sy+dy, mask) {
				free++
			}
		}
	}
	t.Logf("free cells around spawn: %d/121", free)
	if free < 60 {
		t.Errorf("only %d/121 cells around the spawn are walkable — the grid looks over-blocked", free)
	}
}
