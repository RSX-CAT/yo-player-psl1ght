package renpy

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeRpyc(t *testing.T, path, who, what string) {
	t.Helper()
	py := fmt.Sprintf(`
import pickle, zlib, types, sys
renpy = types.ModuleType("renpy")
ast = types.ModuleType("renpy.ast")
renpy.ast = ast
sys.modules["renpy"] = renpy
sys.modules["renpy.ast"] = ast
def make(name):
    cls = type(name, (), {})
    cls.__module__ = "renpy.ast"
    setattr(ast, name, cls)
    return cls
Say, Label = make("Say"), make("Label")
class N:
    def __init__(self, cls, state):
        self.cls = cls; self.state = state
    def __reduce__(self):
        return (self.cls, (), self.state)
script = N(Label, (None, {"name": "start", "block": [N(Say, (None, {"who": %q, "what": %q}))]}))
raw = pickle.dumps((1, "k", [script]), protocol=2)
open(%q, "wb").write(zlib.compress(raw))
`, who, what, path)
	out, err := exec.Command("python3", "-c", py).CombinedOutput()
	if err != nil {
		t.Fatalf("python rpyc: %v\n%s", err, out)
	}
}

func writeRpa(t *testing.T, rpaPath string, files map[string]string) {
	t.Helper()
	py := `
import pickle, zlib, os, sys
rpa, key = sys.argv[1], 0x12345678
pairs = sys.argv[2:]
blobs = []
for i in range(0, len(pairs), 2):
    blobs.append((pairs[i], open(pairs[i+1], "rb").read()))
header_len = len("RPA-3.0 %016x %08x\n" % (0, key))
index = {}
body = b""
for name, data in blobs:
    off = header_len + len(body)
    index[name] = [(off ^ key, len(data) ^ key)]
    body += data
index_off = header_len + len(body)
header = ("RPA-3.0 %016x %08x\n" % (index_off, key)).encode()
if len(header) != header_len:
    raise SystemExit("header width changed")
open(rpa, "wb").write(header + body + zlib.compress(pickle.dumps(index, protocol=2)))
`
	args := []string{"-c", py, rpaPath}
	for name, src := range files {
		args = append(args, name, src)
	}
	out, err := exec.Command("python3", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("python rpa: %v\n%s", err, out)
	}
}

