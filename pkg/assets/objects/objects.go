// SPDX-License-Identifier: GPL-3.0-only

// Package objects reads Divine Divinity's static\objects.000:
// the global object catalogue (148-byte struct x 7208 entries, parallel to
// CPacked imagelist 0).
package objects

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// EntrySize is the on-disk stride.
const EntrySize = 0x94

// Static-behaviour bit indices in `Object.SBFlags`.
const (
	SBSleep             = 1 << 0
	SBTransparent       = 1 << 1
	SBShadow            = 1 << 2
	SBUseClass          = 1 << 3
	SBRealBlack         = 1 << 4
	SBForceBackwall     = 1 << 5
	SBForceLeftwall     = 1 << 6
	SBForceFloor        = 1 << 7
	SBNoLookThrough     = 1 << 8
	SBTwinkle           = 1 << 9
	SBBow               = 1 << 10
	SBDirectMove        = 1 << 11
	SBAmbientSound      = 1 << 12
	SBLightBlocker      = 1 << 13
	SBLightBridge       = 1 << 14
	SBDontLoopAnimation = 1 << 15
	SBMakeFloating      = 1 << 16
	SBWalkOn            = 1 << 17
	SBAdditive          = 1 << 18
	SBNeedPerfectMatch  = 1 << 19
	SBShowInObjectBox   = 1 << 20
	SBPutOn             = 1 << 21
)

// Object is one parsed entry of objects.000.
type Object struct {
	// Header — pre-name fields.
	FlagsA         uint32   // +0x00 — s_* state bits
	SubValues      [16]byte // +0x04 — packed value pool referenced by FlagsA bits
	Weight         uint32   // +0x14
	AnimationIndex int32    // +0x18 — index into APacked anim sets; -1 = no anim
	SBFlags        uint32   // +0x1c — 22-bit static-behaviour bitfield (see SB* consts)

	// Name — 32-byte NUL-padded ASCII at +0x20.
	Name string

	// Post-name fields.
	ID                     uint32 // +0x30 — identical to file index, written by loader
	Class                  uint32 // +0x34
	BreakAnimationIndex    uint32 // +0x38
	ClothingCode           string // +0x3c — 16-byte string in struct, NUL-padded
	FloatingImageIndex     uint32 // +0x4c
	FloatingListIndex      uint32 // +0x50
	FloatingHighlightIndex uint32 // +0x54
	FloatingPressedIndex   uint32 // +0x58
	FloatingDisabledIndex  uint32 // +0x5c

	// Sprite-local pixel coordinates of the object's ground / cube anchor.
	// The engine adds these to the object's world (X, Y) to produce the
	// spatial-hash bucket key (FUN_00582890) and draws the sprite so this pixel
	// sits at world (X, Y).
	// The +0x60..+0x63 pair is initialised to (-1, -1) for every
	// shipped entry, and +0x64/+0x66 carry the real anchor.

	AnchorX            int16  // +0x64
	AnchorY            int16  // +0x66
	WeaponAnimation    uint32 // +0x68
	TradePriority      uint32 // +0x6c
	FloatingGroup      uint32 // +0x70
	AutomapEntry       uint32 // +0x7c
	BridgePatchXOffset int16  // +0x80
	BridgePatchYOffset int16  // +0x82
	BridgePatchXSize   int16  // +0x84
	BridgePatchYSize   int16  // +0x86
}

// Catalog is the whole objects.000 file — a flat slice indexed by id.
type Catalog struct {
	Entries []Object
}

// Errors reported by this package.
var (
	ErrSizeNotMultiple = errors.New("objects: file size is not a multiple of 148")
	ErrShortRead       = errors.New("objects: short read")
)

// Decode parses an objects.000 file from r.
func Decode(r io.Reader) (*Catalog, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(buf)%EntrySize != 0 {
		return nil, fmt.Errorf("%w: %d bytes", ErrSizeNotMultiple, len(buf))
	}
	n := len(buf) / EntrySize
	out := &Catalog{Entries: make([]Object, n)}
	for i := range n {
		entry := buf[i*EntrySize : (i+1)*EntrySize]
		decodeEntry(&out.Entries[i], entry)
	}
	return out, nil
}

func decodeEntry(o *Object, entry []byte) {
	le := binary.LittleEndian

	o.FlagsA = le.Uint32(entry[0x00:])
	copy(o.SubValues[:], entry[0x04:0x14])
	o.Weight = le.Uint32(entry[0x14:])
	o.AnimationIndex = int32(le.Uint32(entry[0x18:]))
	o.SBFlags = le.Uint32(entry[0x1c:])
	o.Name = readNulPadded(entry[0x20:0x40])
	o.ID = le.Uint32(entry[0x30:])
	o.Class = le.Uint32(entry[0x34:])
	o.BreakAnimationIndex = le.Uint32(entry[0x38:])
	o.ClothingCode = readNulPadded(entry[0x3c:0x4c])
	o.FloatingImageIndex = le.Uint32(entry[0x4c:])
	o.FloatingListIndex = le.Uint32(entry[0x50:])
	o.FloatingHighlightIndex = le.Uint32(entry[0x54:])
	o.FloatingPressedIndex = le.Uint32(entry[0x58:])
	o.FloatingDisabledIndex = le.Uint32(entry[0x5c:])
	o.AnchorX = int16(le.Uint16(entry[0x64:]))
	o.AnchorY = int16(le.Uint16(entry[0x66:]))
	o.WeaponAnimation = le.Uint32(entry[0x68:])
	o.TradePriority = le.Uint32(entry[0x6c:])
	o.FloatingGroup = le.Uint32(entry[0x70:])
	o.AutomapEntry = le.Uint32(entry[0x7c:])
	o.BridgePatchXOffset = int16(le.Uint16(entry[0x80:]))
	o.BridgePatchYOffset = int16(le.Uint16(entry[0x82:]))
	o.BridgePatchXSize = int16(le.Uint16(entry[0x84:]))
	o.BridgePatchYSize = int16(le.Uint16(entry[0x86:]))
}

// HasSB returns whether the entry has a particular static-behaviour
// flag set. Use one of the SB* constants.
func (o Object) HasSB(mask uint32) bool { return o.SBFlags&mask != 0 }

func readNulPadded(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
