package renpy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIIndex(t *testing.T) {
	ts := httptest.NewServer(newUIMux())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(b), "Ren'Py to PS3") {
		t.Fatalf("status=%d body=%s", res.StatusCode, b)
	}
}

func TestUIRunInfoAndReject(t *testing.T) {
	ts := httptest.NewServer(newUIMux())
	defer ts.Close()

	bad, err := http.Post(ts.URL+"/api/run", "application/json", strings.NewReader(`{"args":["ui"]}`))
	if err != nil {
		t.Fatal(err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("ui recursion should be rejected, got %d", bad.StatusCode)
	}

	dir := t.TempDir()
	writeRpyc(t, filepath.Join(dir, "script.rpyc"), "Eileen", "Hi")
	body := `{"args":["info",` + jsonQuote(dir) + `]}`
	res, err := http.Post(ts.URL+"/api/run", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	s := string(out)
	if !strings.Contains(s, "renpy.ast.Label") || !strings.Contains(s, "__EXIT__ 0") {
		t.Fatalf("info via ui: %s", s)
	}
}

func TestResolveGameDir(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Game")
	if err := os.Mkdir(game, 0755); err != nil {
		t.Fatal(err)
	}
	got, ok := resolveGameDir(root)
	if !ok || got != game {
		t.Fatalf("root: %q %v", got, ok)
	}
	got, ok = resolveGameDir(game)
	if ok || got != game {
		t.Fatalf("game folder itself: %q %v", got, ok)
	}
	plain := t.TempDir()
	got, ok = resolveGameDir(plain)
	if ok || got != plain {
		t.Fatalf("plain: %q %v", got, ok)
	}
}

func TestUIFs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".secret"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "Visible"), 0755); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(newUIMux())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/api/fs?path=" + url.QueryEscape(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var listing fsListing
	if err := json.NewDecoder(res.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range listing.Entries {
		if e.Name == "a.txt" && !e.IsDir {
			found = true
		}
		if e.Name == ".secret" || (len(e.Name) > 0 && e.Name[0] == '.') {
			t.Fatalf("hidden entry listed: %s", e.Name)
		}
	}
	if !found {
		t.Fatalf("listing %#v", listing)
	}

	res, err = http.Get(ts.URL + "/api/fs?path=" + url.QueryEscape(dir) + "&dirs=1")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	listing = fsListing{}
	if err := json.NewDecoder(res.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	sawVisible := false
	for _, e := range listing.Entries {
		if !e.IsDir {
			t.Fatalf("dirs=1 included %s", e.Name)
		}
		if e.Name == ".secret" {
			t.Fatal("dot folder listed")
		}
		if e.Name == "Visible" {
			sawVisible = true
		}
	}
	if !sawVisible {
		t.Fatalf("visible folder missing: %#v", listing.Entries)
	}

	res, err = http.Get(ts.URL + "/api/fs?path=")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	listing = fsListing{}
	if err := json.NewDecoder(res.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if listing.Path != home {
		t.Fatalf("linux browse starts at $HOME, got %q home %q", listing.Path, home)
	}
	homeParent := browseParent(home, false)
	if homeParent == home || homeParent == "" {
		t.Fatalf("parent of home: %q", homeParent)
	}
}

func TestWindowsBrowseParent(t *testing.T) {
	if normalizeHomeDrive("c:") != `C:\` || normalizeHomeDrive(`D:\`) != `D:\` {
		t.Fatal(normalizeHomeDrive("c:"), normalizeHomeDrive(`D:\`))
	}
	if browseParent(`C:\`, true) != "::drives" || browseParent(`C:/`, true) != "::drives" {
		t.Fatal("up from home drive")
	}
	if browseParent(`C:\Users`, true) != `C:\` {
		t.Fatal(browseParent(`C:\Users`, true))
	}
	if browseParent(`C:\Users\me`, true) != `C:\Users` {
		t.Fatal(browseParent(`C:\Users\me`, true))
	}
}

func TestUIIndexMentionsWait(t *testing.T) {
	ts := httptest.NewServer(newUIMux())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	s := string(b)
	if !strings.Contains(s, "several minutes") || !strings.Contains(s, "picker-desktop") || strings.Contains(s, "picker-home") {
		t.Fatalf("picker hints: %s", s)
	}
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
