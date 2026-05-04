// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Config is the input to New / Run. Every field is a plain value;
// callers (cmd/divine and any future GUI launcher) are responsible
// for picking sensible defaults before passing a Config in. The
// debug-only Dir / WalkFrame / Slot fields use -1 to mean "no
// override".
type Config struct {
	// GamedataDir is the resolved path to a Divine Divinity install.
	GamedataDir string

	// Screenshot, when non-empty, makes the game render exactly one frame, save
	// it to this path as PNG, and exit.
	Screenshot string

	// Region is the world partition to load (0..4). Pre-populate from
	// ReadSpawn().Region for the engine-faithful default.
	Region int

	// Zoom is the initial render zoom; 1.0 = native 1:1 pixel.
	Zoom float64

	// PosX, PosY are the starting world position in pixels. Pre- populate from
	// ReadSpawn() for the survivor's basement.
	PosX, PosY float64

	// WindowW, WindowH set the initial window size.
	WindowW, WindowH int

	// Class is the hero class folder name under static\heroes\.
	Class string

	// Debug overrides. -1 = no override.
	Dir       int // force player Dir 0..7
	WalkFrame int // force walk anim (slot 1) at this AnimIdx
	Slot      int // force AnimSlot
}

func Run(cfg Config) error {
	g, err := New(cfg)
	if err != nil {
		return err
	}
	ebiten.SetWindowSize(g.winW, g.winH)
	ebiten.SetWindowTitle("OpenDivine")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}