func TestScriptsInsideRpa(t *testing.T) {
	dir := t.TempDir()
	game := filepath.Join(dir, "game")
	if err := os.MkdirAll(filepath.Join(game, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(dir, "archived.rpyc")
	loose := filepath.Join(game, "script.rpyc")
	nested := filepath.Join(game, "sub", "chapter.rpyc")
	only := filepath.Join(dir, "only.rpyc")
	writeRpyc(t, archived, "Eileen", "from-archive")
	writeRpyc(t, loose, "Eileen", "from-loose")
	writeRpyc(t, nested, "Eileen", "from-sub")
	writeRpyc(t, only, "Eileen", "only-archive")
	writeRpa(t, filepath.Join(game, "archive.rpa"), map[string]string{
		"script.rpyc":      archived,
		"only-in-rpa.rpyc": only,
	})

	units, found, fromArchive := loadScriptUnits(game)
	if found != 3 || fromArchive != 1 {
		t.Fatalf("found=%d fromArchive=%d", found, fromArchive)
	}
	prog := CompileUnits(units, false)
	if _, ok := prog.Labels["start"]; !ok {
		t.Fatal("missing start label")
	}
	var texts []string
	for _, ins := range prog.Code {
		if ins.Op.String() == "Say" {
			texts = append(texts, prog.Str(ins.B))
		}
	}
	joined := strings.Join(texts, ",")
	for _, want := range []string{"from-loose", "from-sub", "only-archive"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "from-archive") {
		t.Fatal("loose script did not replace the archived copy")
	}

	os.Remove(loose)
	os.Remove(nested)
	units, found, fromArchive = loadScriptUnits(game)
	if found != 2 || fromArchive != 2 || len(units) != 2 {
		t.Fatalf("archive only: found=%d fromArchive=%d units=%d", found, fromArchive, len(units))
	}
	if Run([]string{"info", game}) != 0 {
		t.Fatal("info")
	}
	if Run([]string{"compile", game}) != 0 {
		t.Fatal("compile")
	}
}

func TestUnwrapDeepCycle(t *testing.T) {
	d := NewDict()
	node := &PyObject{ClassName: "renpy.ast.Say", State: d}
	d.Set("next", node)
	d.Set("self", Tuple{d})
	box := &listBox{items: []any{node}}
	d.Set("block", box)
	out := unwrapDeep(node)
	p, ok := out.(*PyObject)
	if !ok || p.ClassName != "renpy.ast.Say" {
		t.Fatalf("got %#v", out)
	}
	state := asDict(p.State)
	if state == nil {
		t.Fatal("missing state")
	}
	if back, _ := state.Get("next"); back != p {
		t.Fatal("next cycle was not preserved")
	}
	block, _ := state.Get("block")
	if _, ok := block.(List); !ok {
		t.Fatalf("block type %T", block)
	}
}

func TestPickleIntAndList(t *testing.T) {
	// PROTO 2, BININT1 42, STOP
	v, err := LoadPickle([]byte{0x80, 0x02, 'K', 42, '.'})
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.(int); !ok || n != 42 {
		t.Fatalf("got %#v", v)
	}

	// empty list: PROTO2 EMPTY_LIST STOP
	v, err = LoadPickle([]byte{0x80, 0x02, ']', '.'})
	if err != nil {
		t.Fatal(err)
	}
	if l, ok := v.(List); !ok || len(l) != 0 {
		t.Fatalf("empty list: %#v", v)
	}
}

func TestPicklePythonRoundtrip(t *testing.T) {
	py := `import pickle, sys
sys.stdout.buffer.write(pickle.dumps({"hello": [1, 2, "x"], "t": (True, False, None)}, protocol=2))
`
	out, err := exec.Command("python3", "-c", py).Output()
	if err != nil {
		t.Skip("python3 pickle:", err)
	}
	v, err := LoadPickle(out)
	if err != nil {
		t.Fatal(err)
	}
	d := asDict(v)
	if d == nil {
		t.Fatalf("want dict, got %T", v)
	}
	hello, ok := d.Get("hello")
	if !ok {
		t.Fatal("missing hello")
	}
	seq := asSeq(hello)
	if len(seq) != 3 {
		t.Fatalf("hello list %#v", hello)
	}
}

func TestPickleRenpyAst(t *testing.T) {
	py := `
import pickle, sys, types
renpy = types.ModuleType("renpy")
ast = types.ModuleType("renpy.ast")
renpy.ast = ast
sys.modules["renpy"] = renpy
sys.modules["renpy.ast"] = ast
Say = type("Say", (), {})
Say.__module__ = "renpy.ast"
ast.Say = Say
class Inst:
    def __reduce__(self):
        return (Say, (), (None, {"who": "Eileen", "what": "Hello"}))
sys.stdout.buffer.write(pickle.dumps((1, "key", [Inst()]), protocol=2))
`
	out, err := exec.Command("python3", "-c", py).Output()
	if err != nil {
		t.Skip(err)
	}
	v, err := LoadPickle(out)
	if err != nil {
		t.Fatal(err)
	}
	stmts := FindStatementList(v)
	if stmts == nil {
		t.Fatalf("no statements in %#v", v)
	}
	nodes := ToNodes(stmts)
	if len(nodes) != 1 || nodes[0].Short() != "Say" {
		t.Fatalf("nodes=%v", nodes)
	}
	if nodes[0].Text("who") != "Eileen" || nodes[0].Text("what") != "Hello" {
		t.Fatalf("say fields %#v %#v", nodes[0].Raw("who"), nodes[0].Raw("what"))
	}
}

func TestExprCompiler(t *testing.T) {
	p := NewIrProgram()
	ep := CompileExpr("a + 1 and True", p)
	if ep == nil {
		t.Fatal("compile failed")
	}
	if CompileExpr("foo(bar)", p) != nil {
		t.Fatal("calls should be unsupported")
	}
	if CompileExpr("x ** 2", p) != nil {
		t.Fatal("** should be unsupported")
	}
	if CompileExpr("max(1, 2, 3)", p) == nil {
		t.Fatal("max() should compile")
	}
}

func TestRpkRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.rpk")
	entries := []RpkEntry{
		{Name: "manifest", Data: []byte("hello")},
		{Name: "a/b.png", Data: []byte{1, 2, 3}},
	}
	if err := WriteRpk(path, entries); err != nil {
		t.Fatal(err)
	}
	toc, err := ReadRpkToc(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(toc) != 2 || toc[0].Name != "manifest" || toc[0].Length != 5 {
		t.Fatalf("%#v", toc)
	}
}

func TestBytecodeRoundtrip(t *testing.T) {
	p := NewIrProgram()
	p.SetLabel("start", 0)
	p.Emit(NewInstr(IrLabel, p.Intern("start")))
	p.Emit(NewInstr(IrSay, p.Intern("Eileen"), p.Intern("Hi")))
	p.Emit(NewInstr(IrEnd))
	b := WriteBytecode(p)
	d, err := ReadBytecode(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Code) != 3 || d.Strings[0] != "start" {
		t.Fatalf("%#v %#v", d.Code, d.Strings)
	}
	if d.Labels["start"] != 0 {
		t.Fatal(d.Labels)
	}
}

func TestCompileSayAndJump(t *testing.T) {
	node := func(cls string, attrs map[string]any) *PyObject {
		d := NewDict()
		for k, v := range attrs {
			d.Set(k, v)
		}
		return &PyObject{ClassName: "renpy.ast." + cls, State: Tuple{nil, d}}
	}
	start := node("Label", map[string]any{
		"name":  "start",
		"block": List{node("Say", map[string]any{"who": "e", "what": "hi"}), node("Jump", map[string]any{"target": "start"})},
	})
	prog := Compile([]any{start})
	if _, ok := prog.Labels["start"]; !ok {
		t.Fatal("missing start")
	}
	foundSay := false
	for _, ins := range prog.Code {
		if ins.Op == IrSay {
			foundSay = true
			if prog.Str(ins.B) != "hi" {
				t.Fatalf("say text %q", prog.Str(ins.B))
			}
		}
	}
	if !foundSay {
		t.Fatal("no say")
	}
}

func TestRunUsage(t *testing.T) {
	if Run(nil) != 0 {
		t.Fatal("empty args")
	}
	if Run([]string{"nope"}) != 1 {
		t.Fatal("unknown")
	}
}

func TestPackTinyGame(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	dir := t.TempDir()
	game := filepath.Join(dir, "MyGame", "game")
	if err := os.MkdirAll(game, 0755); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(game, "bg.png")
	if out, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=red:s=8x8", "-frames:v", "1", pngPath).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg png: %v\n%s", err, out)
	}
	writeRpyc(t, filepath.Join(game, "script.rpyc"), "Eileen", "Hello")
	outRpk := filepath.Join(dir, "out.rpk")
	rc := Pack(game, outRpk, ff, 1920, 1080, false, false, false, false)
	if rc != 0 {
		t.Fatalf("pack rc=%d", rc)
	}
	toc, err := ReadRpkToc(outRpk)
	if err != nil {
		t.Fatal(err)
	}
	if len(toc) < 3 {
		t.Fatalf("toc %#v", toc)
	}
	if Run([]string{"pack", game, outRpk, "--max", "64x64", "--no-cache"}) != 0 {
		t.Fatal("pack command")
	}
	logPath := filepath.Join(dir, "out.log")
	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "transcript: "+logPath) || !strings.Contains(string(body), "PACK_DONE rc=0") {
		t.Fatalf("transcript %q", body)
	}
}

