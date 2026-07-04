// SPDX-License-Identifier: GPL-3.0-only

package collision

import "testing"

// TestObjectMask locks down the engine's mask derivation order
// (re_docs/formats/collide.md): player_block wins over door, doors
// contribute closed/locked bits (an open unlocked door stamps
// nothing), walk_through kills default blockers, and hard blockers
// strip the climb bit.
func TestObjectMask(t *testing.T) {
	tests := []struct {
		name string
		o    ObjectState
		want uint16
	}{
		{"default object blocks", ObjectState{}, MaskStatic},
		{"walk-through decal stamps nothing", ObjectState{WalkThrough: true}, 0},
		{"closed unlocked door", ObjectState{Door: true, Closed: true}, MaskDoorClosed},
		{"closed locked door", ObjectState{Door: true, Closed: true, Locked: true}, MaskDoorClosed | MaskStatic},
		{"open unlocked door stamps nothing", ObjectState{Door: true}, 0},
		{"player-block", ObjectState{PlayerBlock: true}, MaskPlayerBlock},
		{"player-block walk-through", ObjectState{PlayerBlock: true, WalkThrough: true}, 0},
		{"player-block wins over door", ObjectState{PlayerBlock: true, Door: true, Closed: true}, MaskPlayerBlock},
		{"walk-on stair", ObjectState{WalkThrough: true, WalkOn: true}, MaskWalkOn},
		{"hard blocker strips climb", ObjectState{WalkOn: true}, MaskStatic},
		{"light and lever extras", ObjectState{Light: true, Lever: true}, MaskStatic | MaskLight | MaskLever},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ObjectMask(tc.o); got != tc.want {
				t.Errorf("ObjectMask(%+v) = %#x, want %#x", tc.o, got, tc.want)
			}
		})
	}
}

// TestStampRectangle pins the rasterizer's inclusive cell rectangle
// (fcn.0056d720): u spans (x+y)>>5 .. (x+y+x_extent)>>5 and v spans
// (y−width/2)>>5 .. y>>5.
func TestStampRectangle(t *testing.T) {
	g := NewGrid()
	// x+y = 1024 → u ∈ [32, 35] (x_extent 96); y = 512 → v ∈ [14, 16]
	// (width 128 → y−64).
	c := Cube{X: 512, Y: 512, XExtent: 96, Width: 128, Mask: MaskStatic}
	g.Stamp(c)

	for _, p := range []struct{ x, y int }{
		{512, 512},           // anchor cell (u32, v16)
		{512 + 96, 512},      // right edge (u35, v16)
		{512 + 64, 512 - 64}, // top edge (v14), u still in range
	} {
		if !g.Blocked(p.x, p.y, MoverMask) {
			t.Errorf("cell at (%d,%d) not blocked, want blocked", p.x, p.y)
		}
	}
	// Just outside on each axis.
	if g.Blocked(512+96+32, 512, MoverMask) { // u36
		t.Error("cell right of the rect blocked, want free")
	}
	if g.Blocked(512, 512-96, MoverMask) { // v13 (u drops to 29 — also outside)
		t.Error("cell above the rect blocked, want free")
	}
	if g.Blocked(512-33, 512, MoverMask) { // u31
		t.Error("cell left of the rect blocked, want free")
	}
}

// TestRefcountOverlap pins the blocker refcount semantics
// (fcn.0056d890): where two blockers overlap, removing one leaves the
// cell blocked; removing both frees it.
func TestRefcountOverlap(t *testing.T) {
	g := NewGrid()
	a := Cube{X: 512, Y: 512, Width: 64, Mask: MaskStatic}
	b := Cube{X: 512, Y: 512, Width: 64, Mask: MaskStatic}
	g.Stamp(a)
	g.Stamp(b)
	g.Unstamp(a)
	if !g.Blocked(512, 512, MoverMask) {
		t.Error("overlapped cell freed by removing one of two blockers")
	}
	g.Unstamp(b)
	if g.Blocked(512, 512, MoverMask) {
		t.Error("cell still blocked after removing both blockers")
	}
}

// TestDoorToggle pins the remove+add door transition: a closed locked
// door blocks movers; re-stamping it open and unlocked frees the cell
// while an unrelated wall in the same cell keeps blocking.
func TestDoorToggle(t *testing.T) {
	g := NewGrid()
	door := Cube{X: 512, Y: 512, Width: 64,
		Mask: ObjectMask(ObjectState{Door: true, Closed: true, Locked: true})}
	g.Stamp(door)
	if !g.Blocked(512, 512, MoverMask) {
		t.Fatal("locked door does not block")
	}
	g.Unstamp(door)
	door.Mask = ObjectMask(ObjectState{Door: true}) // open, unlocked
	g.Stamp(door)
	if g.Blocked(512, 512, MoverMask) {
		t.Error("open door still blocks movers")
	}

	wall := Cube{X: 512, Y: 512, Width: 64, Mask: MaskStatic}
	g.Stamp(wall)
	if !g.Blocked(512, 512, MoverMask) {
		t.Error("wall sharing the door's cell does not block")
	}
}

// TestOutOfGrid: coordinates outside the engine's index gate are not
// walkable.
func TestOutOfGrid(t *testing.T) {
	g := NewGrid()
	if !g.Blocked(-100, -100, MoverMask) {
		t.Error("negative-index cell walkable, want blocked")
	}
}
