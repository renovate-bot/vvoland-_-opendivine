// SPDX-License-Identifier: GPL-3.0-only

package cpacked

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestRealImagelist(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	idxPath := filepath.Join(gamedata, "static/imagelists/CPackedi.0c")
	blobPath := filepath.Join(gamedata, "static/imagelists/CPackedb.0c")
	idx, err := os.ReadFile(idxPath)
	assert.NilError(t, err)

	blobFile, err := os.Open(blobPath)
	assert.NilError(t, err)
	defer blobFile.Close()

	st, _ := blobFile.Stat()

	r, err := NewReader(idx, blobFile, st.Size())
	assert.NilError(t, err)
	t.Logf("imagelist: %d entries", r.Count())

	// Decompress every entry - proof the LZO pipeline works for the whole
	// shipped imagelist, not just the first few.
	for i := range r.Count() {
		payload, err := r.CellPayload(i)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Equal(len(payload) > 0, true), "entry %d decompressed to zero bytes", i)
	}
	t.Logf("decompressed all %d cells cleanly", r.Count())

	// Decode a sample of standard-flag cells fully into RGB565.
	tested := 0
	for i := 0; i < r.Count() && tested < 50; i++ {
		e, _ := r.Entry(i)
		if e.Flags != FlagStandard {
			continue
		}
		c, err := r.DecodeCell(i)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(c.Width, int(e.Width)), "entry %d width mismatch", i)
		assert.Check(t, cmp.Equal(c.Height, int(e.Height)), "entry %d height mismatch", i)
		assert.Check(t, cmp.Len(c.RGB565, c.Width*c.Height*2), "entry %d raster size mismatch", i)
		tested++
	}
	t.Logf("decoded %d standard-flag cells into RGB565", tested)
}

func TestBadIndexAlignment(t *testing.T) {
	idx := bytes.Repeat([]byte{0}, EntrySize-1)
	_, err := NewReader(idx, bytes.NewReader(nil), 0)
	assert.Assert(t, cmp.ErrorContains(err, ErrIndexAlignment.Error()))
}

func TestEmpty(t *testing.T) {
	r, err := NewReader(nil, bytes.NewReader(nil), 0)
	assert.NilError(t, err)
	assert.Assert(t, cmp.Equal(r.Count(), 0))
}
