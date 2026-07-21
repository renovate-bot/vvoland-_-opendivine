// SPDX-License-Identifier: GPL-3.0-only

// Package osiris implements Divine Divinity's story-rule runtime.
package osiris

import (
	"errors"
	"fmt"
)

// Type is the low-nibble type tag used by Osiris values and handles.
type Type uint8

const (
	TypeInteger     Type = 1
	TypeNPC         Type = 4
	TypeObject      Type = 5
	TypeDialog      Type = 6
	TypeRegion      Type = 7
	TypeLocation    Type = 8
	TypeNPCClass    Type = 9
	TypeObjectClass Type = 10
	TypeDialogEvent Type = 11
	TypeEngine      Type = 12
	TypeFunction    Type = 13
	TypeReal        Type = TypeFunction
	TypeSRegion     Type = 15
)

const maxValuePayload = 1<<28 - 1

var (
	// ErrInvalidType reports a value with an unused zero or out-of-range tag.
	ErrInvalidType = errors.New("osiris: invalid value type")
	// ErrPayloadOverflow reports a value that does not fit above the type tag.
	ErrPayloadOverflow = errors.New("osiris: value payload exceeds 28 bits")
)

// Value stores an Osiris value as (payload << 4) | type.
type Value uint32

// Handle uses the same representation as Value.
type Handle = Value

// NewValue encodes payload with its Osiris type tag.
func NewValue(kind Type, payload uint32) (Value, error) {
	if kind == 0 || kind > 15 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidType, kind)
	}
	if payload > maxValuePayload {
		return 0, fmt.Errorf("%w: %#x", ErrPayloadOverflow, payload)
	}
	return Value(payload<<4 | uint32(kind)), nil
}

// Type returns the value's low-nibble type tag.
func (v Value) Type() Type {
	return Type(uint32(v) & 0xf)
}

// Payload returns the untyped 28-bit value.
func (v Value) Payload() uint32 {
	return uint32(v) >> 4
}
