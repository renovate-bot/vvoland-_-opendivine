// SPDX-License-Identifier: GPL-3.0-only

package story

import (
	"encoding/binary"
	"errors"
	"io"
)

const maxStringBytes = 1 << 20

var errStringTooLong = errors.New("story: string exceeds 1 MiB")

type reader struct {
	r      io.Reader
	cipher byte
}

func (r *reader) u8() (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r.r, b[:])
	return b[0], err
}

func (r *reader) u32() (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r.r, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func (r *reader) bytes(dst []byte) error {
	_, err := io.ReadFull(r.r, dst)
	return err
}

func (r *reader) string() (string, error) {
	buf := make([]byte, 0, 32)
	for range maxStringBytes {
		b, err := r.u8()
		if err != nil {
			return "", err
		}
		b ^= r.cipher
		if b == 0 {
			return string(buf), nil
		}
		buf = append(buf, b)
	}
	return "", errStringTooLong
}
