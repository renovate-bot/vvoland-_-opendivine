// SPDX-License-Identifier: GPL-3.0-only

// Package tga decodes the subset of Targa (TGA) images that ship with
// Divine Divinity's `static\back*.tga` and `static\load_screen_*.tga`
// files: uncompressed 24-bit BGR (image type 2).
//
// We decode straight to image.NRGBA so the result can be wrapped in
// an *ebiten.Image without further conversion.
// 16-bit, RLE, and the 32-bit alpha variant are not implemented yet.
package tga

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
)

// header is the 18-byte TGA header. Only the fields we read are
// named; the rest are skipped.
type header struct {
	IDLength      uint8
	ColorMapType  uint8
	ImageType     uint8
	CMapFirst     uint16
	CMapLen       uint16
	CMapDepth     uint8
	XOrigin       uint16
	YOrigin       uint16
	Width         uint16
	Height        uint16
	BitsPerPixel  uint8
	ImageDescript uint8
}

// Decode reads a 24-bit uncompressed BGR TGA.
func Decode(r io.Reader) (*image.NRGBA, error) {
	var h header
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("tga header: %w", err)
	}
	if h.ImageType != 2 {
		return nil, fmt.Errorf("tga: unsupported image type %d (only 2 = uncompressed truecolor)", h.ImageType)
	}
	if h.BitsPerPixel != 24 {
		return nil, fmt.Errorf("tga: unsupported bit depth %d (only 24)", h.BitsPerPixel)
	}
	if h.ColorMapType != 0 {
		return nil, errors.New("tga: color-mapped images not supported")
	}
	// Skip image ID + colour map (both empty in shipped TGAs but
	// handle nonzero for robustness).
	if _, err := io.CopyN(io.Discard, r, int64(h.IDLength)); err != nil {
		return nil, fmt.Errorf("tga: skip ID: %w", err)
	}
	if h.CMapLen != 0 {
		cmBytes := int64(h.CMapLen) * int64((h.CMapDepth+7)/8)
		if _, err := io.CopyN(io.Discard, r, cmBytes); err != nil {
			return nil, fmt.Errorf("tga: skip colormap: %w", err)
		}
	}

	w, hgt := int(h.Width), int(h.Height)
	pix := make([]byte, 3*w*hgt)
	if _, err := io.ReadFull(r, pix); err != nil {
		return nil, fmt.Errorf("tga: pixel data: %w", err)
	}

	// Origin: ImageDescript bit 5 (0x20) = top-left origin; clear =
	// bottom-left (the writer-default and the orientation shipped in
	// every divinity TGA observed). When bit 5 is clear we flip rows
	// during the BGR -> RGBA copy.
	topLeft := h.ImageDescript&0x20 != 0

	out := image.NewNRGBA(image.Rect(0, 0, w, hgt))
	for y := range hgt {
		srcRow := y
		if !topLeft {
			srcRow = hgt - 1 - y
		}
		src := pix[srcRow*w*3 : srcRow*w*3+w*3]
		dst := out.Pix[y*out.Stride : y*out.Stride+w*4]
		for x := range w {
			b := src[x*3+0]
			g := src[x*3+1]
			r := src[x*3+2]
			dst[x*4+0] = r
			dst[x*4+1] = g
			dst[x*4+2] = b
			dst[x*4+3] = 0xff
		}
	}
	return out, nil
}