func TestCLIInfoAndCompile(t *testing.T) {
	dir := t.TempDir()
	writeRpyc(t, filepath.Join(dir, "script.rpyc"), "Eileen", "Hi")
	if Run([]string{"info", dir}) != 0 {
		t.Fatal("info")
	}
	outRbc := filepath.Join(dir, "g.rbc")
	if Run([]string{"compile", dir, outRbc}) != 0 {
		t.Fatal("compile")
	}
	if _, err := os.Stat(outRbc); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "g.log")
	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "transcript: "+logPath) || !strings.Contains(string(body), "IR summary") {
		t.Fatalf("transcript %q", body)
	}
	if _, err := os.Stat(outRbc + ".log"); err == nil {
		t.Fatal("log should use the output name, not the output name plus .log")
	}
}

func TestTranscriptPath(t *testing.T) {
	if got := transcriptPath(filepath.Join("bundles", "MyGame.rpk")); filepath.Base(got) != "MyGame.log" {
		t.Fatal(got)
	}
	if got := transcriptPath(filepath.Join("bundles", "chapter.rbc")); filepath.Base(got) != "chapter.log" {
		t.Fatal(got)
	}
}

func TestNormalizeRpkPath(t *testing.T) {
	dir := t.TempDir()
	game := filepath.Join(dir, "MyGame", "game")
	if err := os.MkdirAll(game, 0755); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "bundles")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := normalizeRpkPath(game, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "MyGame.rpk" || filepath.Dir(got) != outDir {
		t.Fatalf("dir out: %s", got)
	}
	got, err = normalizeRpkPath(game, filepath.Join(dir, "bundle"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "bundle.rpk" {
		t.Fatalf("missing ext: %s", got)
	}
	want := filepath.Join(dir, "ok.RPK")
	got, err = normalizeRpkPath(game, want)
	if err != nil || got != want {
		t.Fatalf("keep rpk: %s %v", got, err)
	}
}

func TestPutAssetCaseFold(t *testing.T) {
	m := map[string][]byte{}
	putAsset(m, "Eileen.png", []byte{1})
	putAsset(m, "eileen.png", []byte{2})
	if len(m) != 1 {
		t.Fatalf("want 1 key, got %#v", m)
	}
	if got := m["Eileen.png"]; len(got) != 1 || got[0] != 2 {
		t.Fatalf("last write should win with first casing: %#v", m)
	}
}

func TestScanClassesNullMemoDoesNotShift(t *testing.T) {
	// PROTO2, "renpy", "ast", BININT1 0, MEMOIZE, BINGET 0, STACK_GLOBAL, STOP
	b := []byte{0x80, 0x02, 0x8c, 5}
	b = append(b, "renpy"...)
	b = append(b, 0x8c, 3)
	b = append(b, "ast"...)
	b = append(b, 'K', 0, 0x94, 'h', 0, 0x93, '.')
	classes := ScanClasses(b)
	if _, ok := classes["renpy.ast"]; !ok {
		t.Fatalf("missing renpy.ast in %v", classSetKeys(classes))
	}
	if _, ok := classes["ast."]; ok {
		t.Fatal("null memo shifted STACK_GLOBAL into ast.")
	}
	if _, ok := classes["ast.ast"]; ok {
		t.Fatal("null memo replayed ast as a second string")
	}
}

func TestParseMaxSize(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
		ok   bool
	}{
		{"1920x1080", 1920, 1080, true},
		{"1280×720", 1280, 720, true},
		{"768 x 576", 768, 576, true},
		{"720X480", 720, 480, true},
		{"1920", 1920, 1920, true},
		{"8", 0, 0, false},
		{"nope", 0, 0, false},
		{"1920x", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := parseMaxSize(c.in)
		if ok != c.ok || w != c.w || h != c.h {
			t.Fatalf("%q -> %d %d %v, want %d %d %v", c.in, w, h, ok, c.w, c.h, c.ok)
		}
	}
	if fitFactor(1920, 1080, 720, 480) != 0.375 {
		t.Fatal(fitFactor(1920, 1080, 720, 480))
	}
	if fitFactor(1920, 1080, 1920, 1080) != 1 {
		t.Fatal(fitFactor(1920, 1080, 1920, 1080))
	}
	if fitFactor(1280, 720, 768, 576) != float64(768)/float64(1280) {
		t.Fatal(fitFactor(1280, 720, 768, 576))
	}
}

func TestImageArgsSingleFrame(t *testing.T) {
	args := imageArgs("doc/_static/ajax-loader.gif", "out.png", "scale=8:8")
	got := strings.Join(args, " ")
	if !strings.Contains(got, "-frames:v 1") || !strings.Contains(got, "-update 1") {
		t.Fatal(args)
	}
	if args[len(args)-1] != "out.png" {
		t.Fatal(args)
	}
}

func TestGifToPng(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	dir := t.TempDir()
	gifPath := filepath.Join(dir, "ajax-loader.gif")
	pngPath := filepath.Join(dir, "out.png")
	if out, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=blue:s=8x8", "-frames:v", "4", gifPath).CombinedOutput(); err != nil {
		t.Fatalf("make gif: %v\n%s", err, out)
	}
	ok, errText := ff.Image(gifPath, pngPath, 1920, 1920)
	if !ok {
		t.Fatal(errText)
	}
	if st, err := os.Stat(pngPath); err != nil || st.Size() == 0 {
		t.Fatal(err, st)
	}
}

func TestGifPackKeepsScriptName(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	dir := t.TempDir()
	game := filepath.Join(dir, "game")
	if err := os.MkdirAll(game, 0755); err != nil {
		t.Fatal(err)
	}
	gifPath := filepath.Join(game, "ajax-loader.gif")
	if out, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=blue:s=8x8", "-frames:v", "4", gifPath).CombinedOutput(); err != nil {
		t.Fatalf("make gif: %v\n%s", err, out)
	}
	outRpk := filepath.Join(dir, "out.rpk")
	if rc := Pack(game, outRpk, ff, 720, 480, false, false, false, false); rc != 0 {
		t.Fatalf("pack rc=%d", rc)
	}
	toc, err := ReadRpkToc(outRpk)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range toc {
		names = append(names, e.Name)
		if e.Name == "assets/ajax-loader.png" {
			t.Fatal("gif was renamed, the player looks up ajax-loader.gif")
		}
	}
	found := false
	for _, n := range names {
		if n == "assets/ajax-loader.gif" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing assets/ajax-loader.gif in %v", names)
	}
	raw := readRpkEntry(t, outRpk, "assets/ajax-loader.gif")
	if len(raw) < 8 || raw[0] != 0x89 || string(raw[1:4]) != "PNG" {
		t.Fatal("default pack should store one PNG frame under the .gif name")
	}
}

