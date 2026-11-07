//go:build windows

package renpy

import (
	"io/fs"
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDrives     = kernel32.NewProc("GetLogicalDrives")
	shell32                  = syscall.NewLazyDLL("shell32.dll")
	procSHGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
	ole32                    = syscall.NewLazyDLL("ole32.dll")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
)

// FOLDERID_Desktop. A bitmask from GetLogicalDrives avoids Stat on A: through Z:,
// which can sit for minutes on an empty card reader or a disconnected network drive.
var folderidDesktop = syscall.GUID{
	Data1: 0xB4BFCC3A,
	Data2: 0xDB2C,
	Data3: 0x424C,
	Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41},
}

func windowsHidden(e fs.DirEntry) bool {
	info, err := e.Info()
	if err != nil {
		return false
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil {
		return false
	}
	return data.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}

func windowsDrives() []fsEntry {
	r, _, _ := procGetLogicalDrives.Call()
	mask := uint32(r)
	var out []fsEntry
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		out = append(out, fsEntry{Name: root, Path: root, IsDir: true})
	}
	return out
}

func knownDesktop() string {
	var path *uint16
	r, _, _ := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderidDesktop)),
		0,
		0,
		uintptr(unsafe.Pointer(&path)),
	)
	if r != 0 || path == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(path)))
	n := 0
	for *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(path)) + uintptr(n*2))) != 0 && n < 32768 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(path, n))
}
