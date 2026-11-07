package renpy

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// LoadRpycPickle returns the decompressed pickle bytes of a .rpyc file.
func LoadRpycPickle(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decompressRpyc(raw)
}

// LoadAst unpickles a .rpyc into an object tree.
func LoadAst(path string) (any, error) {
	b, err := LoadRpycPickle(path)
	if err != nil {
		return nil, err
	}
	return LoadPickle(b)
}

// LoadStatements returns the top-level statement list of a script file.
func LoadStatements(path string) (List, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadStatementsBytes(raw)
}

// LoadStatementsBytes returns the top-level statement list of .rpyc bytes.
func LoadStatementsBytes(raw []byte) (List, error) {
	b, err := decompressRpyc(raw)
	if err != nil {
		return nil, err
	}
	root, err := LoadPickle(b)
	if err != nil {
		return nil, err
	}
	return FindStatementList(root), nil
}

func FindStatementList(root any) List {
	if t, ok := root.(Tuple); ok {
		for _, part := range t {
			if l := asSeq(part); l != nil && looksLikeStatements(l) {
				return List(l)
			}
		}
	}
	if l := asSeq(root); l != nil && looksLikeStatements(l) {
		return List(l)
	}
	return nil
}

func looksLikeStatements(l []any) bool {
	for _, item := range l {
		if p := asPy(item); p != nil && strings.HasPrefix(p.ClassName, "renpy.ast.") {
			return true
		}
	}
	return false
}

func decompressRpyc(raw []byte) ([]byte, error) {
	if len(raw) >= 10 && string(raw[:10]) == "RENPY RPC2" {
		pos := 10
		for pos+12 <= len(raw) {
			slot := int(int32(binary.LittleEndian.Uint32(raw[pos:])))
			start := int(int32(binary.LittleEndian.Uint32(raw[pos+4:])))
			length := int(int32(binary.LittleEndian.Uint32(raw[pos+8:])))
			pos += 12
			if slot == 0 {
				break
			}
			if slot == 1 {
				return inflate(raw[start : start+length])
			}
		}
		return nil, fmt.Errorf("RPYC2 container has no data slot (1)")
	}
	return inflate(raw)
}
