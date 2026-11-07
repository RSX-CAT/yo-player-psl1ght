package renpy

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type scriptFile struct {
	Name        string
	Data        []byte
	FromArchive bool
}

// collectScripts gathers .rpyc files from the game folder, including
// subdirectories and entries stored inside .rpa archives. A loose file
// with the same path replaces the archived copy.
func collectScripts(gameDir string) []scriptFile {
	type slot struct {
		name        string
		data        []byte
		fromArchive bool
	}
	found := map[string]slot{}
	put := func(name string, data []byte, fromArchive bool) {
		name = strings.TrimPrefix(filepath.ToSlash(name), "./")
		key := strings.ToLower(name)
		if fromArchive {
			if _, ok := found[key]; ok {
				return
			}
		}
		found[key] = slot{name, data, fromArchive}
	}

	_ = filepath.WalkDir(gameDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".rpa") {
			return nil
		}
		arc, err := OpenRpa(p)
		if err != nil {
			errln("  (skip)", d.Name()+":", err)
			return nil
		}
		idx, err := arc.ReadIndex()
		if err != nil {
			errln("  (skip)", d.Name()+":", err)
			return nil
		}
		for name, segs := range idx {
			if !strings.EqualFold(filepath.Ext(name), ".rpyc") {
				continue
			}
			data, err := arc.ReadFile(segs)
			if err != nil {
				errln("  (skip)", name+":", err)
				continue
			}
			put(name, data, true)
		}
		return nil
	})

	_ = filepath.WalkDir(gameDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".rpyc") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			errln("  (skip)", relativePath(gameDir, p)+":", err)
			return nil
		}
		put(relativePath(gameDir, p), data, false)
		return nil
	})

	keys := make([]string, 0, len(found))
	for k := range found {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]scriptFile, 0, len(keys))
	for _, k := range keys {
		s := found[k]
		out = append(out, scriptFile{Name: s.name, Data: s.data, FromArchive: s.fromArchive})
	}
	return out
}

// loadScriptUnits compiles every collected .rpyc. found is the number of
// script files, fromArchive how many of those came from an .rpa.
func loadScriptUnits(gameDir string) (units [][]any, found, fromArchive int) {
	scripts := collectScripts(gameDir)
	found = len(scripts)
	for _, s := range scripts {
		if s.FromArchive {
			fromArchive++
		}
		list, err := LoadStatementsBytes(s.Data)
		if err != nil {
			errln("  (skip)", s.Name+":", err)
			continue
		}
		if list == nil {
			errln("  (skip)", s.Name+": no statement list")
			continue
		}
		units = append(units, []any(list))
	}
	return units, found, fromArchive
}
