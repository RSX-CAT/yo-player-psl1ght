package renpy

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//go:embed webui/*
var webUIFiles embed.FS

var (
	uiRunMu sync.Mutex
	uiBusy  bool
)

func uiCommand(args []string) int {
	port := 8765
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" {
			if i+1 >= len(args) {
				errln("usage: renpy-to-ps3 ui [--port N]")
				return 1
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 || n > 65535 {
				errln("error: bad --port")
				return 1
			}
			port = n
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		errln("error: listen:", err)
		return 1
	}
	url := "http://" + ln.Addr().String() + "/"
	logln("Ren'Py to PS3 UI:", url)
	logln("(listening on localhost only; close this window to quit)")
	openBrowser(url)
	if err := http.Serve(ln, newUIMux()); err != nil {
		errln("error:", err)
		return 1
	}
	return 0
}

func newUIMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", uiStatic)
	mux.HandleFunc("/api/run", uiRun)
	mux.HandleFunc("/api/fs", uiFs)
	return mux
}

func uiStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(webUIFiles, "webui/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	_, _ = w.Write(data)
}

type uiRunReq struct {
	Args []string `json:"args"`
}

func uiRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req uiRunReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Args) == 0 {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if !uiArgsOK(req.Args) {
		http.Error(w, "bad args", http.StatusBadRequest)
		return
	}

	uiRunMu.Lock()
	if uiBusy {
		uiRunMu.Unlock()
		http.Error(w, "already running", http.StatusConflict)
		return
	}
	uiBusy = true
	uiRunMu.Unlock()
	defer func() {
		uiRunMu.Lock()
		uiBusy = false
		uiRunMu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	out := &flushWriter{w: w, f: flusher}

	prevOut, prevErr := Stdout, Stderr
	Stdout, Stderr = out, out
	code := Run(req.Args)
	Stdout, Stderr = prevOut, prevErr
	_, _ = io.WriteString(out, "\n__EXIT__ "+strconv.Itoa(code)+"\n")
}

func uiArgsOK(args []string) bool {
	switch strings.ToLower(args[0]) {
	case "list", "extract", "info", "ast", "atldump", "script", "compile", "pack", "rpk":
		return true
	default:
		return false
	}
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

type fsEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"dir"`
}

type fsListing struct {
	Path       string    `json:"path"`
	Parent     string    `json:"parent"`
	Desktop    string    `json:"desktop,omitempty"`
	ShowDrives bool      `json:"showDrives,omitempty"`
	Entries    []fsEntry `json:"entries"`
	Error      string    `json:"error,omitempty"`
}

func uiFs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Query().Get("path")
	dirsOnly := r.URL.Query().Get("dirs") == "1"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(listFS(p, dirsOnly))
}

func listFS(p string, dirsOnly bool) fsListing {
	p = strings.TrimSpace(p)
	if p == "" {
		p = defaultBrowsePath()
	}
	if p == "::drives" {
		return withPlaces(fsListing{Entries: windowsDrives()})
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return withPlaces(fsListing{Path: p, Error: err.Error()})
	}
	st, err := os.Stat(abs)
	if err != nil {
		return withPlaces(fsListing{Path: abs, Parent: parentPath(abs), Error: err.Error()})
	}
	if !st.IsDir() {
		abs = filepath.Dir(abs)
		st, err = os.Stat(abs)
		if err != nil || !st.IsDir() {
			return withPlaces(fsListing{Path: abs, Parent: parentPath(abs), Error: "not a folder"})
		}
	}
	ents, err := os.ReadDir(abs)
	if err != nil {
		return withPlaces(fsListing{Path: abs, Parent: parentPath(abs), Error: err.Error()})
	}
	out := fsListing{Path: abs, Parent: parentPath(abs)}
	for _, e := range ents {
		name := e.Name()
		if name == "." || name == ".." || hiddenName(name) || windowsHidden(e) {
			continue
		}
		isDir := e.IsDir()
		if dirsOnly && !isDir {
			continue
		}
		out.Entries = append(out.Entries, fsEntry{
			Name:  name,
			Path:  filepath.Join(abs, name),
			IsDir: isDir,
		})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		if out.Entries[i].IsDir != out.Entries[j].IsDir {
			return out.Entries[i].IsDir
		}
		return strings.ToLower(out.Entries[i].Name) < strings.ToLower(out.Entries[j].Name)
	})
	return withPlaces(out)
}

func withPlaces(l fsListing) fsListing {
	l.Desktop = desktopDir()
	l.ShowDrives = runtime.GOOS == "windows"
	return l
}

func hiddenName(name string) bool {
	return name != "" && name[0] == '.'
}

func existingDir(p string) string {
	if p == "" {
		return ""
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return ""
	}
	return p
}

func userHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return existingDir(h)
}

func desktopDir() string {
	return existingDir(knownDesktop())
}

// defaultBrowsePath is %HOMEDRIVE%\ on Windows (Up from there lists the other
// drive letters) and $HOME on Linux.
func defaultBrowsePath() string {
	if runtime.GOOS == "windows" {
		if d := normalizeHomeDrive(os.Getenv("HOMEDRIVE")); d != "" {
			return d
		}
		if h, err := os.UserHomeDir(); err == nil {
			if d := normalizeHomeDrive(filepath.VolumeName(h)); d != "" {
				return d
			}
		}
		return "::drives"
	}
	if h := userHome(); h != "" {
		return h
	}
	return "/"
}

// normalizeHomeDrive turns "C:" or "c:\" into "C:\".
func normalizeHomeDrive(d string) string {
	d = strings.TrimSpace(d)
	d = strings.TrimRight(d, `\/`)
	if len(d) < 2 || d[1] != ':' {
		return ""
	}
	letter := d[0]
	if letter >= 'a' && letter <= 'z' {
		letter = letter - 'a' + 'A'
	}
	if letter < 'A' || letter > 'Z' {
		return ""
	}
	return string(letter) + `:\`
}

func isWindowsDriveRoot(p string) bool {
	p = strings.TrimRight(strings.ReplaceAll(p, "/", `\`), `\`)
	if len(p) != 2 || p[1] != ':' {
		return false
	}
	letter := p[0]
	if letter >= 'a' && letter <= 'z' {
		letter = letter - 'a' + 'A'
	}
	return letter >= 'A' && letter <= 'Z'
}

func parentPath(p string) string {
	return browseParent(p, runtime.GOOS == "windows")
}

func browseParent(p string, windows bool) string {
	if windows {
		p = strings.ReplaceAll(p, "/", `\`)
		if isWindowsDriveRoot(p) {
			return "::drives"
		}
		trimmed := strings.TrimRight(p, `\`)
		i := strings.LastIndex(trimmed, `\`)
		if i < 0 {
			return "::drives"
		}
		parent := trimmed[:i]
		if isWindowsDriveRoot(parent) {
			return normalizeHomeDrive(parent)
		}
		if parent == "" {
			return "::drives"
		}
		return parent
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return parent
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
