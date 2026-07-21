// SPDX-License-Identifier: GPL-3.0-only

package story

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"grono.dev/opendivine/internal/testutils"
)

func storyHeaderFixture() []byte {
	var buf bytes.Buffer
	buf.WriteByte(0)

	description := "Osiris save file " + strings.Repeat("x", 52-len("Osiris save file "))
	buf.WriteString(description)
	buf.WriteByte(0)
	buf.Write([]byte{1, 4, 0, 0})

	var engineVersion [engineVersionBytes]byte
	copy(engineVersion[:], "2.7.68")
	buf.Write(engineVersion[:])
	_ = binary.Write(&buf, binary.LittleEndian, uint32(version14Leading))
	return buf.Bytes()
}

func TestDecodeHeader(t *testing.T) {
	t.Parallel()

	header, err := DecodeHeader(bytes.NewReader(storyHeaderFixture()))
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(header.VersionMajor, uint8(1)))
	assert.Check(t, cmp.Equal(header.VersionMinor, uint8(4)))
	assert.Check(t, cmp.Equal(header.DebugFlags, [2]uint8{}))
	assert.Check(t, cmp.Equal(header.EngineVersion, "2.7.68"))
	assert.Check(t, cmp.Equal(header.Cipher, uint8(0xad)))
	assert.Check(t, strings.HasPrefix(header.SaveDescription, "Osiris save file "))
}

func TestDecodeHeaderRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func([]byte) []byte
		want   error
	}{
		{
			name: "marker",
			mutate: func(data []byte) []byte {
				data[0] = 1
				return data
			},
			want: ErrBadHeader,
		},
		{
			name: "description",
			mutate: func(data []byte) []byte {
				copy(data[1:], "Not an Osiris file")
				return data
			},
			want: ErrBadHeader,
		},
		{
			name: "unterminated description",
			mutate: func(data []byte) []byte {
				data[saveDescriptionBytes] = 'x'
				return data
			},
			want: ErrBadHeader,
		},
		{
			name: "version",
			mutate: func(data []byte) []byte {
				data[1+saveDescriptionBytes+1] = 3
				return data
			},
			want: ErrUnsupportedVersion,
		},
		{
			name: "engine version",
			mutate: func(data []byte) []byte {
				start := 1 + saveDescriptionBytes + 4
				for i := range engineVersionBytes {
					data[start+i] = 'x'
				}
				return data
			},
			want: ErrBadHeader,
		},
		{
			name: "leading count",
			mutate: func(data []byte) []byte {
				binary.LittleEndian.PutUint32(data[len(data)-4:], 127)
				return data
			},
			want: ErrBadLeadingCount,
		},
		{
			name: "truncated",
			mutate: func(data []byte) []byte {
				return data[:len(data)-1]
			},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeHeader(bytes.NewReader(tc.mutate(storyHeaderFixture())))
			assert.Assert(t, err != nil)
			if tc.want != nil {
				assert.Assert(t, errors.Is(err, tc.want), "error %v does not match %v", err, tc.want)
			}
		})
	}
}

func TestReaderString(t *testing.T) {
	t.Parallel()

	const text = "ObjectUsed"
	encoded := make([]byte, 0, len(text)+1)
	for _, b := range append([]byte(text), 0) {
		encoded = append(encoded, b^version14Cipher)
	}

	rd := reader{r: bytes.NewReader(encoded), cipher: version14Cipher}
	got, err := rd.string()
	assert.NilError(t, err)
	assert.Equal(t, got, text)
}

func TestRealHeader(t *testing.T) {
	gamedata := testutils.TestGameData(t)
	path := filepath.Join(gamedata, "main/startup/story.000")

	f, err := os.Open(path)
	assert.NilError(t, err)
	defer f.Close()

	header, err := DecodeHeader(f)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(header.VersionMajor, uint8(1)))
	assert.Check(t, cmp.Equal(header.VersionMinor, uint8(4)))
	assert.Check(t, cmp.Equal(header.EngineVersion, "2.7.68"))
	assert.Check(t, cmp.Equal(header.Cipher, uint8(0xad)))
}
