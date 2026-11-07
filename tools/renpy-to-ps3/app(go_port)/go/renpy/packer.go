package renpy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	imageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".bmp": true, ".gif": true, ".webp": true, ".tga": true}
	audioExt = map[string]bool{".mp3": true, ".ogg": true, ".oga": true, ".wav": true, ".opus": true, ".m4a": true, ".aac": true, ".flac": true}
	videoExt = map[string]bool{".webm": true, ".mpg": true, ".mpeg": true, ".avi": true, ".mp4": true, ".ogv": true, ".mkv": true, ".mov": true}
	fontExt  = map[string]bool{".ttf": true, ".otf": true}
	skipExt  = map[string]bool{".rpa": true, ".rpyc": true, ".rpy": true, ".rpyb": true, ".rpymc": true, ".py": true, ".pyc": true, ".txt": true, ".json": true, ".ico": true, ".md": true}
)

const cacheVersion = 2

// putAsset stores data under name, matching C# OrdinalIgnoreCase: last write
// wins, but the first-seen casing of the key is kept.
func putAsset(m map[string][]byte, name string, data []byte) {
	for k := range m {
		if strings.EqualFold(k, name) {
			m[k] = data
			return
		}
	}
	m[name] = data
}

func Pack(gameDir, outRpk string, ff *Ffmpeg, maxW, maxH int, asciiText, useCache, clearCache, animatedGif bool) int {
	staging, err := os.MkdirTemp("", "rpk_stage_")
	if err != nil {
		errln("error:", err)
		return 1
	}
	defer os.RemoveAll(staging)

	cacheDir := filepath.Join(baseDir(), "asset-cache")
	ffFp := ffmpegFingerprint(ff)
	if useCache {
		if clearCache {
			_ = os.RemoveAll(cacheDir)
		}
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			logln("  (cache disabled:", err.Error()+")")
			useCache = false
		}
	}

	units, scriptsFound, scriptsInRpa := loadScriptUnits(gameDir)
	logln("scripts:", len(units), "compiled of", scriptsFound, ".rpyc,", scriptsInRpa, "read from .rpa")
	prog := CompileUnits(units, asciiText)
	gui := BuildGuiManifest(prog)
	rbc := WriteBytecode(prog)

	nativeW := parseGuiInt(gui, "native_w")
	nativeH := parseGuiInt(gui, "native_h")
	uniform := nativeW > 0 && nativeH > 0
	factor := 1.0
	if uniform {
		factor = fitFactor(nativeW, nativeH, maxW, maxH)
	}
	if uniform && factor < 1.0 {
		gui += "asset_scale=" + strconv.FormatFloat(factor, 'f', 6, 64) + "\n"
	}
	maxLabel := strconv.Itoa(maxW) + "x" + strconv.Itoa(maxH)
	if uniform {
		logln("scaling: uniform x" + strconv.FormatFloat(factor, 'f', 3, 64) + " (native " + strconv.Itoa(nativeW) + "x" + strconv.Itoa(nativeH) + ", max " + maxLabel + ")")
	} else {
		logln("scaling: per-image cap " + maxLabel + " (native resolution unknown)")
	}
	sceneKeys := sceneImageKeys(prog)

	assets := map[string][]byte{}
	ents, _ := os.ReadDir(gameDir)
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".rpa") {
			continue
		}
		arc, err := OpenRpa(filepath.Join(gameDir, e.Name()))
		if err != nil {
			errln("  (skip)", e.Name()+":", err)
			continue
		}
		idx, err := arc.ReadIndex()
		if err != nil {
			errln("  (skip)", e.Name()+":", err)
			continue
		}
		for name, segs := range idx {
			data, err := arc.ReadFile(segs)
			if err != nil {
				continue
			}
			putAsset(assets, name, data)
		}
	}
	_ = filepath.WalkDir(gameDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if skipExt[ext] {
			return nil
		}
		if imageExt[ext] || audioExt[ext] || videoExt[ext] || fontExt[ext] {
			data, err := os.ReadFile(p)
			if err == nil {
				putAsset(assets, relativePath(gameDir, p), data)
			}
		}
		return nil
	})

	type job struct {
		name string
		data []byte
	}
	work := make([]job, 0, len(assets))
	for n, d := range assets {
		work = append(work, job{n, d})
	}

	type result struct {
		outName  string
		outBytes []byte
		fail     string
		kind     int
		via      int
		srcLen   int
	}

	var (
		packed                                              []RpkEntry
		failed                                              []string
		images, audio, video, fonts, skipped, gifs, blurred int
		cacheHits, encoded, passed                          int
		srcBytes, dstBytes                                  int64
		done                                                int32
		jobID                                               int32
		mu                                                  sync.Mutex
	)
	total := len(work)
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	ch := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for kv := range ch {
				ext := strings.ToLower(filepath.Ext(kv.name))
				id := atomic.AddInt32(&jobID, 1)
				var outName string
				var outBytes []byte
				var fail string
				kind, via := -1, -1
				didBlur := false
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							fail = kv.name + " : " + fmt.Sprint(rec)
						}
					}()
					if imageExt[ext] {
						kind = 0
						// JPEG stays JPEG. Other images are PNG bytes under the script's
						// filename, so ajax-loader.gif is still found. The Animated GIF
						// beta keeps a real GIF instead of one PNG frame.
						outExt := ".png"
						if ext == ".jpg" || ext == ".jpeg" {
							outExt = ".jpg"
						} else if ext == ".gif" && animatedGif {
							outExt = ".gif"
						}
						var errStr string
						blurSides := outExt != ".gif" && wantBlurSides(kv.data, ext, kv.name, sceneKeys, nativeW, nativeH, maxW, maxH)
						ok := convertAsset(ff, kv.data, ext, outExt, staging, int(id), uniform, factor, maxW, maxH, cacheDir, useCache, blurSides, ffFp, &outBytes, &errStr, &via)
						if ok {
							outName = kv.name
							didBlur = blurSides
						} else {
							fail = kv.name + " : " + errStr
						}
					} else if audioExt[ext] {
						kind = 1
						var errStr string
						ok := convertAsset(ff, kv.data, ext, ".ogg", staging, int(id), uniform, factor, maxW, maxH, cacheDir, useCache, false, ffFp, &outBytes, &errStr, &via)
						if ok {
							outName = changeExt(kv.name, ".ogg")
						} else {
							fail = kv.name + " : " + errStr
						}
					} else if videoExt[ext] {
						kind = 2
						var errStr string
						ok := convertAsset(ff, kv.data, ext, ".mp4", staging, int(id), uniform, factor, maxW, maxH, cacheDir, useCache, false, ffFp, &outBytes, &errStr, &via)
						if ok {
							outName = changeExt(kv.name, ".mp4")
						} else {
							fail = kv.name + " : " + errStr
						}
					} else if fontExt[ext] {
						kind = 3
						outName = kv.name
						outBytes = kv.data
					} else {
						kind = 4
					}
				}()
				mu.Lock()
				if didBlur {
					blurred++
				}
				srcBytes += int64(len(kv.data))
				if outName != "" {
					packed = append(packed, RpkEntry{outName, outBytes})
					dstBytes += int64(len(outBytes))
					switch kind {
					case 0:
						images++
						if ext == ".gif" {
							gifs++
						}
					case 1:
						audio++
					case 2:
						video++
					case 3:
						fonts++
					}
					switch via {
					case 0:
						encoded++
					case 1:
						cacheHits++
					case 2:
						passed++
					}
				} else if fail != "" {
					failed = append(failed, fail)
				} else if kind == 4 {
					skipped++
				}
				n := int(atomic.AddInt32(&done, 1))
				if n%25 == 0 || n == total {
					logln("  [" + strconv.Itoa(n) + "/" + strconv.Itoa(total) + "] converting assets...")
				}
				mu.Unlock()
				_ = result{}
			}
		}()
	}
	for _, j := range work {
		ch <- j
	}
	close(ch)
	wg.Wait()

	sort.Slice(packed, func(i, j int) bool { return packed[i].Name < packed[j].Name })

	di := filepath.Base(strings.TrimRight(filepath.Clean(gameDir), string(os.PathSeparator)))
	parent := filepath.Base(filepath.Dir(filepath.Clean(gameDir)))
	title := parent
	if title == "." || title == "/" || title == "" {
		title = di
	}
	man := strings.Builder{}
	man.WriteString("title: " + title + "\n")
	entry := "(no start label)"
	if _, ok := prog.Labels["start"]; ok {
		entry = "start"
	}
	man.WriteString("entry: " + entry + "\n")
	man.WriteString("rpk_version: " + strconv.FormatUint(uint64(RpkVersion), 10) + "\n")
	man.WriteString("images: " + strconv.Itoa(images) + "\n")
	man.WriteString("audio: " + strconv.Itoa(audio) + "\n")
	man.WriteString("video: " + strconv.Itoa(video) + "\n")
	man.WriteString("fonts: " + strconv.Itoa(fonts) + "\n")

	entries := []RpkEntry{
		{"manifest", []byte(man.String())},
		{"game.gui", []byte(gui)},
		{"game.rbc", rbc},
	}
	for _, a := range packed {
		entries = append(entries, RpkEntry{"assets/" + a.Name, a.Data})
	}
	if err := WriteRpk(outRpk, entries); err != nil {
		errln("error:", err)
		return 1
	}

	st, _ := os.Stat(outRpk)
	unsup := uniqueCount(prog.Unsupported)
	unres := uniqueCount(prog.Unresolved)
	logln("")
	logln("== pack summary ==")
	logln("images:", images, " audio:", audio, " video:", video, " fonts:", fonts, " skipped:", skipped)
	if gifs > 0 && animatedGif {
		logln("animated gif:", gifs, "kept as GIF (beta). The player loops them only with the animated-GIF decoder.")
	} else if gifs > 0 {
		logln("animated gif:", gifs, "stored as one PNG frame (the PS3 player shows a still PNG or JPEG, not a GIF or Motion JPEG)")
	}
	if blurred > 0 {
		logln("4:3 scene images:", blurred, "centered on "+maxLabel+" with a blurred background on the left and right")
	}
	if useCache {
		logln("asset cache:", cacheHits, "reused,", passed, "passthrough,", encoded, "encoded (dir:", cacheDir+")")
	} else {
		logln("asset cache: DISABLED (--no-cache) -- everything re-encoded fresh")
	}
	logln("bytecode:", len(prog.Code), "instrs ("+strconv.Itoa(len(rbc))+" bytes); labels:", len(prog.Labels))
	logln("unsupported nodes:", unsup, " unresolved targets:", unres)
	printNotes(prog)
	pct := int64(0)
	if srcBytes > 0 {
		pct = 100 * dstBytes / srcBytes
	}
	logln("asset bytes:", srcBytes, "->", dstBytes, " ("+strconv.FormatInt(pct, 10)+"% of source)")
	logln("failed conversions:", len(failed))
	sort.Strings(failed)
	for k := 0; k < len(failed) && k < 15; k++ {
		logln("    " + failed[k])
	}
	rpkSize := int64(0)
	if st != nil {
		rpkSize = st.Size()
	}
	logln("")
	logln("wrote", outRpk, "("+strconv.FormatInt(rpkSize, 10)+" bytes,", len(entries), "entries)")
	if len(failed) == 0 {
		return 0
	}
	return 2
}

