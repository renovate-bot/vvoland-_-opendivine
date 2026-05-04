// SPDX-License-Identifier: GPL-3.0-only

package game

// Per-frame depth-sort port of CSpriteSorter (div.exe:0x00547000):
//   - sortRecord builds the 7-int record emitted per
//     sprite, plus an attached AABB and dependency list for the
//     topological pass.
//   - compareSortRecords ports FUN_00546e40, the pairwise compare.
//   - aabbOverlap ports FUN_00471690, the overlap gate.

type sortRecord struct {
	r      [7]int
	bboxX1 int // world AABB minX
	bboxY1 int // world AABB minY (top-left in our render model)
	bboxX2 int // world AABB maxX
	bboxY2 int // world AABB maxY

	// inst >= 0: index into g.insts (a world objectInst).
	// inst <  0: character index = -(inst+1) into the local chars[].
	inst    int
	visited bool
	deps    []int // indices into the visible list (not g.insts)
}

// compareSortRecords reproduces FUN_00546e40.
// Returns +1 if a is "in front" of b, -1 if b is in front, 0 never.
func compareSortRecords(a, b *sortRecord) int {
	if b.r[3] <= a.r[2] {
		return 1
	}
	if a.r[3] <= b.r[2] {
		return -1
	}
	if a.r[5]+a.r[4] < b.r[4] {
		// a's elevation top sits below b's elevation floor.
		if b.r[1] <= a.r[0] {
			return 1
		}
		return -1
	}
	if b.r[5]+b.r[4] < a.r[4] {
		if a.r[1] < b.r[1] {
			return 1
		}
		return -1
	}
	// X tiebreak.
	if b.r[0] <= a.r[0] {
		return 1
	}
	return -1
}

// aabbOverlap is FUN_00471690 inclusive AABB overlap test.
// All four args are min..max in their axis already (we pre-normalise so we
// don't need the engine's swap).
func aabbOverlap(ax1, ay1, ax2, ay2, bx1, by1, bx2, by2 int) bool {
	if ax2 < bx1 || bx2 < ax1 {
		return false
	}
	if ay2 < by1 || by2 < ay1 {
		return false
	}
	return true
}