func TestGifBetaKeepsAnimation(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	dir := t.TempDir()
	game := filepath.Join(dir, "game")
	if err := os.MkdirAll(game, 0755); err != nil {
		t.Fatal(err)
	}
	gifPath := filepath.Join(game, "ajax-loader.gif")
	if out, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=blue:s=8x8", "-frames:v", "4", gifPath).CombinedOutput(); err != nil {
		t.Fatalf("make gif: %v\n%s", err, out)
	}
	outRpk := filepath.Join(dir, "out.rpk")
	if rc := Pack(game, outRpk, ff, 720, 480, false, false, false, true); rc != 0 {
		t.Fatalf("pack rc=%d", rc)
	}
	raw := readRpkEntry(t, outRpk, "assets/ajax-loader.gif")
	src, err := os.ReadFile(gifPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(src) {
		t.Fatal("beta gif should pass a small animation through unchanged")
	}
}

func readRpkEntry(t *testing.T, path, name string) []byte {
	t.Helper()
	toc, err := ReadRpkToc(path)
	if err != nil {
		t.Fatal(err)
	}
	var ent *RpkTocEntry
	for i := range toc {
		if toc[i].Name == name {
			ent = &toc[i]
			break
		}
	}
	if ent == nil {
		t.Fatalf("missing %s", name)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	buf := make([]byte, ent.Length)
	if _, err := f.ReadAt(buf, ent.Offset); err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestVideoOddHeight(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	if !strings.Contains(videoScaleFilter(1920, 1080), "trunc(iw/2)*2") {
		t.Fatal(videoScaleFilter(1920, 1080))
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "attack.webm")
	out := filepath.Join(dir, "attack.mp4")
	if log, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=red:s=16x15:d=0.2", "-frames:v", "3", in).CombinedOutput(); err != nil {
		t.Fatalf("make webm: %v\n%s", err, log)
	}
	ok, errText := ff.Video(in, out, 1920, 1080)
	if !ok {
		t.Fatal(errText)
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		t.Fatalf("mp4: %v", err)
	}
}

func pngSizeHeader(w, h int) []byte {
	b := make([]byte, 24)
	copy(b, []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13})
	copy(b[12:16], []byte("IHDR"))
	b[16] = byte(w >> 24)
	b[17] = byte(w >> 16)
	b[18] = byte(w >> 8)
	b[19] = byte(w)
	b[20] = byte(h >> 24)
	b[21] = byte(h >> 16)
	b[22] = byte(h >> 8)
	b[23] = byte(h)
	return b
}