func printNotes(prog *IrProgram) {
	seen := map[string]struct{}{}
	var notes []string
	for _, n := range prog.Notes {
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			notes = append(notes, n)
		}
	}
	sort.Strings(notes)
	if len(notes) == 0 {
		logln("fidelity notes: none")
	} else {
		logln("fidelity notes:", len(notes))
	}
	for k := 0; k < len(notes) && k < 15; k++ {
		logln("    " + notes[k])
	}
}

func convertAsset(ff *Ffmpeg, src []byte, inExt, outExt, staging string, id int, uniform bool, factor float64, maxW, maxH int, cacheDir string, useCache, blurSides bool, ffFp string, outBytes *[]byte, errStr *string, via *int) bool {
	*outBytes = nil
	*errStr = ""
	*via = 0

	var cachePath string
	if useCache {
		key := cacheKey(src, inExt, outExt, uniform, factor, maxW, maxH, blurSides, ffFp)
		cachePath = filepath.Join(cacheDir, key+outExt)
		if b, err := os.ReadFile(cachePath); err == nil {
			*outBytes = b
			*via = 1
			return true
		}
	}
	if useCache && !blurSides && canPassThrough(src, inExt, outExt, uniform, factor, maxW, maxH) {
		*outBytes = src
		*via = 2
		storeCache(cachePath, *outBytes)
		return true
	}

	inp := filepath.Join(staging, "in"+strconv.Itoa(id)+inExt)
	outp := filepath.Join(staging, "out"+strconv.Itoa(id)+outExt)
	if err := os.WriteFile(inp, src, 0644); err != nil {
		*errStr = err.Error()
		return false
	}
	var ok bool
	var e string
	if outExt == ".ogg" {
		ok, e = ff.Audio(inp, outp)
	} else if outExt == ".gif" {
		if uniform {
			ok, e = ff.GifScaled(inp, outp, factor)
		} else {
			ok, e = ff.Gif(inp, outp, maxW, maxH)
		}
	} else if outExt == ".mp4" {
		if uniform {
			ok, e = ff.VideoScaled(inp, outp, factor)
		} else {
			ok, e = ff.Video(inp, outp, maxW, maxH)
		}
	} else if blurSides {
		ok, e = ff.ImageBlurSides(inp, outp, maxW, maxH)
	} else if uniform {
		ok, e = ff.ImageScaled(inp, outp, factor)
	} else {
		ok, e = ff.Image(inp, outp, maxW, maxH)
	}
	if ok {
		if b, err := os.ReadFile(outp); err == nil {
			*outBytes = b
		}
	} else if e == "" {
		*errStr = "no output produced"
	} else {
		if nl := strings.IndexByte(e, '\n'); nl >= 0 {
			e = e[:nl]
		}
		*errStr = e
	}
	_ = os.Remove(inp)
	_ = os.Remove(outp)
	if *outBytes != nil {
		*via = 0
		storeCache(cachePath, *outBytes)
		return true
	}
	return false
}

