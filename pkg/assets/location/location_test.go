// SPDX-License-Identifier: GPL-3.0-only

package location

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func TestRoundTrip(t *testing.T) {
	in := &File{
		Tag: SubTagStory,
		Records: []Record{
			{V0: 40, V1: 58416, V3: 0, Name: "stps_hero"},
			{V0: 8000, V1: 3728, V3: 0, Name: "stps_Joram"},
			{V0: 1, V1: 2, V3: 3, Name: ""},
		},
	}
	var buf bytes.Buffer
	assert.NilError(t, Encode(&buf, in))

	out, err := Decode(&buf)
	assert.NilError(t, err)
	assert.Assert(t, cmp.DeepEqual(out, in))
}

func TestBadMagic(t *testing.T) {
	_, err := Decode(bytes.NewReader([]byte("Not a Divinity file")))
	assert.Assert(t, cmp.Equal(err == nil, false))
}

func TestRealFile(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	path := filepath.Join(gamedata, "global/location.000")

	f, err := os.Open(path)
	assert.NilError(t, err)
	defer f.Close()

	parsed, err := Decode(f)
	assert.NilError(t, err)

	t.Logf("tag=%s, %d records; first=%+v", parsed.Tag, len(parsed.Records), parsed.Records[0])
	assert.Check(t, cmp.Equal(parsed.Tag, SubTagStory))
	assert.Assert(t, cmp.Equal(len(parsed.Records) > 0, true), "zero records")
}
