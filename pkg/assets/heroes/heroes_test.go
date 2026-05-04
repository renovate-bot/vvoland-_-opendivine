// SPDX-License-Identifier: GPL-3.0-only

package heroes

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestKey(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	data, err := os.ReadFile(filepath.Join(gamedata, "static/heroes/surf.key"))
	assert.NilError(t, err)

	k, err := DecodeKey(bytes.NewReader(data))
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(k.MaxWidth, 232))
	assert.Check(t, cmp.Equal(k.MaxHeight, 253))
	assert.Assert(t, cmp.Equal(len(k.Groups) > 0, true), "expected at least one group")
	t.Logf("surf.key: max=%d×%d, %d groups", k.MaxWidth, k.MaxHeight, len(k.Groups))

	// First group should be MAA0 [0..480).
	g := k.Groups[0]
	assert.Check(t, cmp.Equal(g.Name, "MAA0"))
	assert.Check(t, cmp.Equal(g.Start, 0))
	assert.Check(t, cmp.Equal(g.End, 480))
}

func TestIDC(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	data, err := os.ReadFile(filepath.Join(gamedata, "static/heroes/surfA.idc"))
	assert.NilError(t, err)

	assert.Check(t, cmp.Equal(len(data)%idcRecordSize, 0))
	recs, err := DecodeIDC(bytes.NewReader(data))
	assert.NilError(t, err)

	want := len(data) / idcRecordSize
	assert.Check(t, cmp.Len(recs, want))
	t.Logf("surfA.idc: %d records", len(recs))

	// AttachPairs tail is all 0xffff for variant-A records (the anchor layer);
	// B/D variants populate specific pair slots.
	// surfA is variant A so we expect every pair to be -1.
	for i, r := range recs {
		for j, v := range r.AttachPairs {
			assert.Check(t, cmp.Equal(v, int16(-1)), "rec[%d].AttachPairs[%d] mismatch", i, j)
		}
		// Width/Height should be plausible sprite sizes.
		assert.Check(t, cmp.Equal(r.Width > 0 && r.Width <= 1024, true), "rec[%d] width implausible: %d", i, r.Width)
		assert.Check(t, cmp.Equal(r.Height > 0 && r.Height <= 1024, true), "rec[%d] height implausible: %d", i, r.Height)
	}
	// First record's offset must be 0, size matches first frame's data extent.
	assert.Check(t, cmp.Equal(recs[0].Offset, uint32(0)))
}
