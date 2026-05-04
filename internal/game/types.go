// SPDX-License-Identifier: GPL-3.0-only

package game

import "github.com/hajimehoshi/ebiten/v2"

const (
	defaultWindowW = 1280
	defaultWindowH = 720
	cellPx         = 64                   // engine cell size in native pixels
	worldCellsX    = 512                  // cells horizontally per region
	worldCellsY    = 1024                 // cells vertically per region
	worldXPx       = worldCellsX * cellPx // 32768 native world width
	worldYPx       = worldCellsY * cellPx // 65536 native world height
)

// sprite is a decoded object image.
type sprite struct {
	img *ebiten.Image
}

// objectInst is one placed object in the world.
type objectInst struct {
	X, Y, ObjID int
	Layer       int
	// Elev is the precomputed Y offset.
	// For most objects Elev = Layer (the engine's cumulative pixel elevation
	// per FUN_005830c0).
	// For walls (SBLightBlocker), Elev = 0, walls render at top-left regardless
	// of layer.
	Elev int
}

// floorCell is a populated world cell with non-default floor data.
type floorCell struct {
	CellX, CellY uint16
	FloorID      int16
	OverlayID    int16
}

// aabb is an axis-aligned blocker box in world pixels.
// Used for player-wall collision queries.
// X/Y is the top-left.
type aabb struct {
	X, Y, W, H int
}

func (a aabb) intersects(b aabb) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
