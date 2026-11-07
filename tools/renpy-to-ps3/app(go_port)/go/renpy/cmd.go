package renpy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Log is where command output goes (defaults to stdout/stderr). The GUI replaces this.
var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

func logf(format string, args ...any) { fmt.Fprintf(Stdout, format, args...) }
func logln(args ...any)               { fmt.Fprintln(Stdout, args...) }
func errf(format string, args ...any) { fmt.Fprintf(Stderr, format, args...) }
func errln(args ...any)               { fmt.Fprintln(Stderr, args...) }

// transcriptPath names the run log after the output file.
// "MyGame.rpk" becomes "MyGame.log" in the same folder.
func transcriptPath(outputPath string) string {
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" || name == "." {
		name = base
	}
	return filepath.Join(dir, name+".log")
}

// beginTranscript copies following log lines into the transcript file.
// The returned function is safe to call when the file could not be created.
func beginTranscript(outputPath string) func() {
	path := transcriptPath(outputPath)
	f, err := os.Create(path)
	if err != nil {
		errln("error: transcript:", err)
		return func() {}
	}
	origOut, origErr := Stdout, Stderr
	Stdout = io.MultiWriter(origOut, f)
	Stderr = io.MultiWriter(origErr, f)
	logln("transcript:", path)
	return func() {
		Stdout, Stderr = origOut, origErr
		_ = f.Close()
	}
}

// Run is the command-line engine the GUI calls with the same args.
func Run(args []string) (code int) {
	defer func() {
		if rec := recover(); rec != nil {
			errln("error:", rec)
			code = 1
		}
	}()
	if len(args) == 0 {
		printUsage()
		return 0
	}
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "list":
		if len(args) < 2 {
			errln("error: missing <rpa-file>")
			printUsage()
			return 1
		}
		return listCommand(args[1])
	case "extract":
		if len(args) < 3 {
			errln("usage: renpy-to-ps3 extract <rpa-file> <output-dir>")
			return 1
		}
		return extractCommand(args[1], args[2])
	case "info":
		if len(args) < 2 {
			errln("error: missing <game-dir>")
			printUsage()
			return 1
		}
		return infoCommand(args[1])
	case "ast":
		if len(args) < 2 {
			errln("error: missing <rpyc-file>")
			return 1
		}
		label := ""
		if len(args) > 2 {
			label = args[2]
		}
		return astCommand(args[1], label)
	case "atldump":
		if len(args) < 2 {
			errln("error: missing <rpyc-file>")
			return 1
		}
		return atlDumpCommand(args[1])
	case "script":
		if len(args) < 2 {
			errln("error: missing <rpyc-file>")
			return 1
		}
		return scriptCommand(args[1])
	case "compile":
		if len(args) < 2 {
			errln("error: missing <rpyc-file-or-game-dir>")
			return 1
		}
		full := len(args) > 2 && args[2] == "full"
		var outRbc string
		if len(args) > 2 && args[2] != "full" {
			outRbc = args[2]
		} else if len(args) > 3 {
			outRbc = args[3]
		}
		return compileCommand(args[1], full, outRbc)
	case "pack":
		if len(args) < 3 {
			errln("usage: renpy-to-ps3 pack <game-dir> <out.rpk> [--max WxH] [--ascii-text] [--ffmpeg <path>] [--no-cache] [--clear-cache] [--beta gif]")
			return 1
		}
		return packCommand(args)
	case "rpk":
		if len(args) < 2 {
			errln("error: missing <rpk-file>")
			return 1
		}
		return rpkCommand(args[1])
	case "ui":
		return uiCommand(args[1:])
	default:
		errln("error: unknown command '" + cmd + "'")
		printUsage()
		return 1
	}
}

func printUsage() {
	logln("renpy-to-ps3 - Convert Ren'Py games to PS3 packages")
	logln()
	logln("Commands:")
	logln("  list <rpa-file>                         List archive contents")
	logln("  extract <rpa-file> <output>             Extract archive")
	logln("  info <game-dir>                         Show construct compatibility")
	logln("  compile <rpyc|game-dir> [out.rbc]       Compile to IR / bytecode")
	logln("  pack <game-dir> <out.rpk> [--max WxH]   Convert + compile + bundle")
	logln("       [--beta gif]                       Beta: keep animated GIF frames")
	logln("       [--no-cache] [--clear-cache]       Asset cache: bypass / wipe-then-rebuild")
	logln("  rpk <file>                              Inspect an .rpk bundle")
	logln("  ui [--port N]                           Local web UI (Windows 7 / any browser)")
	logln()
}

