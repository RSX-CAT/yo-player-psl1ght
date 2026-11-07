package renpy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type RpaSegment struct {
	Offset int64
	Length int64
	Prefix []byte
}

func (s RpaSegment) BodyLength() int64 { return s.Length - int64(len(s.Prefix)) }

type RpaArchive struct {
	Path        string
	Version     string
	IndexOffset int64
	Key         int64
}

func OpenRpa(path string) (*RpaArchive, error) {
	a := &RpaArchive{Path: path}
	if err := a.readHeader(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *RpaArchive) readHeader() error {
	f, err := os.Open(a.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	line, err := r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return err
	}
	header := strings.TrimSpace(string(bytesTrimNL(line)))
	parts := strings.Fields(header)
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "RPA-") {
		return fmt.Errorf("not a recognized RPA archive (RPA-1.0 archives use a separate .rpi index and are not supported)")
	}
	a.Version = parts[0]
	off, err := strconv.ParseInt(parts[1], 16, 64)
	if err != nil {
		return err
	}
	a.IndexOffset = off
	if len(parts) > 2 {
		a.Key, _ = strconv.ParseInt(parts[2], 16, 64)
	}
	return nil
}

func bytesTrimNL(b []byte) []byte {
	return []byte(strings.TrimRight(string(b), "\r\n"))
}

func (a *RpaArchive) ReadIndex() (map[string][]RpaSegment, error) {
	f, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	remain := st.Size() - a.IndexOffset
	if _, err := f.Seek(a.IndexOffset, io.SeekStart); err != nil {
		return nil, err
	}
	comp := make([]byte, remain)
	if _, err := io.ReadFull(f, comp); err != nil {
		return nil, err
	}
	pickled, err := inflate(comp)
	if err != nil {
		return nil, err
	}
	root, err := LoadPickle(pickled)
	if err != nil {
		return nil, err
	}
	table := asDict(root)
	if table == nil {
		return nil, fmt.Errorf("RPA index is not a dictionary as expected")
	}
	index := make(map[string][]RpaSegment, table.Len())
	for _, e := range table.Items() {
		name := coerceName(e.k)
		var segs []RpaSegment
		for _, segObj := range asSeq(e.v) {
			tup, ok := segObj.(Tuple)
			if !ok || len(tup) < 2 {
				if seq := asSeq(segObj); seq != nil && len(seq) >= 2 {
					tup = Tuple(seq)
				} else {
					continue
				}
			}
			off, _ := toInt64(tup[0])
			ln, _ := toInt64(tup[1])
			off ^= a.Key
			ln ^= a.Key
			var prefix []byte
			if len(tup) > 2 {
				prefix = coercePrefix(tup[2])
			}
			segs = append(segs, RpaSegment{Offset: off, Length: ln, Prefix: prefix})
		}
		index[name] = segs
	}
	return index, nil
}

func RpaFileSize(segs []RpaSegment) int64 {
	var total int64
	for _, s := range segs {
		total += s.Length
	}
	return total
}

func (a *RpaArchive) ReadFile(segs []RpaSegment) ([]byte, error) {
	f, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []byte
	buf := make([]byte, 81920)
	for _, seg := range segs {
		if len(seg.Prefix) > 0 {
			out = append(out, seg.Prefix...)
		}
		remaining := seg.BodyLength()
		if remaining < 0 {
			return nil, fmt.Errorf("segment length is smaller than its prefix; archive may be corrupt")
		}
		if _, err := f.Seek(seg.Offset, io.SeekStart); err != nil {
			return nil, err
		}
		for remaining > 0 {
			want := len(buf)
			if int64(want) > remaining {
				want = int(remaining)
			}
			n, err := f.Read(buf[:want])
			if n <= 0 {
				if err == nil {
					err = io.ErrUnexpectedEOF
				}
				return nil, fmt.Errorf("unexpected end of archive while reading a file: %w", err)
			}
			out = append(out, buf[:n]...)
			remaining -= int64(n)
		}
	}
	return out, nil
}

func (a *RpaArchive) ExtractAll(outputDir string, index map[string][]RpaSegment, progress func(count, total int, name string)) error {
	total := len(index)
	count := 0
	root := filepath.Clean(outputDir) + string(os.PathSeparator)
	for name, segs := range index {
		relative := filepath.FromSlash(name)
		outPath := filepath.Clean(filepath.Join(outputDir, relative))
		if !strings.HasPrefix(outPath, root) && outPath != strings.TrimSuffix(root, string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes output directory: %s", name)
		}
		if dir := filepath.Dir(outPath); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		data, err := a.ReadFile(segs)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return err
		}
		count++
		if progress != nil {
			progress(count, total, name)
		}
	}
	return nil
}

func coerceName(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case []byte:
		return string(k)
	}
	return toString(key)
}

func coercePrefix(prefix any) []byte {
	if prefix == nil {
		return nil
	}
	if b, ok := prefix.([]byte); ok {
		return b
	}
	if s, ok := prefix.(string); ok {
		return latin1Bytes(s)
	}
	return nil
}
