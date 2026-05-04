// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"grono.dev/opendivine/internal/game/character"
	"grono.dev/opendivine/pkg/assets/collide"
	"grono.dev/opendivine/pkg/assets/objects"
)

func New(cfg Config) (*Game, error) {
	g := &Game{
		gameDir:      cfg.GamedataDir,
		region:       cfg.Region,
		floorTiles:   map[int16]*ebiten.Image{},
		objSprites:   map[int]*sprite{},
		zoom:         cfg.Zoom,
		camX:         cfg.PosX,
		camY:         cfg.PosY,
		cameraFollow: true,
		winW:         cfg.WindowW,
		winH:         cfg.WindowH,
		showFloors:   true,
		showObjects:  true,
		shotPath:     cfg.Screenshot,
	}

	fr, err := openImagelist(cfg.GamedataDir, floorImagelistID)
	if err != nil {
		return nil, fmt.Errorf("imagelist %d: %w", floorImagelistID, err)
	}
	g.floorReader = fr
	or, err := openImagelist(cfg.GamedataDir, objectImagelistID)
	if err != nil {
		return nil, fmt.Errorf("imagelist %d: %w", objectImagelistID, err)
	}
	g.objReader = or

	cf, err := os.Open(cfg.GamedataDir + "/static/objects.000")
	if err != nil {
		return nil, fmt.Errorf("open objects.000: %w", err)
	}
	g.catalog, err = objects.Decode(cf)
	cf.Close()
	if err != nil {
		return nil, fmt.Errorf("decode objects.000: %w", err)
	}

	// Collide.0, per-sprite cube records aligned with imagelist 0.
	// Used for player-wall collision; non-blocking on failure (player
	// just walks through walls if the file is missing).
	colf, err := os.Open(cfg.GamedataDir + "/static/imagelists/Collide.0")
	if err == nil {
		g.collide0, err = collide.Decode(colf)
		colf.Close()
		if err != nil {
			log.Printf("collide.0 decode: %v", err)
			g.collide0 = nil
		}
	} else {
		log.Printf("collide.0 open: %v", err)
	}

	// All 5 variants, the composer picks which .key group to use
	// from each based on the character's anim slot and equipment.
	player, err := character.Load(cfg.GamedataDir, cfg.Class, "A", "B", "C", "D", "E")
	if err != nil {
		log.Printf("player load: %v", err)
		player = &character.Character{Dir: 2}
	}
	player.X = cfg.PosX
	player.Y = cfg.PosY
	// AnimSlot 0 = engine cVar4='B' = stand idle (breathing).
	// Slot 11 ('G') is a move-style anim and slot 2 ('Q') is punch.
	player.AnimSlot = 0

	// Debug overrides. -1 means "not set" by convention; these flags
	// are debug-only so the negative-sentinel ambiguity is fine.
	if cfg.Dir >= 0 && cfg.Dir < 8 {
		player.Dir = cfg.Dir
		player.PinAnim = true
	}
	if cfg.WalkFrame >= 0 {
		player.AnimSlot = 1
		player.AnimIdx = cfg.WalkFrame
		player.PinAnim = true
	}
	if cfg.Slot >= 0 {
		player.ForceSlot = cfg.Slot
	}

	g.player = player

	if err := g.loadRegion(cfg.Region); err != nil {
		return nil, err
	}
	return g, nil
}
