// SPDX-License-Identifier: GPL-3.0-only

// Package collision implements the engine's movement collision
// (re_docs/formats/collide.md): there is no per-move geometric test —
// each placed object's collide cube is RASTERIZED into a cell-flag
// grid at placement (an inclusive rectangle in iso cell space), and
// movers test whole cells by flag mask. Doors change state by
// re-rasterizing with a new mask.
package collision

// Cell-flag bits, as stamped by the engine's rasterizer and tested by
// its walkability function (fcn.0056f3c0). Movers test with
// MaskStatic|MaskPlayerBlock|MaskAgent (0x13).
const (
	MaskStatic      uint16 = 0x0001 // hard blocker (default objects; locked doors; map records)
	MaskPlayerBlock uint16 = 0x0002 // sb_player_block objects
	MaskDoorClosed  uint16 = 0x0004 // closed door (sight/line tests)
	MaskSight       uint16 = 0x0008 // sb_no_look_through
	MaskAgent       uint16 = 0x0010 // an agent's leg-destination occupancy
	MaskLight       uint16 = 0x0080 // sb_light
	MaskLever       uint16 = 0x0100 // sb_lever
	MaskWalkOn      uint16 = 0x0400 // sb_walk_on climb cells (height-gated)
)

// MoverMask is what the engine's movement steppers test candidate
// cells with (mask 0x13 at the fcn.0056f3c0 call sites).
const MoverMask = MaskStatic | MaskPlayerBlock | MaskAgent

// gridMax is the engine's valid-index bound (index <= 0x200800 gates
// every grid access).
const gridMax = 0x200800

// Grid is the walkability grid: the engine's worldmap cell array
// ([0x74eca0]) reduced to the fields movement needs. Cells are
// addressed by the engine's own index formula v<<10 + u with
// u = (x+y)>>5, v = y>>5 — including its aliasing for u > 1023, so
// stamps and tests always agree with the original.
type Grid struct {
	flags []uint16
	// blockRef is the engine's per-cell blocker refcount (the top
	// nibble of the flags word in the original; kept separate here):
	// stamps with MaskStatic|MaskPlayerBlock increment it, removals
	// decrement, and flag bits clear only when it reaches zero.
	blockRef []uint8
}

// NewGrid returns an empty walkability grid.
func NewGrid() *Grid {
	return &Grid{
		flags:    make([]uint16, gridMax+1),
		blockRef: make([]uint8, gridMax+1),
	}
}

// cellIndex maps world coordinates to the engine's cell index:
// u = (x+y)>>5, v = y>>5, index = v<<10 + u (fcn.0056d720 and every
// other site). ok is false outside the engine's index gate.
func cellIndex(x, y int) (int, bool) {
	u := (x + y) >> 5
	v := y >> 5
	idx := v<<10 + u
	return idx, idx >= 0 && idx <= gridMax
}

// CellIndex maps world (x, y) to the engine's cell index (v<<10 + u,
// u = (x+y)>>5, v = y>>5); ok is false outside the engine's index gate.
// Exposed so callers (e.g. a debug overlay) can tell which single cell
// a mover occupies — the engine's movers carry no radius and own
// exactly one cell.
func CellIndex(x, y int) (idx int, ok bool) {
	return cellIndex(x, y)
}

// CellUV returns the (u, v) cell coordinates for world (x, y):
// u = (x+y)>>5, v = y>>5. Movers work in (u, v) space to apply the
// engine's per-direction corner-cell offsets.
func CellUV(x, y int) (u, v int) {
	return (x + y) >> 5, y >> 5
}

// BlockedUV reports whether cell (u, v) has any of the mask bits set.
// Out-of-range cells (past the engine's index gate) count as blocked.
func (g *Grid) BlockedUV(u, v int, mask uint16) bool {
	idx := v<<10 + u
	if idx < 0 || idx > gridMax {
		return true
	}
	return g.flags[idx]&mask != 0
}

// Blocked reports whether the cell containing world (x, y) has any of
// the mask bits set — the walkability core (destination-cell test of
// fcn.0056f3c0; the per-direction corner chains and the climb gate
// belong to the leg stepper and are not modelled yet).
func (g *Grid) Blocked(x, y int, mask uint16) bool {
	idx, ok := cellIndex(x, y)
	if !ok {
		return true // out of the engine's grid = not walkable
	}
	return g.flags[idx]&mask != 0
}

