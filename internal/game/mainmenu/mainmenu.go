// SPDX-License-Identifier: GPL-3.0-only

// Package mainmenu renders OpenDivine's main menu screen.
//
// Button labels are placeholder English; localised labels and the
// engine's 3D backdrop are not yet wired.
package mainmenu

import "github.com/hajimehoshi/ebiten/v2"

// Action is what a button click asks the host to do.
type Action int

const (
	ActionNone Action = iota
	ActionNewGame
	ActionLoadGame
	ActionSaveGame
	ActionOptions
	ActionViewIntro
	ActionCredits
	ActionResume
	ActionQuit
)

// Button describes one menu entry.
type Button struct {
	Label    string
	Action   Action
	Disabled bool
}

// Menu is the main-menu Screen.
//
// HasLiveGame controls whether Resume / Save are enabled. Backdrop,
// if non-nil, is blitted behind the buttons; otherwise Draw falls
// back to a programmatic gradient.
type Menu struct {
	Buttons     []Button
	HasLiveGame bool
	Backdrop    *ebiten.Image

	pending Action
	hovered int // index of hovered button, -1 if none
	w, h    int
}

// New builds the engine-faithful 8-button menu. hasLiveGame controls
// whether Resume / Save are clickable.
func New(hasLiveGame bool) *Menu {
	m := &Menu{
		HasLiveGame: hasLiveGame,
		hovered:     -1,
	}
	m.Buttons = []Button{
		{"Resume", ActionResume, !hasLiveGame},
		{"New Game", ActionNewGame, false},
		{"Load Game", ActionLoadGame, true}, // not yet implemented
		{"Save Game", ActionSaveGame, !hasLiveGame},
		{"Options", ActionOptions, true},
		{"View Intro", ActionViewIntro, true},
		{"Credits", ActionCredits, true},
		{"Quit", ActionQuit, false},
	}
	return m
}

// TakeAction returns the last clicked action and clears it.
func (m *Menu) TakeAction() Action {
	a := m.pending
	m.pending = ActionNone
	return a
}

// buttonRect returns the screen rect for button index i: a stack of
// equal rows whose size tracks itemPx, starting 45% down the window.
// The hit region is the centre half of the window width; Draw centres
// the text in it.
func (m *Menu) buttonRect(i int) (x, y, w, h int) {
	lh := m.lineH()
	startY := m.h * 35 / 100

	w = m.w / 2
	x = (m.w - w) / 2
	y = startY + i*lh
	return x, y, w, lh
}
