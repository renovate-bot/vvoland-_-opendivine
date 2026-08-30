// SPDX-License-Identifier: GPL-3.0-only

package apacked

import (
	"encoding/binary"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDecodeAndFrame(t *testing.T) {
	index := make([]byte, 2*IndexRecordSize)
	binary.LittleEndian.PutUint32(index[0:], 0)
	binary.LittleEndian.PutUint32(index[4:], 2)
	binary.LittleEndian.PutUint32(index[8:], 0)
	binary.LittleEndian.PutUint32(index[IndexRecordSize:], 7)
	binary.LittleEndian.PutUint32(index[IndexRecordSize+4:], 1)
	binary.LittleEndian.PutUint32(index[IndexRecordSize+8:], 2*FrameRecordSize)

	frames := make([]byte, 3*FrameRecordSize)
	putFrame := func(i int, image, x, y int32) {
		o := i * FrameRecordSize
		binary.LittleEndian.PutUint32(frames[o:], uint32(image))
		binary.LittleEndian.PutUint32(frames[o+12:], uint32(x))
		binary.LittleEndian.PutUint32(frames[o+16:], uint32(y))
		binary.LittleEndian.PutUint32(frames[o+20:], ^uint32(0))
	}
	putFrame(0, 10, -2, 4)
	putFrame(1, 11, -1, 5)
	putFrame(2, 12, 0, 6)

	f, err := Decode(index, frames)
	assert.NilError(t, err)
	assert.Equal(t, len(f.Records), 2)
	assert.Equal(t, len(f.Frames), 3)

	got, ok := f.Frame(0, 2)
	assert.Assert(t, ok)
	assert.Equal(t, got.ImageBank, uint32(0))
	assert.Equal(t, got.ImageIndex, int32(10))

	got, ok = f.Frame(0, 3)
	assert.Assert(t, ok)
	assert.Equal(t, got.ImageIndex, int32(11))

	got, ok = f.Frame(1, 0)
	assert.Assert(t, ok)
	assert.Equal(t, got.ImageBank, uint32(7))
	assert.Equal(t, got.ImageIndex, int32(12))

	got, ok = f.Frame(1, 1)
	assert.Assert(t, ok)
	assert.Equal(t, got.ImageIndex, int32(12))
	_, ok = f.Frame(0, -1)
	assert.Assert(t, !ok)
}

func TestDecodeRejectsMalformedData(t *testing.T) {
	_, err := Decode(make([]byte, IndexRecordSize-1), nil)
	assert.ErrorContains(t, err, "index")

	_, err = Decode(nil, make([]byte, FrameRecordSize-1))
	assert.ErrorContains(t, err, "frames")

	index := make([]byte, IndexRecordSize)
	binary.LittleEndian.PutUint32(index[4:], 1)
	binary.LittleEndian.PutUint32(index[8:], FrameRecordSize)
	_, err = Decode(index, make([]byte, FrameRecordSize))
	assert.ErrorContains(t, err, "ends at")
}
