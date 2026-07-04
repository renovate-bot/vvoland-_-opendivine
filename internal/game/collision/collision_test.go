// SPDX-License-Identifier: GPL-3.0-only

package collision

import "testing"

// TestCubeBlocks locks down the documented cube semantics
// (re_docs/formats/collide.md): the box spans [X, X+Width] with centre
// anchor + Width/2 (NOT centred on the anchor — the pre-engine
// implementation's square-AABB-around-the-anchor is the regression
// this guards), XExtent extends the reach rightward when it exceeds
// Width, and a disabled cube (open door) never blocks.
func TestCubeBlocks(t *testing.T) {
	tests := []struct {
		name      string
		cube      Cube
		px, py, r float64
		want      bool
	}{
		{
			name: "centre of the symmetric box blocks",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: true},
			px:   120, py: 100, r: 6, want: true,
		},
		{
			name: "left of the anchor is outside (box is not centred on the anchor)",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: true},
			px:   80, py: 100, r: 6, want: false,
		},
		{
			name: "just past the box right edge with margin",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: true},
			px:   170, py: 100, r: 6, want: false,
		},
		{
			name: "x_extent reach blocks beyond width",
			cube: Cube{X: 100, Y: 100, Width: 40, XExtent: 120, Enabled: true},
			px:   200, py: 100, r: 6, want: true,
		},
		{
			name: "vertical clearance is width/2 + r",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: true},
			px:   120, py: 130, r: 6, want: false,
		},
		{
			name: "vertically inside width/2 + r",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: true},
			px:   120, py: 124, r: 6, want: true,
		},
		{
			name: "disabled cube never blocks",
			cube: Cube{X: 100, Y: 100, Width: 40, Enabled: false},
			px:   120, py: 100, r: 6, want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cube.Blocks(tc.px, tc.py, tc.r); got != tc.want {
				t.Errorf("Cube%+v.Blocks(%v, %v, %v) = %v, want %v",
					tc.cube, tc.px, tc.py, tc.r, got, tc.want)
			}
		})
	}
}

// TestCubeDistance pins the reach metric interaction checks share with
// blocking: zero on the core segment (which is inset by Width/2 so the
// capsule's on-axis span is exactly [X, X+reach]), Euclidean off it.
func TestCubeDistance(t *testing.T) {
	c := Cube{X: 100, Y: 100, Width: 40, XExtent: 120, Enabled: true}
	// Core segment runs [120, 200]; capsule surface spans [100, 220] on-axis.
	if d := c.Distance(150, 100); d != 0 {
		t.Errorf("on-segment distance = %v, want 0", d)
	}
	if d := c.Distance(120, 130); d != 30 {
		t.Errorf("below-core distance = %v, want 30", d)
	}
	if d := c.Distance(240, 100); d != 40 {
		t.Errorf("past-reach distance = %v, want 40 (core ends at 200)", d)
	}
	// On-axis blocking edge: the capsule surface ends at X+reach = 220,
	// so with mover radius 6 the strict-< threshold sits at 226.
	if c.Blocks(226, 100, 6) {
		t.Error("Blocks at X+reach+r, want free (strict <)")
	}
	if !c.Blocks(225, 100, 6) {
		t.Error("no Blocks just inside X+reach+r, want blocked")
	}
}
