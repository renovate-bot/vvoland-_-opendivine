// SPDX-License-Identifier: GPL-3.0-only

// Package heroes reads the static\heroes\<class><sex>{.key,A..E.idc}
// triplets: a hero animation archive shared with APacked imagelists.
package heroes

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"

	lzo "github.com/anchore/go-lzo"
)

// IDCRecord is one 40-byte entry in an .idc sprite catalogue.
//
// Layout (matches engine reads in FUN_0050ac30 lines 224-238):
//
//	+0  Offset   uint32  byte position into the frame's group
//	                     decompression buffer (not absolute in
//	                     the .bic file)
//	+4  Size     uint32  byte size of frame data
//	+8  XMin     int16   sprite bbox left edge in COMPOSITE coords
//	+10 YMin     int16   sprite bbox top edge in composite coords
//	+12 Width    uint16  sprite bbox width
//	+14 Height   uint16  sprite bbox height (also the line-record
//	                     count — empty rows still take 8 bytes)
//	+16 AttachPairs [12]int16   6 (x,y) attach points used to
//	                     anchor sub-sprites (e.g. weapon to hand)
//	                     in composite coords; 0xffff = unused.
//
// The "hotspot" is implicit: the agent's world position (X, Y)
// corresponds to a per-class anchor (CX, CY) in composite coords
// (set by FUN_0050bb10 — see heroClassAnchors in cmd/divine).
// A frame's world top-left is therefore (X + XMin - CX,
// Y + YMin - CY), and its bbox is Width × Height.
type IDCRecord struct {
	Offset      uint32
	Size        uint32
	XMin        int16
	YMin        int16
	Width       uint16
	Height      uint16
	AttachPairs [12]int16 // 6 (x,y) pairs; -1 = unused slot
}

const idcRecordSize = 40

// DecodeIDC parses a .idc file into a flat slice of records.
func DecodeIDC(r io.Reader) ([]IDCRecord, error) {
	var out []IDCRecord
	for i := 0; ; i++ {
		var rec IDCRecord
		if err := binary.Read(r, binary.LittleEndian, &rec); err != nil {
			if err == io.EOF {
				return out, nil
			}
			if err == io.ErrUnexpectedEOF {
				return nil, fmt.Errorf("heroes: idc rec[%d] truncated: %w", i, err)
			}
			return nil, fmt.Errorf("heroes: idc rec[%d]: %w", i, err)
		}
		out = append(out, rec)
	}
}

// Group is a named sprite-id range in a .key directory.
type Group struct {
	Name  string
	Start int // first sprite id (inclusive)
	End   int // one past last sprite id
}

// Key is the parsed contents of a .key file.
type Key struct {
	MaxWidth  int
	MaxHeight int
	CenterX   int
	CenterY   int
	Groups    []Group
}

// DecodeKey parses a .key text file (CRLF-terminated, ASCII).
func DecodeKey(r io.Reader) (*Key, error) {
	out := &Key{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// "Max size W,H"
		if rest, ok := strings.CutPrefix(line, "Max size "); ok {
			w, h, err := splitTwoInts(rest)
			if err != nil {
				return nil, fmt.Errorf("heroes: max size %q: %w", line, err)
			}
			out.MaxWidth, out.MaxHeight = w, h
			continue
		}
		// "Center X,Y"
		if rest, ok := strings.CutPrefix(line, "Center "); ok {
			x, y, err := splitTwoInts(rest)
			if err != nil {
				return nil, fmt.Errorf("heroes: center %q: %w", line, err)
			}
			out.CenterX, out.CenterY = x, y
			continue
		}
		// "<name>,<start>,<end>"
		parts := strings.Split(line, ",")
		if len(parts) != 3 {
			return nil, fmt.Errorf("heroes: malformed key line %q", line)
		}
		s, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("heroes: key start %q: %w", line, err)
		}
		e, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("heroes: key end %q: %w", line, err)
		}
		out.Groups = append(out.Groups, Group{Name: parts[0], Start: s, End: e})
	}
	return out, sc.Err()
}