// Cube is one object's stamp parameters, kept so the object can
// re-rasterize on state change (the engine's remove+add on door
// toggle).
type Cube struct {
	X, Y    int // object world position + collide record anchor_x/anchor_y
	XExtent int // collide i16[3]: spans the iso u axis
	Width   int // collide i16[5]: width/2 spans the v axis
	Mask    uint16
}

// cells calls fn for every cell of the cube's inclusive rectangle,
// exactly per the engine rasterizer fcn.0056d720:
//
//	u ∈ [(x+y)>>5, (x+y+x_extent)>>5],  v ∈ [(y−width/2)>>5, y>>5]
func (c *Cube) cells(fn func(idx int)) {
	u1 := (c.X + c.Y) >> 5
	u2 := (c.X + c.Y + c.XExtent) >> 5
	v1 := (c.Y - c.Width/2) >> 5
	v2 := c.Y >> 5
	for v := v1; v <= v2; v++ {
		for u := u1; u <= u2; u++ {
			idx := v<<10 + u
			if idx >= 0 && idx <= gridMax {
				fn(idx)
			}
		}
	}
}

// ContainsCell reports whether world (x, y) falls in one of the cells
// of the cube's stamp rectangle.
func (c *Cube) ContainsCell(x, y int) bool {
	pu := (x + y) >> 5
	pv := y >> 5
	u1 := (c.X + c.Y) >> 5
	u2 := (c.X + c.Y + c.XExtent) >> 5
	v1 := (c.Y - c.Width/2) >> 5
	v2 := c.Y >> 5
	return pu >= u1 && pu <= u2 && pv >= v1 && pv <= v2
}

// Stamp rasterizes the cube into the grid (fcn.0056d720): OR the mask
// into each cell, counting hard blockers. A zero mask stamps nothing.
func (g *Grid) Stamp(c Cube) {
	if c.Mask == 0 {
		return
	}
	c.cells(func(idx int) {
		g.flags[idx] |= c.Mask
		if c.Mask&(MaskStatic|MaskPlayerBlock) != 0 && g.blockRef[idx] < 255 {
			g.blockRef[idx]++
		}
	})
}

// Unstamp is the inverse (fcn.0056d890): decrement the blocker
// refcount and clear the cube's bits only when no other blocker still
// owns the cell.
func (g *Grid) Unstamp(c Cube) {
	if c.Mask == 0 {
		return
	}
	c.cells(func(idx int) {
		if c.Mask&(MaskStatic|MaskPlayerBlock) != 0 && g.blockRef[idx] > 0 {
			g.blockRef[idx]--
		}
		if g.blockRef[idx] == 0 {
			g.flags[idx] &^= c.Mask
		}
	})
}

// ObjectState is the flags-word state the mask derivation reads
// (instance s_* bits + two catalogue sb_* behaviour bits).
type ObjectState struct {
	PlayerBlock   bool // s_player_block
	WalkThrough   bool // s_walk_through
	Door          bool // s_door
	Closed        bool // s_closed (runtime state)
	Locked        bool // s_locked (runtime state)
	Light         bool // s_light
	Lever         bool // s_lever
	WalkOn        bool // catalogue sb_walk_on
	NoLookThrough bool // catalogue sb_no_look_through
}

// ObjectMask derives the cell mask from the object flags word, in the
// engine's exact order (fcn.00572100 / fcn.0056e2c0; note the cube
// `type` field plays no part):
//
//	if sb_player_block: mask = walk_through ? 0 : 0x2
//	elif sb_door:       mask = (closed ? 0x4 : 0) | (locked ? 0x1 : 0)
//	else:               mask = walk_through ? 0 : 0x1
//	plus light|lever|walk_on|no_look_through extras;
//	hard blockers strip the climb bit.
func ObjectMask(o ObjectState) uint16 {
	var mask uint16
	switch {
	case o.PlayerBlock:
		if !o.WalkThrough {
			mask = MaskPlayerBlock
		}
	case o.Door:
		if o.Closed {
			mask |= MaskDoorClosed
		}
		if o.Locked {
			mask |= MaskStatic
		}
	default:
		if !o.WalkThrough {
			mask = MaskStatic
		}
	}
	if o.Light {
		mask |= MaskLight
	}
	if o.Lever {
		mask |= MaskLever
	}
	if o.WalkOn {
		mask |= MaskWalkOn
	}
	if o.NoLookThrough {
		mask |= MaskSight
	}
	if mask&MaskStatic != 0 {
		mask &^= MaskWalkOn
	}
	return mask
}