func cacheKey(src []byte, inExt, outExt string, uniform bool, factor float64, maxW, maxH int, blurSides bool, ffFp string) string {
	u := "P"
	if uniform {
		u = "U"
	}
	blur := "0"
	if blurSides {
		blur = "1"
	}
	hdr := "v" + strconv.Itoa(cacheVersion) + "|" + ffFp + "|" + strings.ToLower(inExt) + "|" + outExt + "|" + u + "|" +
		strconv.FormatFloat(factor, 'f', 6, 64) + "|" + strconv.Itoa(maxW) + "x" + strconv.Itoa(maxH) + "|blur=" + blur + "|"
	h := sha256.New()
	h.Write([]byte(hdr))
	h.Write(src)
	return hex.EncodeToString(h.Sum(nil))
}

func storeCache(path string, data []byte) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	if _, err := os.Stat(path); err == nil {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, path)
}

func canPassThrough(src []byte, inExt, outExt string, uniform bool, factor float64, maxW, maxH int) bool {
	inExt = strings.ToLower(inExt)
	var w, h int
	ok := false
	if inExt == ".gif" && outExt == ".gif" {
		if uniform {
			return factor >= 1.0
		}
		w, h, ok = gifLogicalSize(src)
	} else if inExt == ".png" && outExt == ".png" {
		ok = pngPlain(src, &w, &h)
	} else if (inExt == ".jpg" || inExt == ".jpeg") && outExt == ".jpg" {
		ok = jpgBaseline(src, &w, &h)
	} else {
		return false
	}
	if !ok {
		return false
	}
	if uniform {
		return factor >= 1.0
	}
	return w > 0 && h > 0 && w <= maxW && h <= maxH
}

