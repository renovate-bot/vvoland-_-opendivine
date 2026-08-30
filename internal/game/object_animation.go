// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"grono.dev/opendivine/pkg/assets/apacked"
)

// objectAnimationTicksPerFrame is the interim 100 ms frame duration from the
// animation timing plan, expressed in the engine's 40 Hz simulation ticks.
const objectAnimationTicksPerFrame = 4

// advanceObjectAnimations advances timing for each ambient world-object
// animation. Each selected frame is held for the configured number of ticks.
// Interactive objects retain their animation index for state-driven playback,
// but must not loop while idle.
func (g *Game) advanceObjectAnimations() {
	if g.objectAnimations == nil {
		return
	}
	for i := range g.insts {
		in := &g.insts[i]
		if in.Interactive || in.AnimationIndex < 0 {
			continue
		}
		if _, ok := g.objectAnimationFrame(*in); !ok {
			continue
		}
		in.AnimationTick++
		if in.AnimationTick >= objectAnimationTicksPerFrame {
			in.AnimationTick = 0
			in.AnimationFrame++
		}
	}
}

// objectAnimationFrame resolves the current frame of a placed object's
// animation. The frame counter is kept on the instance so objects created at
// different times do not have to share a phase.
func (g *Game) objectAnimationFrame(in objectInst) (apacked.Frame, bool) {
	if g.objectAnimations == nil || in.AnimationIndex < 0 {
		return apacked.Frame{}, false
	}
	return g.objectAnimations.Frame(int(in.AnimationIndex), in.AnimationFrame)
}

// objectImage returns the image and authored draw offsets for an object. An
// invalid or unavailable animation falls back to the object's static sprite.
func (g *Game) objectImage(in objectInst) (*ebiten.Image, int, int) {
	if !in.Interactive {
		if frame, ok := g.objectAnimationFrame(in); ok &&
			frame.ImageBank == objectImagelistID && frame.ImageIndex >= 0 {
			if spr := g.objectSprite(int(frame.ImageIndex)); spr != nil {
				return spr.img, int(frame.OffsetX), int(frame.OffsetY)
			}
		}
	}
	if spr := g.objectSprite(in.ObjID); spr != nil {
		return spr.img, 0, 0
	}
	return nil, 0, 0
}
