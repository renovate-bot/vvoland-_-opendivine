// SPDX-License-Identifier: GPL-3.0-only

// Package collision implements the engine's cube-based movement
// collision (re_docs/formats/collide.md): each object whose collide
// record has Type != 0 contributes a cube anchored at the object's
// world position (the engine overwrites the record's file anchors with
// world X/Y at load), and movement runs a sqrt-distance test of the
// mover against the cube's width / x_extent.
package collision

import "math"

// Cube is one object's collision shape in the ground plane.
//
// Per the collide record semantics: the anchor (X, Y) is the object's
// world position; Width is the symmetric collision-box width read as
// centre = anchor + width/2 (FUN_004eca40/FUN_00448020), so the box
// spans [X, X+Width] horizontally and, per the depth-sort mirrors,
// ±Width/2 vertically around Y; XExtent is the asymmetric rightward
// reach (the right-edge offset from the anchor, which can exceed
// Width for off-centre sprites).
//
// The exact engine block primitive is diffuse (collide.md 🟡), but the
// documented narrow phase is a Euclidean distance test against
// width/x_extent. We model that as a capsule whose on-axis span is
// exactly [X, X+max(Width, XExtent)]: the core segment runs from
// X+Width/2 to X+reach−Width/2 with radius Width/2, so a centred
// sprite degenerates to the engine's documented circle at
// anchor+width/2 and an off-centre one extends to its x_extent right
// edge.
type Cube struct {
	X, Y    float64 // anchor = object world position
	Width   float64 // symmetric box width (radius = Width/2)
	XExtent float64 // asymmetric rightward reach; 0 or < Width for most sprites
	Enabled bool    // false while a door stands open
}

// reach is the cube's full on-axis extent from the anchor.
func (c *Cube) reach() float64 {
	return math.Max(c.Width, c.XExtent)
}

// segment returns the capsule core's start and end X (end >= start).
func (c *Cube) segment() (x0, x1 float64) {
	r := c.Width / 2
	x0 = c.X + r
	x1 = math.Max(c.X+c.reach()-r, x0)
	return x0, x1
}

// Blocks reports whether a mover at (px, py) with radius r intersects
// the cube — the sqrt-distance narrow phase.
func (c *Cube) Blocks(px, py, r float64) bool {
	if !c.Enabled {
		return false
	}
	return c.Distance(px, py) < c.Width/2+r
}

// Distance returns the Euclidean distance from a point to the cube's
// core segment (0 when on the segment). Interaction reach checks use
// it too, so reach and blocking agree on the shape.
func (c *Cube) Distance(px, py float64) float64 {
	x0, x1 := c.segment()
	cx := math.Min(math.Max(px, x0), x1)
	return math.Hypot(px-cx, py-c.Y)
}