// parseMaxSize reads a screen cap. "1920x1080", "1280×720" and "768 x 576"
// are width and height. A single number is that many pixels on both edges.
func parseMaxSize(s string) (w, h int, ok bool) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "×", "x")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ToLower(s)
	if i := strings.IndexByte(s, 'x'); i > 0 {
		var errW, errH error
		w, errW = strconv.Atoi(s[:i])
		h, errH = strconv.Atoi(s[i+1:])
		if errW != nil || errH != nil || w < 16 || h < 16 {
			return 0, 0, false
		}
		return w, h, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 16 {
		return 0, 0, false
	}
	return n, n, true
}

func isFourThree(w, h int) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	r := float64(w) / float64(h)
	return r >= 1.30 && r <= 1.37
}

func isSixteenNine(w, h int) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	r := float64(w) / float64(h)
	return r >= 1.70 && r <= 1.85
}

func quotedFile(s string) string {
	i := strings.IndexAny(s, `"'`)
	if i < 0 {
		return ""
	}
	q := s[i]
	rest := s[i+1:]
	j := strings.IndexByte(rest, q)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func addSceneFile(keys map[string]bool, file string) {
	file = strings.ReplaceAll(file, "\\", "/")
	file = strings.TrimPrefix(file, "/")
	low := strings.ToLower(file)
	if low == "" {
		return
	}
	keys[low] = true
	keys[strings.ToLower(filepath.Base(low))] = true
}

// sceneImageKeys is the set of files used as `scene` backgrounds.
func sceneImageKeys(prog *IrProgram) map[string]bool {
	keys := map[string]bool{}
	if prog == nil {
		return keys
	}
	files := map[string]string{}
	scan := func(code []Instr) {
		for _, in := range code {
			if in.Op != IrImage {
				continue
			}
			body := strings.TrimSpace(prog.Str(in.B))
			if strings.HasPrefix(strings.ToLower(body), "solid") {
				continue
			}
			file := quotedFile(body)
			if file == "" {
				continue
			}
			files[prog.Str(in.A)] = file
		}
	}
	scan(prog.Code)
	scan(prog.InitCode)
	note := func(code []Instr) {
		for _, in := range code {
			if in.Op != IrScene {
				continue
			}
			name := prog.Str(in.A)
			if f, ok := files[name]; ok {
				addSceneFile(keys, f)
				continue
			}
			if q := quotedFile(name); q != "" {
				addSceneFile(keys, q)
				continue
			}
			if strings.Contains(name, ".") {
				addSceneFile(keys, name)
			}
		}
	}
	note(prog.Code)
	note(prog.InitCode)
	return keys
}

func assetIsScene(keys map[string]bool, assetName string) bool {
	if len(keys) == 0 {
		return false
	}
	n := strings.ToLower(strings.ReplaceAll(assetName, "\\", "/"))
	n = strings.TrimPrefix(n, "/")
	if keys[n] || keys[strings.ToLower(filepath.Base(n))] {
		return true
	}
	for k := range keys {
		if k != "" && strings.HasSuffix(n, "/"+k) {
			return true
		}
	}
	return false
}

func imagePixelSize(b []byte, ext string) (w, h int, ok bool) {
	switch strings.ToLower(ext) {
	case ".png":
		if len(b) < 24 || string(b[1:4]) != "PNG" || string(b[12:16]) != "IHDR" {
			return 0, 0, false
		}
		w = int(b[16])<<24 | int(b[17])<<16 | int(b[18])<<8 | int(b[19])
		h = int(b[20])<<24 | int(b[21])<<16 | int(b[22])<<8 | int(b[23])
		return w, h, w > 0 && h > 0
	case ".jpg", ".jpeg":
		return jpegPixelSize(b)
	case ".gif":
		return gifLogicalSize(b)
	default:
		return 0, 0, false
	}
}

func jpegPixelSize(b []byte) (w, h int, ok bool) {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 0, 0, false
	}
	p := 2
	for p+4 <= len(b) {
		if b[p] != 0xFF {
			p++
			continue
		}
		marker := b[p+1]
		if marker == 0xD8 || marker == 0xD9 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			p += 2
			continue
		}
		length := int(b[p+2])<<8 | int(b[p+3])
		if length < 2 || p+2+length > len(b) {
			return 0, 0, false
		}
		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			if length < 7 {
				return 0, 0, false
			}
			h = int(b[p+5])<<8 | int(b[p+6])
			w = int(b[p+7])<<8 | int(b[p+8])
			return w, h, w > 0 && h > 0
		}
		p += 2 + length
	}
	return 0, 0, false
}

