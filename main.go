// SPDX-License-Identifier: GPL-3.0-only

// Command-line entry point for OpenDivine. Parses flags and hands
// off to internal/game; the actual asset loading, sprite composition,
// depth-sort, input handling, and render pipeline live in
// package game.
//
// Controls:
//
//	WASD / arrow keys  pan / move
//	Left click         walk to clicked point
//	Mouse wheel        zoom in/out
//	Shift              move faster
//	F7 / F8            toggle floor / object render
//	F9                 toggle camera follow
//	F12                save screenshot
//	Esc                quit
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"grono.dev/opendivine/internal/game"
	"grono.dev/opendivine/internal/game/gamedata"
)

func main() {
	// Two-phase parse: a silent pre-pass extracts -gamedata so we
	// can resolve the install path and read the survivor's spawn
	// from location.000. The real flag declarations below then use
	// those spawn coordinates as their defaults, which means -help
	// shows actual values like "default 8024" instead of zeros.
	var gameDirArg string
	pre := flag.NewFlagSet("pre", flag.ContinueOnError)
	pre.SetOutput(io.Discard)
	pre.StringVar(&gameDirArg, "gamedata", "", "")
	_ = pre.Parse(os.Args[1:])

	resolved, err := gamedata.Resolve(gameDirArg)
	if err != nil {
		log.Fatalf("gamedata: %v", err)
	}
	spawn, err := game.ReadSpawn(resolved)
	if err != nil {
		log.Printf("spawn: %v (using world centre)", err)
	}

	cfg := game.Config{GamedataDir: resolved}

	flag.StringVar(&gameDirArg, "gamedata", gameDirArg,
		"Divine Divinity install root (auto-detected if blank: env, ./gamedata, then common Steam/GOG paths)")
	flag.IntVar(&cfg.Region, "region", spawn.Region, "world partition (0..4)")
	flag.Float64Var(&cfg.Zoom, "zoom", 1.5, "initial zoom (1.0 = native 1:1)")
	flag.Float64Var(&cfg.PosX, "posx", spawn.X, "initial player X in world pixels")
	flag.Float64Var(&cfg.PosY, "posy", spawn.Y, "initial player Y in world pixels")
	flag.StringVar(&cfg.Class, "class", "wizf", "hero class: surm/surf/warm/warf/wizm/wizf")
	flag.IntVar(&cfg.WindowW, "w", 1280, "window width")
	flag.IntVar(&cfg.WindowH, "h", 720, "window height")
	flag.StringVar(&cfg.Screenshot, "screenshot", "",
		"if set, render one frame, save PNG to this path, and exit")
	flag.IntVar(&cfg.Dir, "dir", -1, "force player Dir 0..7 (debug; -1 = no override)")
	flag.IntVar(&cfg.WalkFrame, "walk", -1, "force walk anim (slot 1) at AnimIdx N (debug; -1 = no override)")
	flag.IntVar(&cfg.Slot, "slot", -1,
		"force AnimSlot (debug; -1 = no override); 0=B, 1=A walk, 2=Q punch, 3=D, 4=E, 5=F, 6=H, 7=P, 11=G, 12=C, 13=Z, 16=J, 17=M/K, 18=U")
	flag.BoolVar(&cfg.SkipMenu, "skipmenu", false,
		"skip the main menu and boot straight into the world")
	flag.StringVar(&cfg.MenuMusic, "Title", "",
		"force a specific music.dat label for menu playback (e.g. \"1\", \"forest\"); empty = first menu track")
	flag.Parse()

	if err := game.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
