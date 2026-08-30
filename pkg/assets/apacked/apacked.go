// SPDX-License-Identifier: GPL-3.0-only

// Package apacked reads the APacked animation tables in
// static/imagelists.
//
// An APackedi record selects a contiguous run of APackedb frame records.
// Each frame then selects an image in the image bank named by its record.
package apacked

import (
	"encoding/binary"
	"fmt"
)

// IndexRecordSize and FrameRecordSize are the on-disk strides of APackedi and
// APackedb records.
const (
	IndexRecordSize = 16
	FrameRecordSize = 32
)

// Record is one APackedi animation descriptor.
type Record struct {
	ImageBank   uint32
	FrameCount  uint32
	FrameOffset uint32
	Reserved    uint32
}

// Frame is one APackedb frame descriptor.
type Frame struct {
	// ImageBank is copied from the animation record that owns this frame.
	ImageBank   uint32
	ImageIndex  int32
	Width       int32
	Height      int32
	OffsetX     int32
	OffsetY     int32
	ImageHandle int32

	MirrorOffsetX int16
	MirrorOffsetY int16
	Reserved      int16
	Padding       int16
}

// File is a decoded APackedi/APackedb pair.
type File struct {
	Records []Record
	Frames  []Frame
}

// Decode parses one APackedi/APackedb pair.
func Decode(index, frames []byte) (*File, error) {
	if len(index)%IndexRecordSize != 0 {
		return nil, fmt.Errorf("apacked: index is %d bytes, not a multiple of %d", len(index), IndexRecordSize)
	}
	if len(frames)%FrameRecordSize != 0 {
		return nil, fmt.Errorf("apacked: frames are %d bytes, not a multiple of %d", len(frames), FrameRecordSize)
	}

	f := &File{
		Records: make([]Record, len(index)/IndexRecordSize),
		Frames:  make([]Frame, len(frames)/FrameRecordSize),
	}
	for i := range f.Records {
		o := i * IndexRecordSize
		f.Records[i] = Record{
			ImageBank:   binary.LittleEndian.Uint32(index[o:]),
			FrameCount:  binary.LittleEndian.Uint32(index[o+4:]),
			FrameOffset: binary.LittleEndian.Uint32(index[o+8:]),
			Reserved:    binary.LittleEndian.Uint32(index[o+12:]),
		}
		r := f.Records[i]
		if r.FrameOffset%FrameRecordSize != 0 {
			return nil, fmt.Errorf("apacked: record %d frame offset %d is not aligned to %d", i, r.FrameOffset, FrameRecordSize)
		}
		end := uint64(r.FrameOffset) + uint64(r.FrameCount)*FrameRecordSize
		if end > uint64(len(frames)) {
			return nil, fmt.Errorf("apacked: record %d frame block ends at %d, data has %d bytes", i, end, len(frames))
		}
	}
	for i := range f.Frames {
		o := i * FrameRecordSize
		f.Frames[i] = Frame{
			ImageIndex:    int32(binary.LittleEndian.Uint32(frames[o:])),
			Width:         int32(binary.LittleEndian.Uint32(frames[o+4:])),
			Height:        int32(binary.LittleEndian.Uint32(frames[o+8:])),
			OffsetX:       int32(binary.LittleEndian.Uint32(frames[o+12:])),
			OffsetY:       int32(binary.LittleEndian.Uint32(frames[o+16:])),
			ImageHandle:   int32(binary.LittleEndian.Uint32(frames[o+20:])),
			MirrorOffsetX: int16(binary.LittleEndian.Uint16(frames[o+24:])),
			MirrorOffsetY: int16(binary.LittleEndian.Uint16(frames[o+26:])),
			Reserved:      int16(binary.LittleEndian.Uint16(frames[o+28:])),
			Padding:       int16(binary.LittleEndian.Uint16(frames[o+30:])),
		}
	}
	return f, nil
}

// Frame returns frame n from animation class class.
//
// Animation frames loop by taking n modulo the class's frame count.
// Negative frame numbers are rejected because they would address before the
// frame run in the original signed-indexing path.
func (f *File) Frame(class, n int) (Frame, bool) {
	if f == nil || class < 0 || class >= len(f.Records) || n < 0 {
		return Frame{}, false
	}
	r := f.Records[class]
	if r.FrameCount == 0 {
		return Frame{}, false
	}
	frame := uint64(r.FrameOffset)/FrameRecordSize + uint64(n)%uint64(r.FrameCount)
	if frame >= uint64(len(f.Frames)) {
		return Frame{}, false
	}
	out := f.Frames[frame]
	out.ImageBank = r.ImageBank
	return out, true
}
