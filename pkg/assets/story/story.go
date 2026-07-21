// SPDX-License-Identifier: GPL-3.0-only

// Package story reads Divine Divinity's compiled Osiris story database.
package story

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	saveDescriptionBytes = 53
	engineVersionBytes   = 128
	version14Cipher      = 0xad
	version14Leading     = 128
)

var (
	// ErrBadHeader reports a stream that is not an Osiris save file.
	ErrBadHeader = errors.New("story: bad Osiris header")
	// ErrUnsupportedVersion reports an Osiris save version whose string cipher
	// is not known.
	ErrUnsupportedVersion = errors.New("story: unsupported save version")
	// ErrBadLeadingCount reports an unexpected version-1.4 header count.
	ErrBadLeadingCount = errors.New("story: bad leading count")
)

// Header is the fixed prefix of story.000 before the DIVObject table count.
type Header struct {
	SaveDescription string
	VersionMajor    uint8
	VersionMinor    uint8
	DebugFlags      [2]uint8
	EngineVersion   string
	Cipher          uint8
}

// DecodeHeader reads and validates the fixed story.000 header.
// It leaves r positioned at the DIVObject table count at offset 190.
func DecodeHeader(r io.Reader) (*Header, error) {
	rd := &reader{r: r}
	return decodeHeader(rd)
}

func decodeHeader(rd *reader) (*Header, error) {
	marker, err := rd.u8()
	if err != nil {
		return nil, fmt.Errorf("story: read marker: %w", err)
	}
	if marker != 0 {
		return nil, fmt.Errorf("%w: marker %#x", ErrBadHeader, marker)
	}

	var description [saveDescriptionBytes]byte
	if err := rd.bytes(description[:]); err != nil {
		return nil, fmt.Errorf("story: read save description: %w", err)
	}
	if description[len(description)-1] != 0 {
		return nil, fmt.Errorf("%w: unterminated save description", ErrBadHeader)
	}
	descriptionText := string(description[:len(description)-1])
	if !strings.HasPrefix(descriptionText, "Osiris save file ") {
		return nil, fmt.Errorf("%w: description %q", ErrBadHeader, descriptionText)
	}

	major, err := rd.u8()
	if err != nil {
		return nil, fmt.Errorf("story: read major version: %w", err)
	}
	minor, err := rd.u8()
	if err != nil {
		return nil, fmt.Errorf("story: read minor version: %w", err)
	}
	if major != 1 || minor != 4 {
		return nil, fmt.Errorf("%w: %d.%d", ErrUnsupportedVersion, major, minor)
	}
	rd.cipher = version14Cipher

	var debug [2]uint8
	if err := rd.bytes(debug[:]); err != nil {
		return nil, fmt.Errorf("story: read debug flags: %w", err)
	}

	var engineVersion [engineVersionBytes]byte
	if err := rd.bytes(engineVersion[:]); err != nil {
		return nil, fmt.Errorf("story: read engine version: %w", err)
	}
	end := bytes.IndexByte(engineVersion[:], 0)
	if end < 0 {
		return nil, fmt.Errorf("%w: unterminated engine version", ErrBadHeader)
	}

	leading, err := rd.u32()
	if err != nil {
		return nil, fmt.Errorf("story: read leading count: %w", err)
	}
	if leading != version14Leading {
		return nil, fmt.Errorf("%w: have %d want %d", ErrBadLeadingCount, leading, version14Leading)
	}

	return &Header{
		SaveDescription: descriptionText,
		VersionMajor:    major,
		VersionMinor:    minor,
		DebugFlags:      debug,
		EngineVersion:   string(engineVersion[:end]),
		Cipher:          rd.cipher,
	}, nil
}
