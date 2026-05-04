// SPDX-License-Identifier: GPL-3.0-only

package game

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"grono.dev/opendivine/pkg/assets/cpacked"
)

const (
	// objectImagelistID is the imagelist containing world object sprites.
	objectImagelistID = 0
	// floorImagelistID is the imagelist containing floor and overlay tiles.
	floorImagelistID = 2
)

// floorTile returns the cached *ebiten.Image for the given floor tile ID,
// decoding it from the imagelist on first access.
func (g *Game) floorTile(id int16) *ebiten.Image {
	if img, ok := g.floorTiles[id]; ok {
		return img
	}
	img, err := decodeFloorTile(g.floorReader, int(id))
	if err != nil {
		g.floorTiles[id] = nil
		return nil
	}
	g.floorTiles[id] = img
	return img
}

// objectSprite returns the cached sprite for the given object ID, decoding it
// from the imagelist on first access.
func (g *Game) objectSprite(id int) *sprite {
	if spr, ok := g.objSprites[id]; ok {
		return spr
	}
	cell, err := g.objReader.DecodeCell(id)
	if err != nil {
		g.objSprites[id] = nil
		return nil
	}
	spr := &sprite{img: ebitenImageFromObject(cell)}
	g.objSprites[id] = spr
	return spr
}

// openImagelist parses CPackedi.<n>c + CPackedb.<n>c.
func openImagelist(gameDir string, n int) (*cpacked.Reader, error) {
	idxPath := fmt.Sprintf("%s/static/imagelists/CPackedi.%dc", gameDir, n)
	blobPath := fmt.Sprintf("%s/static/imagelists/CPackedb.%dc", gameDir, n)
	idx, err := os.ReadFile(idxPath)
	if err != nil {
		return nil, err
	}
	blob, err := os.Open(blobPath)
	if err != nil {
		return nil, err
	}
	st, _ := blob.Stat()
	return cpacked.NewReader(idx, blob, st.Size())
}

// decodeFloorTile reads a 64x64 RGB565 raw tile (flags=0) from imagelist 2 and
// returns it as an ebiten.Image.
func decodeFloorTile(r *cpacked.Reader, id int) (*ebiten.Image, error) {
	e, err := r.Entry(id)
	if err != nil {
		return nil, err
	}
	// 202 of 3363 floor tiles in CPacked.2c carry flags=1, they're
	// span-table sprites (with transparent edges, e.g. building-corner
	// floor pieces).
	// The rest are flags=0 raw 64x64 RGB565.
	if e.Flags&cpacked.FlagStandard != 0 {
		cell, err := r.DecodeCell(id)
		if err != nil {
			return nil, err
		}
		return ebitenImageFromObject(cell), nil
	}
	payload, err := r.CellPayload(id)
	if err != nil {
		return nil, err
	}
	if len(payload) != cellPx*cellPx*2 {
		return nil, fmt.Errorf("floor tile %d: got %d bytes, want %d", id, len(payload), cellPx*cellPx*2)
	}
	img := image.NewNRGBA(image.Rect(0, 0, cellPx, cellPx))
	for y := range cellPx {
		for x := range cellPx {
			i := (y*cellPx + x) * 2
			v := binary.LittleEndian.Uint16(payload[i : i+2])
			r5, g6, b5 := uint8((v>>11)&0x1f), uint8((v>>5)&0x3f), uint8(v&0x1f)
			img.SetNRGBA(x, y, color.NRGBA{
				R: (r5 << 3) | (r5 >> 2),
				G: (g6 << 2) | (g6 >> 4),
				B: (b5 << 3) | (b5 >> 2),
				A: 0xff,
			})
		}
	}
	return ebiten.NewImageFromImage(img), nil
}

// ebitenImageFromObject converts a cpacked.Cell (sparse RGB565 sprite,
// transparent pixels stored as 0) into an ebiten.Image with alpha.
func ebitenImageFromObject(c *cpacked.Cell) *ebiten.Image {
	img := image.NewNRGBA(image.Rect(0, 0, c.Width, c.Height))
	for y := range c.Height {
		for x := range c.Width {
			i := (y*c.Width + x) * 2
			v := binary.LittleEndian.Uint16(c.RGB565[i : i+2])
			if v == 0 {
				continue
			}
			r5, g6, b5 := uint8((v>>11)&0x1f), uint8((v>>5)&0x3f), uint8(v&0x1f)
			img.SetNRGBA(x, y, color.NRGBA{
				R: (r5 << 3) | (r5 >> 2),
				G: (g6 << 2) | (g6 >> 4),
				B: (b5 << 3) | (b5 >> 2),
				A: 0xff,
			})
		}
	}
	return ebiten.NewImageFromImage(img)
}
