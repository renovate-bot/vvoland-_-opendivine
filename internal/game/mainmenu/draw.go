// SPDX-License-Identifier: GPL-3.0-only

package mainmenu

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	colNormal   = color.NRGBA{0xc9, 0xa6, 0x4e, 0xff} // warm gold
	colHover    = color.NRGBA{0xf3, 0xe3, 0xb0, 0xff} // bright cream
	colDisabled = color.NRGBA{0x6a, 0x5c, 0x40, 0xff} // dim gold
	colVersion  = color.NRGBA{0x8a, 0x7a, 0x50, 0xff} // subtle corner text
)

// Layout implements ebiten.Game / game.Screen.
func (m *Menu) Layout(w, h int) (int, int) {
	m.w, m.h = w, h
	return w, h
}

func (m *Menu) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.pending = ActionQuit
		return nil
	}
	mx, my := ebiten.CursorPosition()
	m.hovered = -1
	for i := range m.Buttons {
		x, y, w, h := m.buttonRect(i)
		if mx >= x && mx < x+w && my >= y && my < y+h {
			m.hovered = i
			break
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && m.hovered >= 0 {
		b := m.Buttons[m.hovered]
		if !b.Disabled {
			m.pending = b.Action
		}
	}
	return nil
}

func (m *Menu) itemPx() int {
	n := len(m.Buttons)
	if n == 0 {
		n = 1
	}
	px := m.h * 6 / (n * 14)
	px = min(max(px, 12), 72)
	return px
}

// lineH is the vertical pitch between menu rows.
func (m *Menu) lineH() int { return m.itemPx() * 14 / 10 }

func (m *Menu) Draw(screen *ebiten.Image) {
	if m.Backdrop != nil {
		drawBackdropImage(screen, m.Backdrop, m.w, m.h)
	} else {
		drawBackdropGradient(screen, m.w, m.h)
		m.drawTitle(screen)
	}

	px := m.itemPx()
	for i, b := range m.Buttons {
		_, y, _, h := m.buttonRect(i)
		var c color.NRGBA
		switch {
		case b.Disabled:
			c = colDisabled
		case i == m.hovered:
			c = colHover
		default:
			c = colNormal
		}
		img := textImage(b.Label, px)
		drawTinted(screen, img,
			(m.w-img.Bounds().Dx())/2, // centred on screen
			y+(h-img.Bounds().Dy())/2,
			c)
	}

	vpx := px * 55 / 100
	vpx = max(vpx, 11)
	ver := textImage("OpenDivine v0.0.0-dev", vpx)
	pad := vpx / 2
	drawTinted(screen, ver,
		m.w-ver.Bounds().Dx()-pad,
		m.h-ver.Bounds().Dy()-pad,
		colVersion)
}

func drawTinted(dst, src *ebiten.Image, x, y int, c color.NRGBA) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.Scale(
		float32(c.R)/255, float32(c.G)/255,
		float32(c.B)/255, float32(c.A)/255)
	dst.DrawImage(src, op)
}

func drawBackdropImage(dst, src *ebiten.Image, w, h int) {
	dst.Fill(color.NRGBA{0, 0, 0, 0xff})
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 {
		return
	}
	sx := float64(w) / float64(sw)
	sy := float64(h) / float64(sh)
	scale := sx
	if sy < sx {
		scale = sy
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(
		(float64(w)-float64(sw)*scale)/2,
		(float64(h)-float64(sh)*scale)/2,
	)
	dst.DrawImage(src, op)
}

func drawBackdropGradient(dst *ebiten.Image, w, h int) {
	top := color.NRGBA{0x14, 0x0b, 0x05, 0xff}
	bot := color.NRGBA{0x05, 0x03, 0x02, 0xff}
	// Cheap gradient: a handful of horizontal bands.
	const bands = 32
	bandH := float32(h) / bands
	for i := range bands {
		t := float32(i) / float32(bands-1)
		c := color.NRGBA{
			lerp(top.R, bot.R, t),
			lerp(top.G, bot.G, t),
			lerp(top.B, bot.B, t),
			0xff,
		}
		vector.FillRect(dst, 0, float32(i)*bandH, float32(w), bandH+1, c, false)
	}
}

func (m *Menu) drawTitle(dst *ebiten.Image) {
	tpx := m.itemPx() * 2 // title larger than the menu items
	spx := m.itemPx()
	ti := textImage("DIVINE  DIVINITY", tpx)
	si := textImage("OpenDivine", spx)
	y := m.h / 12
	drawTinted(dst, ti, (m.w-ti.Bounds().Dx())/2, y, colNormal)
	drawTinted(dst, si,
		(m.w-si.Bounds().Dx())/2,
		y+ti.Bounds().Dy()+spx/3,
		colVersion)
}

func lerp(a, b uint8, t float32) uint8 {
	return uint8(float32(a) + (float32(b)-float32(a))*t)
}