func TestWantBlurSides(t *testing.T) {
	p := &IrProgram{intern: map[string]int{}}
	p.Code = []Instr{
		NewInstr(IrImage, p.Intern("bg room"), p.Intern(`"bg/room.png"`)),
		NewInstr(IrScene, p.Intern("bg room")),
		NewInstr(IrImage, p.Intern("eileen"), p.Intern(`"eileen.png"`)),
		NewInstr(IrShow, p.Intern("eileen")),
	}
	keys := sceneImageKeys(p)
	room := pngSizeHeader(1024, 768)
	if !wantBlurSides(room, ".png", "images/bg/room.png", keys, 1024, 768, 1920, 1080) {
		t.Fatal("4:3 scene on 16:9")
	}
	if wantBlurSides(room, ".png", "images/bg/room.png", keys, 1024, 768, 768, 576) {
		t.Fatal("4:3 target should stay 4:3")
	}
	if wantBlurSides(room, ".png", "eileen.png", keys, 1024, 768, 1920, 1080) {
		t.Fatal("sprite")
	}
	if wantBlurSides(pngSizeHeader(1920, 1080), ".png", "images/bg/room.png", keys, 1920, 1080, 1920, 1080) {
		t.Fatal("16:9 game")
	}
	if !strings.Contains(blurSideFilter(1920, 1080), "boxblur") || !strings.Contains(blurSideFilter(1920, 1080), "overlay=(W-w)/2:(H-h)/2") {
		t.Fatal(blurSideFilter(1920, 1080))
	}
}

