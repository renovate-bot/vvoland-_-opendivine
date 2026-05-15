// SPDX-License-Identifier: GPL-3.0-only

package mainmenu

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Menu-text palette, eyeballed from the reference screenshot
// (16603907-divine-divinity-windows-main-menu.png). Not traced from
// div.exe: the original menu text colour is undocumented, and this is
// OpenDivine's own presentation, so the screenshot is the authority.
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

// Update polls input. Mouse hover updates m.hovered; left-click on an
// enabled button latches Pending. ESC and a click on Quit both queue
// ActionQuit.
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

// Draw paints the menu: backdrop, then the button stack as plain
// centred gold text (no boxes, matching the reference screenshot).
// Hovered enabled items brighten; disabled items render dimmed.
func (m *Menu) Draw(screen *ebiten.Image) {
	s := m.uiScale()

	if m.Backdrop != nil {
		drawBackdropImage(screen, m.Backdrop, m.w, m.h)
	} else {
		drawBackdropGradient(screen, m.w, m.h)
		m.drawTitle(screen, s)
	}

	for i, b := range m.Buttons {
		x, y, w, h := m.buttonRect(i)
		var c color.NRGBA
		switch {
		case b.Disabled:
			c = colDisabled
		case i == m.hovered:
			c = colHover
		default:
			c = colNormal
		}
		img := textImage(b.Label)
		tw := int(float64(img.Bounds().Dx()) * s)
		th := int(float64(textH) * s)
		drawTinted(screen, img,
			x+(w-tw)/2, // centre within the (centred) hit rect
			y+(h-th)/2,
			s, c)
	}

	// Version / build string, bottom-right, smaller than the items.
	vs := s * 0.6
	if vs < 1 {
		vs = 1
	}
	ver := textImage("OpenDivine v0.0.0-dev")
	pad := int(6 * s)
	drawTinted(screen, ver,
		m.w-int(float64(ver.Bounds().Dx())*vs)-pad,
		m.h-int(float64(textH)*vs)-pad,
		vs, colVersion)
}

// glyphW / textH are the fixed advance and cell height of the
// ebitenutil debug font; lineH is one menu row at scale 1.
const (
	glyphW = 6
	textH  = 14
	lineH  = textH
)

// textCache memoises rendered (white) text images. Labels and the
// version string are static, so the cache is bounded; the menu draw
// loop is single-threaded under ebiten so no locking is needed.
var textCache = map[string]*ebiten.Image{}

// textImage returns s rendered in the white debug font on a
// transparent background, cached by string.
func textImage(s string) *ebiten.Image {
	if img, ok := textCache[s]; ok {
		return img
	}
	w := max(len(s)*glyphW, 1)
	img := ebiten.NewImage(w, textH)
	ebitenutil.DebugPrintAt(img, s, 0, 0)
	textCache[s] = img
	return img
}

// uiScale is the single font magnification for the whole menu, sized
// so the button stack fills most of the window height. Clamped so it
// stays readable on tiny windows and sane on huge ones.
func (m *Menu) uiScale() float64 {
	n := len(m.Buttons)
	if n == 0 {
		return 1
	}
	s := float64(m.h) * 0.6 / float64(n*lineH)
	if s < 1 {
		s = 1
	}
	if s > 5 {
		s = 5
	}
	return s
}

// drawTinted blits a white text image at (x, y), magnified by scale
// and multiplied by c so the glyphs take c's colour (the debug font's
// black drop-shadow stays dark, keeping text legible over the
// backdrop).
func drawTinted(dst, src *ebiten.Image, x, y int, scale float64, c color.NRGBA) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.Scale(
		float32(c.R)/255, float32(c.G)/255,
		float32(c.B)/255, float32(c.A)/255)
	dst.DrawImage(src, op)
}

// drawBackdropImage scales src to fill (w, h) preserving aspect,
// letterboxing the remainder with black.
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

// drawBackdropGradient is the fallback when no real backdrop is
// available, a vertical dark-brown gradient.
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

func (m *Menu) drawTitle(dst *ebiten.Image, s float64) {
	ts := s * 1.5 // title larger than the menu items
	ti := textImage("DIVINE  DIVINITY")
	si := textImage("OpenDivine")
	y := m.h / 12
	drawTinted(dst, ti, (m.w-int(float64(ti.Bounds().Dx())*ts))/2, y, ts, colNormal)
	drawTinted(dst, si,
		(m.w-int(float64(si.Bounds().Dx())*s))/2,
		y+int(float64(textH)*ts)+int(4*s),
		s, colVersion)
}

func lerp(a, b uint8, t float32) uint8 {
	return uint8(float32(a) + (float32(b)-float32(a))*t)
}
