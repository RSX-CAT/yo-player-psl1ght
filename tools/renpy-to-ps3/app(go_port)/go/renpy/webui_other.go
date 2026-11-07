//go:build !windows

package renpy

import (
	"io/fs"
	"os"
	"path/filepath"
)

func windowsDrives() []fsEntry {
	return nil
}

func windowsHidden(fs.DirEntry) bool {
	return false
}

func knownDesktop() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}
