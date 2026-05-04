// SPDX-License-Identifier: GPL-3.0-only

// agentclass reads the AgentClasses section of main\startup\data.000 - the
// engine's compiled per-character-class table.
//
// Engine refs (in div.exe):
//
//	FUN_004126c0  per-class save (writes 0x318 bytes + strings)
//	FUN_00412750  per-class load (reads 0x318 bytes + strings)
//	FUN_00422d90  AgentClasses save loop (count + classes)
//	FUN_0050ac30  consumer of the per-slot count (param_7 ≤ 20)
//
// The shipping game relies on data.000 alone - the editor sources
// dat\animidx.dat and dat\npclist.sng are NOT distributed with retail (Steam)
// builds.
// data.000 is written by the engine itself when the editor compiles class
// definitions and is then the runtime's authoritative source.
package agentclass

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// agentClassesMarker is the section header string the engine emits at
// div.exe:FUN_00502170 just before the AgentClasses payload.
const agentClassesMarker = "AgentClassesV0.935 25-02-2002"

// classRecordBytes is the fixed-size class block size (0x318)
const classRecordBytes = 0x318

// dirCountOffset is the byte offset within a class record at which the 19
// per-anim-slot direction-count bytes live (FUN_004126c0:24, loop `(int)*(char
// *)(iVar2 + 0x4c + param_1)`).
const dirCountOffset = 0x4c

// numAnimSlots is the slot count per class, hard-coded by the engine
// (FUN_004126c0 loops `iVar2 < 0x13`).
const numAnimSlots = 19

// Class is one entry in the AgentClasses table.
// We only decode fields we need; the full 0x318-byte block has many more
// (alignment ptr, behavior table, stat blocks, etc.) that we can add as the
// reimplementation grows.
type Class struct {
	Name      string              // null-stripped class name (e.g. "Hero")
	DirCounts [numAnimSlots]uint8 // per-anim-slot direction count (≤ 20)
}

// ReadHero parses just the class-0 entry of the AgentClasses section - the
// player's "Hero" class.
// We only read the fixed 0x318 class record (no trailing variable-length
// payloads), which is enough to extract the per-anim-slot direction counts.
// The trailing payloads include a virtual-call serialization at param_1+0x120
// (FUN_004126c0:14) whose byte layout depends on the behavior subclass.
// Iterating beyond class 0 requires decoding that.
func ReadHero(r io.Reader) (*Class, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("agentclass: read data.000: %w", err)
	}
	idx := bytes.Index(data, []byte(agentClassesMarker))
	if idx < 0 {
		return nil, fmt.Errorf("agentclass: section %q not found", agentClassesMarker)
	}
	// Marker is length-prefixed. Payload starts after the marker bytes (and any
	// trailing NUL the writer may have emitted).
	pos := idx + len(agentClassesMarker)
	if pos < len(data) && data[pos] == 0 {
		pos++
	}
	if pos+4 > len(data) {
		return nil, errors.New("agentclass: truncated count")
	}
	pos += 4 // skip class count u32
	if pos+classRecordBytes > len(data) {
		return nil, errors.New("agentclass: truncated class 0 record")
	}
	var c Class
	copy(c.DirCounts[:], data[pos+dirCountOffset:pos+dirCountOffset+numAnimSlots])
	pos += classRecordBytes
	// Class name is the first trailing string after the record.
	name, _, err := readLPString(data, pos)
	if err != nil {
		return nil, fmt.Errorf("agentclass: class 0 name: %w", err)
	}
	c.Name = stripNul(name)
	return &c, nil
}

// readLPString reads a u32 length-prefixed string starting at pos.
func readLPString(data []byte, pos int) (string, int, error) {
	if pos+4 > len(data) {
		return "", pos, errors.New("EOF in length")
	}
	n := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	if n > 1<<16 {
		return "", pos, fmt.Errorf("string len %d implausible", n)
	}
	if pos+int(n) > len(data) {
		return "", pos, fmt.Errorf("EOF in body (n=%d)", n)
	}
	s := string(data[pos : pos+int(n)])
	pos += int(n)
	return s, pos, nil
}

// stripNul drops a single trailing NUL byte if present.
func stripNul(s string) string {
	if len(s) > 0 && s[len(s)-1] == 0 {
		return s[:len(s)-1]
	}
	return s
}