func listCommand(rpaPath string) int {
	if _, err := os.Stat(rpaPath); err != nil {
		errln("error: file not found:", rpaPath)
		return 1
	}
	rpa, err := OpenRpa(rpaPath)
	if err != nil {
		errln("error:", err)
		return 1
	}
	index, err := rpa.ReadIndex()
	if err != nil {
		errln("error:", err)
		return 1
	}
	logln("version :", rpa.Version)
	logln("index   : 0x" + strings.ToUpper(strconv.FormatInt(rpa.IndexOffset, 16)))
	logln("key     : 0x" + strings.ToUpper(strconv.FormatInt(rpa.Key, 16)))
	logln("files   :", len(index))
	logln()
	names := make([]string, 0, len(index))
	for n := range index {
		names = append(names, n)
	}
	sort.Strings(names)
	const sample = 20
	shown := 0
	for _, n := range names {
		if shown >= sample {
			break
		}
		logln("  " + n + "  (" + formatN0(RpaFileSize(index[n])) + " bytes)")
		shown++
	}
	if len(index) > sample {
		logln("  ... and", len(index)-sample, "more")
	}
	return 0
}

func extractCommand(rpaPath, outputDir string) int {
	defer beginTranscript(outputDir)()
	if _, err := os.Stat(rpaPath); err != nil {
		errln("error: file not found:", rpaPath)
		return 1
	}
	rpa, err := OpenRpa(rpaPath)
	if err != nil {
		errln("error:", err)
		return 1
	}
	index, err := rpa.ReadIndex()
	if err != nil {
		errln("error:", err)
		return 1
	}
	logln("Extracting", len(index), "files from", filepath.Base(rpaPath), "...")
	err = rpa.ExtractAll(outputDir, index, func(count, total int, name string) {
		if count%50 == 0 || count == total {
			logln("  [" + strconv.Itoa(count) + "/" + strconv.Itoa(total) + "] " + name)
		}
	})
	if err != nil {
		errln("error:", err)
		return 1
	}
	logln("Done. Extracted", len(index), "files to", outputDir)
	return 0
}

