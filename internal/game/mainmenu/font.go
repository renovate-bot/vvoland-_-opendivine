// SPDX-License-Identifier: GPL-3.0-only

package mainmenu

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"grono.dev/opendivine/assets"
)

// uncial is the parsed embedded display font, shared by every face.
var uncial *sfnt.Font

func init() {
	f, err := opentype.Parse(assets.UncialAntiquaTTF)
	if err != nil {
		log.Fatalf("mainmenu: parsing embedded font: %v", err)
	}
	uncial = f
}

// faceCache memoises font faces by pixel size; textCache memoises
// rendered (white) text images by size+string. The menu draw loop is
// single-threaded under ebiten, so no locking is needed.
var (
	faceCache = map[int]font.Face{}
	textCache = map[textKey]*ebiten.Image{}
)

type textKey struct {
	px int
	s  string
}

// faceAt returns an UncialAntiqua face rendered at px pixels tall.
func faceAt(px int) font.Face {
	if px < 1 {
		px = 1
	}
	if f, ok := faceCache[px]; ok {
		return f
	}
	f, err := opentype.NewFace(uncial, &opentype.FaceOptions{
		Size:    float64(px),
		DPI:     72, // 72 DPI => Size is in pixels
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("mainmenu: building font face: %v", err)
	}
	faceCache[px] = f
	return f
}

// textImage rasterises s in white on a transparent background at the
// given pixel height, cached. White lets drawTinted recolour it.
func textImage(s string, px int) *ebiten.Image {
	key := textKey{px, s}
	if img, ok := textCache[key]; ok {
		return img
	}
	face := faceAt(px)
	m := face.Metrics()
	w := font.MeasureString(face, s).Ceil()
	h := (m.Ascent + m.Descent).Ceil()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	d := font.Drawer{
		Dst:  rgba,
		Src:  image.White,
		Face: face,
		Dot:  fixed.Point26_6{X: 0, Y: m.Ascent},
	}
	d.DrawString(s)
	img := ebiten.NewImageFromImage(rgba)
	textCache[key] = img
	return img
}