// wantBlurSides is true for a 4:3 scene image when the pack target is 16:9
// and the game itself is 4:3. The sharp picture stays in the center. Sprites
// and other files are left alone.
func wantBlurSides(src []byte, inExt, assetName string, sceneKeys map[string]bool, nativeW, nativeH, maxW, maxH int) bool {
	if !isSixteenNine(maxW, maxH) || !isFourThree(nativeW, nativeH) {
		return false
	}
	if !assetIsScene(sceneKeys, assetName) {
		return false
	}
	w, h, ok := imagePixelSize(src, inExt)
	return ok && isFourThree(w, h)
}

// fitFactor is how far a native WxH game must shrink to sit inside maxW x maxH.
// 1 means it already fits. The same factor is applied to every asset.
func fitFactor(nativeW, nativeH, maxW, maxH int) float64 {
	if nativeW <= 0 || nativeH <= 0 || maxW <= 0 || maxH <= 0 {
		return 1
	}
	fw := float64(maxW) / float64(nativeW)
	fh := float64(maxH) / float64(nativeH)
	factor := fw
	if fh < fw {
		factor = fh
	}
	if factor > 1 {
		return 1
	}
	return factor
}

func gifLogicalSize(b []byte) (w, h int, ok bool) {
	if len(b) < 10 || string(b[:3]) != "GIF" {
		return 0, 0, false
	}
	w = int(b[6]) | int(b[7])<<8
	h = int(b[8]) | int(b[9])<<8
	return w, h, w > 0 && h > 0
}

