// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"grono.dev/opendivine/internal/game/character"
	"grono.dev/opendivine/internal/game/collision"
	"grono.dev/opendivine/pkg/assets/collide"
	"grono.dev/opendivine/pkg/assets/cpacked"
	"grono.dev/opendivine/pkg/assets/objects"
)

type Game struct {
	gameDir string
	region  int

	floorTiles  map[int16]*ebiten.Image // lazily decoded floor tiles
	floorReader *cpacked.Reader
	objSprites  map[int]*sprite // lazily decoded object sprites
	objReader   *cpacked.Reader
	catalog     *objects.Catalog // objects.000 (per-id SBFlags)
	collide0    *collide.File    // per-cat-id cube data for imagelist 0

	cells     []floorCell  // all populated cells, sorted by (CellY, CellX)
	insts     []objectInst // all placed objects, sorted by (Layer, Y)
	colliders []collider   // per-object stamps into walkGrid + reach boxes
	// walkGrid: the engine's walkability cell grid; objects rasterize
	// their cubes into it and movers test cells by mask
	// (re_docs/formats/collide.md).
	walkGrid *collision.Grid

	camX, camY float64 // world pixel at screen center
	zoom       float64 // output_pixel / world_pixel
	winW, winH int

	player       *character.Character
	destX, destY float64 // click-to-walk target (world px)
	hasDest      bool
	cameraFollow bool // true: camera locks to player; false: free pan

	showFloors  bool   // F7 toggle
	showObjects bool   // F8 toggle
	wantShot    bool   // F12: capture next frame to screenshot
	shotPath    string // -screenshot flag: write to this path then exit
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.winW, g.winH = outsideWidth, outsideHeight
	return outsideWidth, outsideHeight
}