// BIC is a hero sprite blob with lazy per-group LZO decompression.
//
// The blob is a flat concatenation of compressed animation-group
// blocks (one per Key.Groups entry, in order):
//
//	u32 compressed_size
//	u8  lzo1x_data[compressed_size]
//
// IDCRecord.Offset is a GLOBAL byte offset into the concatenation of
// every block's decompressed buffer.  A frame can therefore reference
// pixel data that lives in any earlier block, which is how the engine
// shares pose/anim frames across direction subgroups.
type BIC struct {
	raw    []byte
	idc    []IDCRecord
	blocks []bicBlock
	cache  map[int][]byte // block index → decompressed buffer
}

type bicBlock struct {
	bicOff     int // absolute .bic offset of this block (points at compressed_size u32)
	csize      int // compressed size in .bic
	usize      int // decompressed size (probed once at OpenBIC; cheap LZO header read)
	cumOff     int // global decompressed offset where this block starts
	startFrame int // first idc record index nominally in this block
	endFrame   int // one past last
}

// OpenBIC pairs a .bic file's bytes with its key/idc metadata so frames
// can be decoded by id.  The returned BIC retains references to bic
// and idc (no copy).  Each compressed block is sized eagerly so that
// any frame can be located by a single binary search on cumOff.
func OpenBIC(bic []byte, k *Key, idc []IDCRecord) (*BIC, error) {
	if k == nil {
		return nil, errors.New("heroes: nil key")
	}
	out := &BIC{raw: bic, idc: idc, cache: map[int][]byte{}}
	off := 0
	cum := 0
	for _, g := range k.Groups {
		if off+4 > len(bic) {
			return nil, fmt.Errorf("heroes: bic truncated at group %s (off=%d)", g.Name, off)
		}
		csize := int(binary.LittleEndian.Uint32(bic[off:]))
		usize, err := lzoUncompressedSize(bic[off+4 : off+4+csize])
		if err != nil {
			return nil, fmt.Errorf("heroes: group %s lzo size: %w", g.Name, err)
		}
		out.blocks = append(out.blocks, bicBlock{
			bicOff:     off,
			csize:      csize,
			usize:      usize,
			cumOff:     cum,
			startFrame: g.Start,
			endFrame:   g.End,
		})
		off += 4 + csize
		cum += usize
	}
	return out, nil
}

// lzoUncompressedSize decompresses an LZO1X stream into successively
// larger buffers until it stops short, returning the actual produced
// size.  Used at index time so we know each block's decompressed
// extent without paying for the full decode.
func lzoUncompressedSize(src []byte) (int, error) {
	// LZO has no length header; grow until success.  In practice
	// hero blocks are <2 MB; start there and double on failure.
	cap := 2 * 1024 * 1024
	for cap <= 64*1024*1024 {
		dst := make([]byte, cap)
		n, err := lzo.Decompress(src, dst)
		if err == nil {
			return n, nil
		}
		cap *= 2
	}
	return 0, errors.New("lzo block exceeds 64 MB")
}

// Frame decodes one frame by its global frame id (matching the IDC
// record index) and returns it as an NRGBA image of the sprite's
// stored bounding box.  Transparent pixels are alpha=0.
//
// IDC offsets for the first frame of each .key group are NOT the
// frame's global decompressed offset — they encode the .bic file
// offset of that block's LZO data (BlobOff + 4).  We detect those by
// matching frameID against block boundaries and substitute the
// correct global offset (the block's cumOff, with frame at local 0).
func (b *BIC) Frame(rec IDCRecord, frameID int) (*image.NRGBA, error) {
	var blk bicBlock
	bi := -1
	for i, bb := range b.blocks {
		if frameID == bb.startFrame {
			// First frame of group: rec.Offset is bogus (back-reference
			// trick); the real position is the block start.
			bi = i
			blk = bb
			break
		}
	}
	var local int
	if bi < 0 {
		// Mid-group frame — use rec.Offset as global cumulative offset.
		gOff := int(rec.Offset)
		bi = b.findBlock(gOff)
		if bi < 0 {
			return nil, fmt.Errorf("heroes: frame %d offset %d outside any block", frameID, gOff)
		}
		blk = b.blocks[bi]
		local = gOff - blk.cumOff
	}
	if local+int(rec.Size) > blk.usize {
		return nil, fmt.Errorf("heroes: frame %d (local=%d sz=%d) crosses block %d boundary at %d",
			frameID, local, rec.Size, bi, blk.usize)
	}
	buf, err := b.blockBuffer(bi)
	if err != nil {
		return nil, err
	}
	return decodeFrame(buf[local:local+int(rec.Size)], rec)
}

