// SPDX-License-Identifier: GPL-3.0-only

package heroes

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"grono.dev/opendivine/internal/testutils"
)

func TestKey(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	data, err := os.ReadFile(filepath.Join(gamedata, "static/heroes/surf.key"))
	if err != nil {
		t.Fatalf("read surf.key: %v", err)
	}
	k, err := DecodeKey(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}
	if k.MaxWidth != 232 || k.MaxHeight != 253 {
		t.Errorf("max = %d×%d, want 232×253", k.MaxWidth, k.MaxHeight)
	}
	if len(k.Groups) == 0 {
		t.Fatal("expected at least one group")
	}
	t.Logf("surf.key: max=%d×%d, %d groups", k.MaxWidth, k.MaxHeight, len(k.Groups))
	// First group should be MAA0 [0..480).
	g := k.Groups[0]
	if g.Name != "MAA0" || g.Start != 0 || g.End != 480 {
		t.Errorf("groups[0] = %+v, want {MAA0, 0, 480}", g)
	}
}

func TestIDC(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	data, err := os.ReadFile(filepath.Join(gamedata, "static/heroes/surfA.idc"))
	if err != nil {
		t.Fatalf("read surfA.idc: %v", err)
	}
	if len(data)%idcRecordSize != 0 {
		t.Errorf("file size %d not a multiple of %d", len(data), idcRecordSize)
	}
	recs, err := DecodeIDC(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeIDC: %v", err)
	}
	want := len(data) / idcRecordSize
	if len(recs) != want {
		t.Errorf("records = %d, want %d", len(recs), want)
	}
	t.Logf("surfA.idc: %d records", len(recs))
	// AttachPairs tail is all 0xffff for variant-A records (the
	// anchor layer); B/D variants populate specific pair slots.
	// surfA is variant A so we expect every pair to be -1.
	for i, r := range recs {
		for j, v := range r.AttachPairs {
			if v != -1 {
				t.Errorf("rec[%d].AttachPairs[%d] = %d, want -1 for variant A", i, j, v)
				return
			}
		}
		// Width/Height should be plausible sprite sizes.
		if r.Width == 0 || r.Width > 1024 || r.Height == 0 || r.Height > 1024 {
			t.Errorf("rec[%d] dimensions %d×%d implausible", i, r.Width, r.Height)
			break
		}
	}
	// First record's offset must be 0, size matches first frame's data extent.
	if recs[0].Offset != 0 {
		t.Errorf("rec[0].Offset = %d, want 0", recs[0].Offset)
	}
}
