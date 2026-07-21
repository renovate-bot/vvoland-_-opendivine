// SPDX-License-Identifier: GPL-3.0-only

package story

import (
	"errors"
	"fmt"
	"io"
)

const (
	maxTableRecords       = 1 << 20
	maxSignatureMaskBytes = 1 << 16
)

var (
	// ErrTooManyRecords reports an implausibly large count-prefixed table.
	ErrTooManyRecords = errors.New("story: too many records")
	// ErrBadSymbol reports an inconsistent DIVObject symbol record.
	ErrBadSymbol = errors.New("story: bad DIVObject symbol")
	// ErrBadFunction reports an inconsistent function definition.
	ErrBadFunction = errors.New("story: bad function definition")
)

// SymbolType identifies a named object in the DIVObject table.
type SymbolType uint8

const (
	SymbolNPC         SymbolType = 4
	SymbolObject      SymbolType = 5
	SymbolDialog      SymbolType = 6
	SymbolRegion      SymbolType = 7
	SymbolLocation    SymbolType = 8
	SymbolNPCClass    SymbolType = 9
	SymbolObjectClass SymbolType = 10
	SymbolDialogEvent SymbolType = 11
	SymbolEngine      SymbolType = 12
	SymbolFunction    SymbolType = 13
	SymbolSRegion     SymbolType = 15
)

// Symbol is one named DIVObject used by compiled story rules.
type Symbol struct {
	Name string
	Type SymbolType
	ID   uint32
}

// FunctionType identifies how an Osiris function participates in rules.
type FunctionType uint8

const (
	FunctionEvent FunctionType = iota + 1
	FunctionQuery
	FunctionCall
	FunctionDatabase
	FunctionProc
	FunctionSysQuery
	FunctionSysCall
)

// Signature describes a function's parameter types and query output mask.
type Signature struct {
	OutputMask     []byte
	ParameterTypes []uint32
}

// Function is one event, query, call, database, procedure, or system verb.
type Function struct {
	Prefix    [8]uint32
	ID        uint32
	Type      FunctionType
	Name      string
	Signature Signature
}

// Tables contains the named definitions before the RETE node section.
type Tables struct {
	Header    *Header
	Symbols   []Symbol
	Functions []Function
	BytesRead int64
}

// DecodeTables reads the story header, DIVObject table, and function table.
// BytesRead is the offset of the RETE node count that follows these tables.
func DecodeTables(r io.Reader) (*Tables, error) {
	rd := &reader{r: r}
	header, err := decodeHeader(rd)
	if err != nil {
		return nil, err
	}
	rd.cipher = header.Cipher

	symbols, err := decodeSymbols(rd)
	if err != nil {
		return nil, err
	}
	functions, err := decodeFunctions(rd)
	if err != nil {
		return nil, err
	}
	return &Tables{
		Header:    header,
		Symbols:   symbols,
		Functions: functions,
		BytesRead: rd.offset,
	}, nil
}

func decodeSymbols(rd *reader) ([]Symbol, error) {
	count, err := rd.u32()
	if err != nil {
		return nil, fmt.Errorf("story: read DIVObject count: %w", err)
	}
	if count > maxTableRecords {
		return nil, fmt.Errorf("%w: DIVObject count %d", ErrTooManyRecords, count)
	}

	symbols := make([]Symbol, count)
	for i := range symbols {
		name, err := rd.string()
		if err != nil {
			return nil, fmt.Errorf("story: DIVObject %d name: %w", i, err)
		}
		kind, err := rd.u8()
		if err != nil {
			return nil, fmt.Errorf("story: DIVObject %d type: %w", i, err)
		}
		kindWord, err := rd.u32()
		if err != nil {
			return nil, fmt.Errorf("story: DIVObject %d repeated type: %w", i, err)
		}
		if kindWord != uint32(kind) {
			return nil, fmt.Errorf("%w %d: type %d repeated as %d", ErrBadSymbol, i, kind, kindWord)
		}
		id, err := rd.u32()
		if err != nil {
			return nil, fmt.Errorf("story: DIVObject %d id: %w", i, err)
		}
		var reserved [8]byte
		if err := rd.bytes(reserved[:]); err != nil {
			return nil, fmt.Errorf("story: DIVObject %d reserved bytes: %w", i, err)
		}
		if reserved != [8]byte{} {
			return nil, fmt.Errorf("%w %d: nonzero reserved bytes", ErrBadSymbol, i)
		}
		symbols[i] = Symbol{Name: name, Type: SymbolType(kind), ID: id}
	}
	return symbols, nil
}

func decodeFunctions(rd *reader) ([]Function, error) {
	count, err := rd.u32()
	if err != nil {
		return nil, fmt.Errorf("story: read function count: %w", err)
	}
	if count > maxTableRecords {
		return nil, fmt.Errorf("%w: function count %d", ErrTooManyRecords, count)
	}

	functions := make([]Function, count)
	for i := range functions {
		function := &functions[i]
		for j := range function.Prefix {
			value, err := rd.u32()
			if err != nil {
				return nil, fmt.Errorf("story: function %d prefix %d: %w", i, j, err)
			}
			function.Prefix[j] = value
		}
		function.ID = function.Prefix[3]

		kind, err := rd.u8()
		if err != nil {
			return nil, fmt.Errorf("story: function %d type: %w", i, err)
		}
		if kind < byte(FunctionEvent) || kind > byte(FunctionSysCall) {
			return nil, fmt.Errorf("%w %d: type %d", ErrBadFunction, i, kind)
		}
		function.Type = FunctionType(kind)

		function.Name, err = rd.string()
		if err != nil {
			return nil, fmt.Errorf("story: function %d name: %w", i, err)
		}
		function.Signature, err = decodeSignature(rd, i)
		if err != nil {
			return nil, err
		}
	}
	return functions, nil
}

func decodeSignature(rd *reader, functionIndex int) (Signature, error) {
	maskLen, err := rd.u32()
	if err != nil {
		return Signature{}, fmt.Errorf("story: function %d output-mask length: %w", functionIndex, err)
	}
	if maskLen > maxSignatureMaskBytes {
		return Signature{}, fmt.Errorf("%w %d: output-mask length %d", ErrBadFunction, functionIndex, maskLen)
	}
	mask := make([]byte, maskLen)
	if err := rd.bytes(mask); err != nil {
		return Signature{}, fmt.Errorf("story: function %d output mask: %w", functionIndex, err)
	}

	paramCount, err := rd.u8()
	if err != nil {
		return Signature{}, fmt.Errorf("story: function %d parameter count: %w", functionIndex, err)
	}
	params := make([]uint32, paramCount)
	for i := range params {
		params[i], err = rd.u32()
		if err != nil {
			return Signature{}, fmt.Errorf("story: function %d parameter %d: %w", functionIndex, i, err)
		}
	}
	return Signature{OutputMask: mask, ParameterTypes: params}, nil
}