// findBlock returns the index of the block that contains the given
// global decompressed offset, or -1 if outside all blocks.
func (b *BIC) findBlock(gOff int) int {
	for i, blk := range b.blocks {
		if gOff >= blk.cumOff && gOff < blk.cumOff+blk.usize {
			return i
		}
	}
	return -1
}

func (b *BIC) blockBuffer(bi int) ([]byte, error) {
	if buf, ok := b.cache[bi]; ok {
		return buf, nil
	}
	blk := b.blocks[bi]
	start := blk.bicOff + 4
	dst := make([]byte, blk.usize)
	n, err := lzo.Decompress(b.raw[start:start+blk.csize], dst)
	if err != nil {
		return nil, fmt.Errorf("heroes: block %d lzo: %w", bi, err)
	}
	if n != blk.usize {
		return nil, fmt.Errorf("heroes: block %d decompressed to %d bytes, want %d", bi, n, blk.usize)
	}
	b.cache[bi] = dst
	return dst, nil
}

// decodeFrame parses a single frame within its group buffer slice.
// Frame layout: u32 total_size, u32 pixel_data_offset, u16 hotspot_x,
// u16 hotspot_y, then hotspot_y line records (CPacked-style), then
// RGB565 pixel data at pixel_data_offset from frame start.
func decodeFrame(frame []byte, rec IDCRecord) (*image.NRGBA, error) {
	if len(frame) < 12 {
		return nil, fmt.Errorf("heroes: frame too small (%d bytes)", len(frame))
	}
	pixelDataOff := int(binary.LittleEndian.Uint32(frame[4:]))
	nLines := int(rec.Height)
	img := image.NewNRGBA(image.Rect(0, 0, int(rec.Width), int(rec.Height)))
	// Cursor walks the line table starting at byte 12.
	pos := 12
	for ly := range nLines {
		if pos+2 > len(frame) {
			return nil, fmt.Errorf("heroes: line %d header out of range (pos=%d)", ly, pos)
		}
		nSpans := int(binary.LittleEndian.Uint16(frame[pos:]))
		if nSpans == 0 {
			pos += 8 // empty line is 8 bytes
			continue
		}
		// Non-empty: u16 num_spans, u32 pixel_offset (MISALIGNED — at
		// pos+2), then N (u16 start_x, u16 length), then u16 pad.
		if pos+2+4+nSpans*4+2 > len(frame) {
			return nil, fmt.Errorf("heroes: line %d body out of range (pos=%d, n=%d)", ly, pos, nSpans)
		}
		pixOff := int(binary.LittleEndian.Uint32(frame[pos+2:]))
		spanBase := pos + 6
		for s := range nSpans {
			startX := int(binary.LittleEndian.Uint16(frame[spanBase+s*4:]))
			length := int(binary.LittleEndian.Uint16(frame[spanBase+s*4+2:]))
			pixAddr := pixelDataOff + pixOff
			if pixAddr+length*2 > len(frame) {
				return nil, fmt.Errorf("heroes: line %d span %d pixels out of range", ly, s)
			}
			for px := range length {
				v := binary.LittleEndian.Uint16(frame[pixAddr+px*2:])
				r5, g6, b5 := uint8((v>>11)&0x1f), uint8((v>>5)&0x3f), uint8(v&0x1f)
				img.SetNRGBA(startX+px, ly, color.NRGBA{
					R: (r5 << 3) | (r5 >> 2),
					G: (g6 << 2) | (g6 >> 4),
					B: (b5 << 3) | (b5 >> 2),
					A: 0xff,
				})
			}
			pixOff += length * 2
		}
		pos += 6 + nSpans*4 + 2
	}
	return img, nil
}

func splitTwoInts(s string) (int, int, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want 2 ints, got %d", len(parts))
	}
	a, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	b, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}