func pngPlain(b []byte, w, h *int) bool {
	*w, *h = 0, 0
	if len(b) < 33 {
		return false
	}
	sig := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	for i := 0; i < 8; i++ {
		if b[i] != sig[i] {
			return false
		}
	}
	if !(b[12] == 'I' && b[13] == 'H' && b[14] == 'D' && b[15] == 'R') {
		return false
	}
	*w = int(b[16])<<24 | int(b[17])<<16 | int(b[18])<<8 | int(b[19])
	*h = int(b[20])<<24 | int(b[21])<<16 | int(b[22])<<8 | int(b[23])
	bitDepth, colorType, interlace := b[24], b[25], b[28]
	if bitDepth != 8 {
		return false
	}
	if colorType != 2 && colorType != 6 {
		return false
	}
	if interlace != 0 {
		return false
	}
	return *w > 0 && *h > 0
}

func jpgBaseline(b []byte, w, h *int) bool {
	*w, *h = 0, 0
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return false
	}
	p := 2
	for p+2 <= len(b) {
		if b[p] != 0xFF {
			return false
		}
		marker := b[p+1]
		p += 2
		if marker == 0xD9 {
			return false
		}
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if p+2 > len(b) {
			return false
		}
		length := int(b[p])<<8 | int(b[p+1])
		if length < 2 || p+length > len(b) {
			return false
		}
		if marker == 0xC0 {
			if p+8 > len(b) {
				return false
			}
			prec := b[p+2]
			*h = int(b[p+3])<<8 | int(b[p+4])
			*w = int(b[p+5])<<8 | int(b[p+6])
			comps := b[p+7]
			if prec != 8 {
				return false
			}
			if comps != 1 && comps != 3 {
				return false
			}
			return *w > 0 && *h > 0
		}
		if marker == 0xC1 || marker == 0xC2 || marker == 0xC3 || (marker >= 0xC5 && marker <= 0xCF && marker != 0xC8) || marker == 0xDA {
			return false
		}
		p += length
	}
	return false
}

func parseGuiInt(gui, key string) int {
	prefix := key + "="
	for _, line := range strings.Split(gui, "\n") {
		if strings.HasPrefix(line, prefix) {
			v, err := strconv.Atoi(strings.TrimSpace(line[len(prefix):]))
			if err == nil {
				return v
			}
		}
	}
	return 0
}

func relativePath(baseDir, full string) string {
	b, _ := filepath.Abs(baseDir)
	f, _ := filepath.Abs(full)
	rel, err := filepath.Rel(b, f)
	if err != nil {
		return filepath.ToSlash(f)
	}
	return filepath.ToSlash(rel)
}

func changeExt(name, ext string) string {
	slash := strings.LastIndex(name, "/")
	dot := strings.LastIndex(name, ".")
	if dot <= slash {
		return name + ext
	}
	return name[:dot] + ext
}

func uniqueCount(ss []string) int {
	m := map[string]struct{}{}
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return len(m)
}

func baseDir() string {
	if p, err := os.Executable(); err == nil {
		return filepath.Dir(p)
	}
	wd, _ := os.Getwd()
	return wd
}