// resolveGameDir uses a child folder named game when the selection is the
// project root (the folder that contains game/) rather than game/ itself.
func resolveGameDir(dir string) (string, bool) {
	dir = filepath.Clean(dir)
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return dir, false
	}
	if strings.EqualFold(filepath.Base(dir), "game") {
		return dir, false
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return dir, false
	}
	for _, e := range ents {
		if e.IsDir() && strings.EqualFold(e.Name(), "game") {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return dir, false
}

func infoCommand(gameDir string) int {
	st, err := os.Stat(gameDir)
	if err != nil || !st.IsDir() {
		errln("error: directory not found:", gameDir)
		return 1
	}
	if next, ok := resolveGameDir(gameDir); ok {
		logln("selected folder contains game; using", next)
		gameDir = next
	}
	scripts := collectScripts(gameDir)
	if len(scripts) == 0 {
		errln("error: no .rpyc files found under", gameDir, "(also checked inside .rpa)")
		return 1
	}
	classes := map[string]struct{}{}
	scanned, failed := 0, 0
	for _, s := range scripts {
		b, err := decompressRpyc(s.Data)
		if err != nil {
			failed++
			errln("  (skip)", s.Name+":", err)
			continue
		}
		for c := range ScanClasses(b) {
			classes[c] = struct{}{}
		}
		scanned++
	}
	buckets := map[string][]string{}
	for c := range classes {
		cat := classify(c)
		buckets[cat] = append(buckets[cat], c)
	}
	logln("game     :", gameDir)
	skip := ""
	if failed > 0 {
		skip = ", " + strconv.Itoa(failed) + " skipped"
	}
	logln("scanned  :", scanned, ".rpyc file(s)"+skip)
	logln("constructs:", len(classes), "distinct classes referenced")
	logln()
	for _, cat := range categoryOrder {
		set := buckets[cat]
		if len(set) == 0 {
			continue
		}
		sort.Strings(set)
		logln(cat + ":")
		for _, c := range set {
			logln("    " + c)
		}
		logln()
	}
	_, screens := buckets[catScreen]
	_, atl := buckets[catAtl]
	_, other := buckets[catOther]
	logln("verdict:")
	if !screens && !atl {
		logln("    Linear core only - strong early-conversion candidate.")
	} else {
		logln("    Linear story is convertible, but full fidelity needs:")
		if atl {
			logln("      - ATL / transforms (sprite animation)")
		}
		if screens {
			logln("      - screen language (menus / GUI - the 'looks identical' bar)")
		}
	}
	if other {
		logln("    NOTE: unrecognized classes present (see 'Other / review') - confirm before converting.")
	}
	return 0
}

func astCommand(rpycPath, labelName string) int {
	if _, err := os.Stat(rpycPath); err != nil {
		errln("error: file not found:", rpycPath)
		return 1
	}
	root, err := LoadAst(rpycPath)
	if err != nil {
		errln("error:", err)
		return 1
	}
	logln("top-level:", shape(root))
	stmts := FindStatementList(root)
	if stmts == nil {
		logln("(could not locate a statement list)")
		return 0
	}
	logln("statements:", len(stmts))
	topHist := map[string]int{}
	for _, s := range ToNodes(stmts) {
		topHist[s.Short()]++
	}
	logln("\n-- TOP-LEVEL class counts --")
	printHist(topHist)
	hist := map[string]int{}
	seen := map[any]struct{}{}
	countClasses(root, hist, seen, 0)
	logln("\n-- WHOLE-TREE class histogram --")
	printHist(hist)
	if labelName != "" {
		logln("\n-- locating label '" + labelName + "' --")
		topMatch := 0
		for _, s := range ToNodes(stmts) {
			if s.Short() == "Label" && s.LabelName() == labelName {
				topMatch++
				block := s.Block("block")
				var kinds []string
				for k := 0; k < len(block) && k < 8; k++ {
					kinds = append(kinds, block[k].Short())
				}
				logln("  found at TOP LEVEL; block has", len(block), "statements:", strings.Join(kinds, ", "))
			}
		}
		if topMatch == 0 {
			logln("  NOT found among top-level statements (it is nested somewhere).")
		}
	}
	return 0
}

func atlDumpCommand(rpycPath string) int {
	if _, err := os.Stat(rpycPath); err != nil {
		errln("error: file not found:", rpycPath)
		return 1
	}
	root, err := LoadAst(rpycPath)
	if err != nil {
		errln("error:", err)
		return 1
	}
	atlWalk(root, map[any]struct{}{}, 0)
	return 0
}

func atlWalk(o any, seen map[any]struct{}, depth int) {
	if o == nil || depth > 40 {
		return
	}
	if p := asPy(o); p != nil {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		isAtl := strings.Contains(p.ClassName, ".atl.")
		if isAtl {
			logln(strings.Repeat(" ", depth*2) + "<" + p.ClassName + ">")
			st := stateDict(p)
			if st != nil {
				for _, e := range st.Items() {
					logln(strings.Repeat(" ", depth*2+2) + toString(e.k) + " = " + renderVal(e.v, 3))
				}
			}
		}
		next := depth
		if isAtl {
			next++
		}
		atlWalk(p.State, seen, next)
		for _, a := range p.Args {
			atlWalk(a, seen, next)
		}
		return
	}
	if t, ok := o.(Tuple); ok {
		for _, it := range t {
			atlWalk(it, seen, depth)
		}
		return
	}
	if d := asDict(o); d != nil {
		for _, e := range d.Items() {
			atlWalk(e.v, seen, depth)
		}
		return
	}
	if l := asSeq(o); l != nil {
		for _, it := range l {
			atlWalk(it, seen, depth)
		}
	}
}

func renderVal(o any, depth int) string {
	if o == nil {
		return "None"
	}
	if s, ok := o.(string); ok {
		return `"` + s + `"`
	}
	if p := asPy(o); p != nil {
		if txt, ok := AsText(p); ok {
			return `expr("` + txt + `")`
		}
		return "<" + p.ClassName + ">"
	}
	if depth <= 0 {
		return shape(o)
	}
	if t, ok := o.(Tuple); ok {
		parts := make([]string, len(t))
		for i, it := range t {
			parts[i] = renderVal(it, depth-1)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
	if l := asSeq(o); l != nil {
		parts := make([]string, len(l))
		for i, it := range l {
			parts[i] = renderVal(it, depth-1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return toString(o)
}

func shape(o any) string {
	if o == nil {
		return "null"
	}
	if s, ok := o.(string); ok {
		if len(s) > 50 {
			s = s[:50] + "..."
		}
		return `"` + s + `"`
	}
	if p := asPy(o); p != nil {
		return "<" + p.ClassName + " state=" + shapeState(p.State) + ">"
	}
	if g, ok := o.(*GlobalRef); ok {
		return "global:" + g.Full()
	}
	if t, ok := o.(Tuple); ok {
		return "tuple[" + strconv.Itoa(len(t)) + "]"
	}
	if d := asDict(o); d != nil {
		return "dict{" + strconv.Itoa(d.Len()) + "}"
	}
	if l := asSeq(o); l != nil {
		return "list[" + strconv.Itoa(len(l)) + "]"
	}
	return fmt.Sprintf("%T=%v", o, o)
}

func shapeState(state any) string {
	if state == nil {
		return "null"
	}
	if t, ok := state.(Tuple); ok {
		extra := ""
		if len(t) > 1 {
			if d := asDict(t[1]); d != nil {
				extra = " attrs{" + strconv.Itoa(d.Len()) + "}"
			}
		}
		return "tuple[" + strconv.Itoa(len(t)) + "]" + extra
	}
	if d := asDict(state); d != nil {
		return "dict{" + strconv.Itoa(d.Len()) + "}"
	}
	return fmt.Sprintf("%T", state)
}

func packCommand(args []string) int {
	gameDir, outRpk := args[1], args[2]
	var err error
	maxW, maxH := 1920, 1080
	ffmpegPath := ""
	asciiText, useCache, clearCache, animatedGif := false, true, false, false
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--max":
			if i+1 >= len(args) {
				errln("error: --max needs WxH, for example 1920x1080")
				return 1
			}
			i++
			w, h, ok := parseMaxSize(args[i])
			if !ok {
				errln("error: --max must be WxH (1920x1080, 1280x720, 768x576, 720x480) or one edge in pixels")
				return 1
			}
			maxW, maxH = w, h
		case "--ffmpeg":
			if i+1 < len(args) {
				i++
				ffmpegPath = args[i]
			}
		case "--beta":
			if i+1 >= len(args) {
				errln("error: --beta needs a name, for example gif")
				return 1
			}
			i++
			switch strings.ToLower(args[i]) {
			case "gif":
				animatedGif = true
			default:
				errln("error: unknown beta feature:", args[i])
				return 1
			}
		case "--ascii-text":
			asciiText = true
		case "--no-cache":
			useCache = false
		case "--clear-cache":
			clearCache = true
		}
	}
	if maxW < 16 || maxH < 16 {
		errln("error: --max too small")
		return 1
	}
	st, err := os.Stat(gameDir)
	if err != nil || !st.IsDir() {
		errln("error: game dir not found:", gameDir)
		return 1
	}
	gameDir, switched := resolveGameDir(gameDir)
	outRpk, err = normalizeRpkPath(gameDir, outRpk)
	if err != nil {
		errln("error:", err)
		return 1
	}
	defer beginTranscript(outRpk)()
	ff, err := NewFfmpeg(ffmpegPath)
	if err != nil {
		errln("error:", err)
		return 1
	}

	if switched {
		logln("selected folder contains game; using", gameDir)
	}
	logln("packing", gameDir, "->", outRpk, " (max", strconv.Itoa(maxW)+"x"+strconv.Itoa(maxH), "ascii-text="+strconv.FormatBool(asciiText)+", animated-gif="+strconv.FormatBool(animatedGif)+", ffmpeg:", ff.Path+")")
	rc := Pack(gameDir, outRpk, ff, maxW, maxH, asciiText, useCache, clearCache, animatedGif)
	logln("PACK_DONE rc=" + strconv.Itoa(rc))
	return rc
}

func rpkCommand(path string) int {
	if _, err := os.Stat(path); err != nil {
		errln("error: file not found:", path)
		return 1
	}
	toc, err := ReadRpkToc(path)
	if err != nil {
		errln("error:", err)
		return 1
	}
	var total int64
	for _, e := range toc {
		total += e.Length
	}
	sort.Slice(toc, func(i, j int) bool { return toc[i].Length > toc[j].Length })
	logln(path+":", len(toc), "entries")
	for i := 0; i < len(toc) && i < 25; i++ {
		logf("  %12s  %s\n", formatN0(toc[i].Length), toc[i].Name)
	}
	if len(toc) > 25 {
		logln("  ... and", len(toc)-25, "more")
	}
	st, _ := os.Stat(path)
	fsz := int64(0)
	if st != nil {
		fsz = st.Size()
	}
	logln("total payload:", formatN0(total), "bytes; file:", formatN0(fsz), "bytes")
	return 0
}

func countClasses(o any, hist map[string]int, seen map[any]struct{}, depth int) {
	if o == nil || depth > 200 {
		return
	}
	if p := asPy(o); p != nil {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		hist[p.ClassName]++
		countClasses(p.State, hist, seen, depth+1)
		for _, it := range p.ListItems {
			countClasses(it, hist, seen, depth+1)
		}
		if p.DictItems != nil {
			for _, e := range p.DictItems.Items() {
				countClasses(e.v, hist, seen, depth+1)
			}
		}
		for _, a := range p.Args {
			countClasses(a, hist, seen, depth+1)
		}
		return
	}
	if t, ok := o.(Tuple); ok {
		for _, it := range t {
			countClasses(it, hist, seen, depth+1)
		}
		return
	}
	if d := asDict(o); d != nil {
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		for _, e := range d.Items() {
			countClasses(e.v, hist, seen, depth+1)
		}
		return
	}
	if l := asSeq(o); l != nil {
		for _, it := range l {
			countClasses(it, hist, seen, depth+1)
		}
	}
}

func scriptCommand(rpycPath string) int {
	if _, err := os.Stat(rpycPath); err != nil {
		errln("error: file not found:", rpycPath)
		return 1
	}
	stmts, err := LoadStatements(rpycPath)
	if err != nil {
		errln("error:", err)
		return 1
	}
	if stmts == nil {
		logln("(no statement list)")
		return 1
	}
	var sb strings.Builder
	for _, node := range ToNodes(stmts) {
		renderScript(node, 0, &sb)
	}
	logf("%s", sb.String())
	return 0
}

func renderScript(n AstNode, indent int, sb *strings.Builder) {
	pad := strings.Repeat(" ", indent*4)
	switch n.Short() {
	case "Label":
		sb.WriteString(pad)
		sb.WriteString("label ")
		sb.WriteString(n.LabelName())
		sb.WriteString(":\n")
		for _, c := range n.Block("block") {
			renderScript(c, indent+1, sb)
		}
	case "Say":
		who, whoOk := AsText(n.Raw("who"))
		what, _ := AsText(n.Raw("what"))
		sb.WriteString(pad)
		if whoOk {
			sb.WriteString(who + " ")
		}
		sb.WriteByte('"')
		sb.WriteString(what)
		sb.WriteString("\"\n")
	case "Jump":
		sb.WriteString(pad + "jump " + n.Text("target") + "\n")
	case "Call":
		sb.WriteString(pad + "call " + n.Text("label") + "\n")
	case "Return":
		sb.WriteString(pad + "return\n")
	case "Pass":
		sb.WriteString(pad + "pass\n")
	case "Scene":
		sb.WriteString(pad + "scene " + n.ImspecName() + "\n")
	case "Show":
		sb.WriteString(pad + "show " + n.ImspecName() + "\n")
	case "Hide":
		sb.WriteString(pad + "hide " + n.ImspecName() + "\n")
	case "With":
		sb.WriteString(pad + "with " + n.Text("expr") + "\n")
	case "Menu":
		sb.WriteString(pad + "menu:\n")
		for _, item := range asSeq(n.Raw("items")) {
			t, ok := item.(Tuple)
			if !ok || len(t) < 2 {
				continue
			}
			caption, _ := AsText(t[0])
			cond, _ := AsText(t[1])
			sb.WriteString(pad + `    "` + caption + `"`)
			if cond != "" && cond != "True" {
				sb.WriteString(" if " + cond)
			}
			sb.WriteString(":\n")
			if len(t) > 2 {
				for _, c := range ToNodes(t[2]) {
					renderScript(c, indent+2, sb)
				}
			}
		}
	case "If":
		first := true
		for _, entry := range asSeq(n.Raw("entries")) {
			t, ok := entry.(Tuple)
			if !ok || len(t) < 2 {
				continue
			}
			cond, _ := AsText(t[0])
			kw := "elif "
			if first {
				kw = "if "
				first = false
			}
			sb.WriteString(pad + kw + cond + ":\n")
			for _, c := range ToNodes(t[1]) {
				renderScript(c, indent+1, sb)
			}
		}
	case "Python", "Init":
		src := n.PyCodeSource()
		if src != "" {
			oneLine := !strings.Contains(src, "\n")
			sb.WriteString(pad)
			if oneLine {
				sb.WriteString("$ " + strings.TrimSpace(src))
			} else {
				sb.WriteString("python: ...")
			}
			sb.WriteByte('\n')
		} else {
			sb.WriteString(pad + "# <" + n.Short() + ">\n")
			for _, c := range n.Block("block") {
				renderScript(c, indent+1, sb)
			}
		}
	case "UserStatement":
		sb.WriteString(pad + n.Text("line") + "\n")
	default:
		sb.WriteString(pad + "# <" + n.Short() + ">\n")
	}
}

func compileCommand(path string, full bool, outRbc string) int {
	if outRbc != "" {
		defer beginTranscript(outRbc)()
	}
	var units [][]any
	st, err := os.Stat(path)
	if err == nil && st.IsDir() {
		if next, ok := resolveGameDir(path); ok {
			logln("selected folder contains game; using", next)
			path = next
		}
		var found int
		units, found, _ = loadScriptUnits(path)
		logln("compiling", len(units), "of", found, ".rpyc file(s) from", path)
	} else if err == nil {
		list, err := LoadStatements(path)
		if err != nil {
			errln("error:", err)
			return 1
		}
		if list == nil {
			errln("error: no statement list")
			return 1
		}
		units = append(units, []any(list))
	} else {
		errln("error: not found:", path)
		return 1
	}

	prog := CompileUnits(units, false)
	if full {
		for a, ins := range prog.Code {
			logf("%5d: %s\n", a, fmtInstr(prog, ins))
		}
	}
	ops := map[string]int{}
	for _, ins := range prog.Code {
		ops[ins.Op.String()]++
	}
	logln("\n== IR summary ==")
	logln("instructions :", len(prog.Code))
	logln("strings      :", len(prog.Strings))
	logln("labels       :", len(prog.Labels))
	logln("atl programs :", len(prog.Atls))
	logln("imagemaps    :", len(prog.ImageMaps))
	for _, imd := range prog.ImageMaps {
		kind := imd.Kind
		if kind == "" {
			kind = "(simple)"
		}
		logln("    [" + kind + "] ground=" + imd.Ground + " idle=" + imd.Idle + " hover=" + imd.Hover +
			" selIdle=" + imd.SelectedIdle + " selHover=" + imd.SelectedHover + "  hotspots=" + strconv.Itoa(len(imd.Hotspots)))
		var names strings.Builder
		for _, hh := range imd.Hotspots {
			names.WriteString(hh.Name)
			names.WriteByte(' ')
		}
		logln("        names:", strings.TrimSpace(names.String()))
	}
	logln("overlays     :", len(prog.Overlays))
	logln("op counts    :")
	printHist(ops)

	unsupCount := map[string]int{}
	for _, u := range prog.Unsupported {
		unsupCount[u]++
	}
	type kv struct {
		k string
		v int
	}
	var unsupList []kv
	for k, v := range unsupCount {
		unsupList = append(unsupList, kv{k, v})
	}
	sort.Slice(unsupList, func(i, j int) bool { return unsupList[i].v > unsupList[j].v })
	nUnsup := len(prog.Unsupported)
	if nUnsup == 0 {
		logln("\nunsupported nodes : none")
	} else {
		logln("\nunsupported nodes :", nUnsup)
	}
	for _, g := range unsupList {
		logf("    %5d  %s\n", g.v, g.k)
	}
	noteCount := map[string]int{}
	for _, nt := range prog.Notes {
		noteCount[nt]++
	}
	if len(prog.Notes) == 0 {
		logln("fidelity notes    : none")
	} else {
		logln("fidelity notes    :", len(prog.Notes))
	}
	printHist(noteCount)

	logln("\n== GUI manifest (game.gui) ==")
	guiTxt := BuildGuiManifest(prog)
	if guiTxt == "" {
		logln("    (no GUI settings found)")
	} else {
		logf("%s", guiTxt)
	}

	unresSet := map[string]struct{}{}
	for _, u := range prog.Unresolved {
		unresSet[u] = struct{}{}
	}
	var unres []string
	for u := range unresSet {
		unres = append(unres, u)
	}
	sort.Strings(unres)
	if len(unres) == 0 {
		logln("unresolved jump/call targets : none")
	} else {
		logln("unresolved jump/call targets :", len(unres))
	}
	for k := 0; k < len(unres) && k < 20; k++ {
		logln("    " + unres[k])
	}
	ok := len(prog.Unsupported) == 0 && len(unres) == 0
	if ok {
		logln("\nverdict: fully lowered to linear-core IR.")
	} else {
		logln("\nverdict: NOT fully convertible yet (see unsupported / unresolved above).")
	}

	if outRbc != "" {
		bytes := WriteBytecode(prog)
		if err := os.WriteFile(outRbc, bytes, 0644); err != nil {
			errln("error:", err)
			return 1
		}
		dec, err := ReadBytecode(bytes)
		rt := err == nil && len(dec.Code) == len(prog.Code) && len(dec.Strings) == len(prog.Strings) && len(dec.Labels) == len(prog.Labels)
		expectEntry := uint32(0)
		if st, ok := prog.Labels["start"]; ok {
			expectEntry = uint32(st)
		}
		if dec != nil && dec.EntryAddr != expectEntry {
			rt = false
		}
		for k := 0; rt && k < len(prog.Code); k++ {
			x, y := prog.Code[k], dec.Code[k]
			if x.Op != y.Op || x.A != y.A || x.B != y.B || x.C != y.C {
				rt = false
			}
		}
		for k := 0; rt && k < len(prog.Strings); k++ {
			if prog.Strings[k] != dec.Strings[k] {
				rt = false
			}
		}
		logln("\nwrote", outRbc, "("+formatN0(int64(len(bytes)))+" bytes)")
		if rt {
			logln("round-trip verify: PASS (decoded == compiled)")
		} else {
			logln("round-trip verify: FAIL")
		}
		entry := uint32(0)
		if dec != nil {
			entry = dec.EntryAddr
		}
		logln("entry 'start' @", entry)
	}
	return 0
}

func prev(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "")
	if len(s) > 50 {
		return s[:50] + "..."
	}
	return s
}

func fmtInstr(p *IrProgram, i Instr) string {
	switch i.Op {
	case IrLabel:
		return "label " + p.Str(i.A) + ":"
	case IrSay:
		who := ""
		if i.A >= 0 {
			who = p.Str(i.A) + " "
		}
		return "SAY " + who + `"` + prev(p.Str(i.B)) + `"`
	case IrScene:
		return "SCENE " + p.Str(i.A)
	case IrShow:
		return "SHOW " + p.Str(i.A)
	case IrHide:
		return "HIDE " + p.Str(i.A)
	case IrWith:
		return "WITH " + p.Str(i.A)
	case IrJump:
		return "JUMP -> " + strconv.Itoa(i.A)
	case IrCall:
		return "CALL -> " + strconv.Itoa(i.A)
	case IrReturn:
		return "RETURN"
	case IrMenuStart:
		return "MENU (" + strconv.Itoa(i.A) + " choices)"
	case IrChoice:
		extra := ""
		if i.B >= 0 {
			extra = " if " + fmtExpr(p, i.B)
		}
		return `  CHOICE "` + prev(p.Str(i.A)) + `"` + extra + " -> " + strconv.Itoa(i.C)
	case IrMenuEnd:
		return "MENUEND"
	case IrIfFalseGoto:
		return "IF NOT (" + fmtExpr(p, i.A) + ") GOTO " + strconv.Itoa(i.C)
	case IrPyExec:
		return "PY " + prev(p.Str(i.A))
	case IrDefault:
		rhs := prev(p.Str(i.B))
		if i.C >= 0 {
			rhs = fmtExpr(p, i.C)
		}
		return "DEFAULT " + p.Str(i.A) + " = " + rhs
	case IrAssign:
		return "ASSIGN " + p.Str(i.A) + " " + assignOpStr(i.B) + " " + fmtExpr(p, i.C)
	case IrImage:
		return "IMAGE " + p.Str(i.A) + " = " + prev(p.Str(i.B))
	case IrUser:
		return "USER " + prev(p.Str(i.A))
	default:
		return i.Op.String()
	}
}

func assignOpStr(kind int) string {
	switch kind {
	case 1:
		return "+="
	case 2:
		return "-="
	case 3:
		return "*="
	case 4:
		return "/="
	case 5:
		return "%="
	default:
		return "="
	}
}

func fmtExpr(p *IrProgram, idx int) string {
	if idx < 0 || idx >= len(p.Exprs) {
		return "?expr#" + strconv.Itoa(idx)
	}
	var sb strings.Builder
	for _, e := range p.Exprs[idx].Ops {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		switch e.Op {
		case ExprPushInt:
			sb.WriteString(strconv.Itoa(e.Arg))
		case ExprPushBool:
			if e.Arg != 0 {
				sb.WriteString("True")
			} else {
				sb.WriteString("False")
			}
		case ExprPushNone:
			sb.WriteString("None")
		case ExprPushFloat:
			sb.WriteString(p.Str(e.Arg))
		case ExprPushStr:
			sb.WriteByte('"')
			sb.WriteString(p.Str(e.Arg))
			sb.WriteByte('"')
		case ExprLoadVar:
			sb.WriteString(p.Str(e.Arg))
		default:
			sb.WriteString(strings.ToLower(exprOpName(e.Op)))
		}
	}
	return sb.String()
}

func exprOpName(op ExprOp) string {
	names := []string{
		"PushInt", "PushBool", "PushNone", "PushFloat", "PushStr", "LoadVar",
		"Neg", "Not", "Add", "Sub", "Mul", "Div", "Mod",
		"Eq", "Ne", "Lt", "Le", "Gt", "Ge", "And", "Or", "FloorDiv", "Max", "Min",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

func printHist(d map[string]int) {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		logf("  %5d  %s\n", d[k], k)
	}
}

func formatN0(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

const (
	catCore   = "Supported (linear core)"
	catPy     = "Supported (python / data)"
	catUser   = "Partial (user statements - depends on verb)"
	catAtl    = "Deferred (ATL / transforms - animation)"
	catScreen = "Not yet supported (screen language / display)"
	catTl     = "Deferred (translations)"
	catOther  = "Other / review"
)

var categoryOrder = []string{catCore, catPy, catUser, catAtl, catScreen, catTl, catOther}

var coreNodes = map[string]struct{}{
	"renpy.ast.Say": {}, "renpy.ast.Scene": {}, "renpy.ast.Show": {}, "renpy.ast.Hide": {}, "renpy.ast.With": {},
	"renpy.ast.Jump": {}, "renpy.ast.Call": {}, "renpy.ast.Return": {}, "renpy.ast.Label": {}, "renpy.ast.Menu": {},
	"renpy.ast.Pass": {}, "renpy.ast.Init": {}, "renpy.ast.If": {}, "renpy.ast.While": {},
	"renpy.ast.Default": {}, "renpy.ast.Define": {}, "renpy.ast.Image": {}, "renpy.ast.EarlyPython": {},
	"renpy.ast.Python": {}, "renpy.ast.PyCode": {}, "renpy.ast.PyExpr": {},
}

func classify(cls string) string {
	if _, ok := coreNodes[cls]; ok {
		return catCore
	}
	if strings.HasSuffix(cls, ".PyExpr") || strings.HasSuffix(cls, ".PyCode") {
		return catCore
	}
	if cls == "renpy.ast.UserStatement" {
		return catUser
	}
	if strings.HasPrefix(cls, "renpy.atl.") || cls == "renpy.ast.Transform" {
		return catAtl
	}
	if strings.Contains(cls, "Translate") {
		return catTl
	}
	if strings.HasPrefix(cls, "renpy.sl2.") || strings.HasPrefix(cls, "renpy.display.") ||
		strings.HasPrefix(cls, "renpy.ui.") || strings.HasPrefix(cls, "renpy.text.") ||
		cls == "renpy.ast.Screen" || cls == "renpy.ast.Style" {
		return catScreen
	}
	if strings.HasPrefix(cls, "renpy.python.") || strings.HasPrefix(cls, "renpy.object.") ||
		strings.HasPrefix(cls, "renpy.revertable.") || strings.HasPrefix(cls, "renpy.parameter.") ||
		strings.HasPrefix(cls, "builtins.") || strings.HasPrefix(cls, "collections.") ||
		cls == "renpy.ast.ArgumentInfo" || cls == "renpy.ast.ParameterInfo" {
		return catPy
	}
	return catOther
}
