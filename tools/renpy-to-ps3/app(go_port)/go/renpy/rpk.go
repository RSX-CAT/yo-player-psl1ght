package renpy

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type RpkEntry struct {
	Name string
	Data []byte
}

type RpkTocEntry struct {
	Name   string
	Offset int64
	Length int64
}

const RpkMagic = "RPK1"
const RpkVersion uint32 = 1

// normalizeRpkPath makes sure pack writes a file that ends in .rpk.
// A directory is treated as the folder to create <game>.rpk in.
func normalizeRpkPath(gameDir, out string) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return "", fmt.Errorf("missing output path")
	}
	if st, err := os.Stat(out); err == nil && st.IsDir() {
		name := filepath.Base(filepath.Clean(filepath.Dir(filepath.Clean(gameDir))))
		if name == "." || name == "" || name == string(filepath.Separator) {
			name = "game"
		}
		out = filepath.Join(out, name+".rpk")
	} else if !strings.EqualFold(filepath.Ext(out), ".rpk") {
		out += ".rpk"
	}
	dir := filepath.Dir(out)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}
	return out, nil
}

func WriteRpk(path string, entries []RpkEntry) error {
	names := make([][]byte, len(entries))
	var tocSize int64
	for i, e := range entries {
		names[i] = []byte(strings.ReplaceAll(e.Name, `\`, `/`))
		tocSize += 4 + int64(len(names[i])) + 8 + 8
	}
	blobStart := int64(4 + 4 + 4 + tocSize)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte(RpkMagic)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, RpkVersion); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(len(entries))); err != nil {
		return err
	}
	off := blobStart
	for i, e := range entries {
		if err := binary.Write(f, binary.LittleEndian, uint32(len(names[i]))); err != nil {
			return err
		}
		if _, err := f.Write(names[i]); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, uint64(off)); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, uint64(len(e.Data))); err != nil {
			return err
		}
		off += int64(len(e.Data))
	}
	for _, e := range entries {
		if _, err := f.Write(e.Data); err != nil {
			return err
		}
	}
	return nil
}

func ReadRpkToc(path string) ([]RpkTocEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, err
	}
	if string(magic) != RpkMagic {
		return nil, fmt.Errorf("not an .rpk (bad magic)")
	}
	var ver, count uint32
	if err := binary.Read(f, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	toc := make([]RpkTocEntry, 0, count)
	for i := uint32(0); i < count; i++ {
		var nlen uint32
		if err := binary.Read(f, binary.LittleEndian, &nlen); err != nil {
			return nil, err
		}
		name := make([]byte, nlen)
		if _, err := io.ReadFull(f, name); err != nil {
			return nil, err
		}
		var off, ln uint64
		if err := binary.Read(f, binary.LittleEndian, &off); err != nil {
			return nil, err
		}
		if err := binary.Read(f, binary.LittleEndian, &ln); err != nil {
			return nil, err
		}
		toc = append(toc, RpkTocEntry{Name: string(name), Offset: int64(off), Length: int64(ln)})
	}
	return toc, nil
}