func TestImageBlurSides(t *testing.T) {
	ff, err := NewFfmpeg("")
	if err != nil {
		t.Skip("ffmpeg:", err)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "room.png")
	out := filepath.Join(dir, "room-wide.png")
	// 80x60, green on the left and blue on the right.
	src := "color=c=0x00ff00:s=40x60[l];color=c=0x0000ff:s=40x60[r];[l][r]hstack"
	if log, err := exec.Command("ffmpeg", "-y", "-filter_complex", src, "-frames:v", "1", in).CombinedOutput(); err != nil {
		t.Fatalf("make png: %v\n%s", err, log)
	}
	ok, errText := ff.ImageBlurSides(in, out, 160, 90)
	if !ok {
		t.Fatal(errText)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 160 || b.Dy() != 90 {
		t.Fatalf("size %dx%d", b.Dx(), b.Dy())
	}
	left := img.At(0, 45)
	right := img.At(159, 45)
	lr, lg, lb, _ := left.RGBA()
	rr, rg, rb, _ := right.RGBA()
	if lg <= lb || rb <= rg {
		t.Fatalf("sides left=%v right=%v", left, right)
	}
	_ = lr
	_ = rr
}

func TestGifArgsKeepFrames(t *testing.T) {
	got := strings.Join(gifArgs("a.gif", "b.gif", "scale=8:8"), " ")
	if strings.Contains(got, "-frames:v") || !strings.Contains(got, "-loop 0") {
		t.Fatal(got)
	}
}

func TestScanClassesTruncated(t *testing.T) {
	ScanClasses([]byte{0x80})
	ScanClasses([]byte{0x8c, 50})
}
