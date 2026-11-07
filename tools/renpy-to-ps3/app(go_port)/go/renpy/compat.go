package renpy

import (
	"bytes"
	"compress/zlib"
	"io"
	"unicode/utf8"
)

func latin1(b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func latin1Bytes(s string) []byte {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		b = append(b, byte(r))
	}
	return b
}

func utf8Slice(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return string(b)
}

// inflate a zlib stream (RFC 1950).
func inflate(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
