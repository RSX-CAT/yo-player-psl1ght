package renpy

import (
	"regexp"
	"strconv"
	"strings"
)

type OvKind byte

const (
	OvImage OvKind = iota
	OvText
	OvImageButton
)

type OvWidget struct {
	Kind      OvKind
	X, Y      int
	A, B      string
	Action    string
	GuardExpr int
}

type OverlayDef struct {
	Name    string
	Widgets []OvWidget
}

var (
	ovDefRe   = regexp.MustCompile(`^(\s*)def\s+([A-Za-z_]\w*)\s*\(\s*\)\s*:`)
	ovIfRe    = regexp.MustCompile(`^(\s*)(if|elif)\s+(.+?)\s*:\s*$`)
	ovElseRe  = regexp.MustCompile(`^(\s*)else\s*:\s*$`)
	ovVboxRe  = regexp.MustCompile(`ui\.(?:vbox|fixed)\(([^)]*)\)`)
	ovCloseRe = regexp.MustCompile(`ui\.close\(\)`)
	ovImageRe = regexp.MustCompile(`ui\.image\(\s*[uU]?["']([^"']+)["']([^)]*)\)`)
	ovTextRe  = regexp.MustCompile(`ui\.text\((.*)\)\s*$`)
	ovBtnRe   = regexp.MustCompile(`ui\.imagebutton\(\s*[uU]?["']([^"']+)["']\s*,\s*[uU]?["']([^"']+)["']([^)]*\bclicked\s*=\s*[^,)]+)`)
	ovXposRe  = regexp.MustCompile(`xpos\s*=\s*(-?\d+)`)
	ovYposRe  = regexp.MustCompile(`ypos\s*=\s*(-?\d+)`)
	ovClickRe = regexp.MustCompile(`clicked\s*=\s*([A-Za-z_][\w.]*)\(\s*[uU]?["']([^"']+)["']\s*\)`)
)

func ParseOverlays(src string, ir *IrProgram) {
	if src == "" || !strings.Contains(src, "ui.") {
		return
	}
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		d := ovDefRe.FindStringSubmatch(lines[i])
		if d == nil {
			continue
		}
		defIndent := len(d[1])
		j := i + 1
		var body []string
		for ; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				body = append(body, lines[j])
				continue
			}
			if indent(lines[j]) <= defIndent {
				break
			}
			body = append(body, lines[j])
		}
		ov := parseOverlayBody(d[2], body, ir)
		if ov != nil && len(ov.Widgets) > 0 {
			ir.PutOverlay(ov)
		}
		i = j - 1
	}
}

func indent(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

type guardPair struct {
	ind  int
	cond string
}

func parseOverlayBody(name string, body []string, ir *IrProgram) *OverlayDef {
	ov := &OverlayDef{Name: name}
	var guards []guardPair
	chain := map[int][]string{}
	pendX, pendY, pendHave := 0, 0, 0

	for _, raw := range body {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		ind := indent(raw)
		for len(guards) > 0 && guards[len(guards)-1].ind >= ind {
			guards = guards[:len(guards)-1]
		}
		if ifm := ovIfRe.FindStringSubmatch(raw); ifm != nil {
			kw, cond := ifm[2], ifm[3]
			var prior []string
			var eff string
			if kw == "if" {
				prior = []string{}
				chain[ind] = prior
				eff = "(" + cond + ")"
			} else {
				prior = chain[ind]
				if prior == nil {
					prior = []string{}
					chain[ind] = prior
				}
				eff = negateAnd(prior, cond)
			}
			guards = append(guards, guardPair{ind, eff})
			chain[ind] = append(chain[ind], cond)
			continue
		}
		if ovElseRe.MatchString(raw) {
			prior := chain[ind]
			if prior == nil {
				prior = []string{}
			}
			guards = append(guards, guardPair{ind, negateAnd(prior, "")})
			delete(chain, ind)
			continue
		}
		delete(chain, ind)

		if ovVboxRe.MatchString(line) {
			v := ovVboxRe.FindStringSubmatch(line)
			readPos(v[1], &pendX, &pendY, &pendHave)
			continue
		}
		if ovCloseRe.MatchString(line) {
			pendHave = 0
			continue
		}

		guard := compileGuard(guards, ir)

		if b := ovBtnRe.FindStringSubmatch(line); b != nil {
			w := OvWidget{Kind: OvImageButton, A: b[1], B: b[2], GuardExpr: guard}
			if pendHave != 0 {
				w.X, w.Y = pendX, pendY
			}
			posFrom(b[3], &w)
			w.Action = parseAction(line)
			ov.Widgets = append(ov.Widgets, w)
			continue
		}
		if im := ovImageRe.FindStringSubmatch(line); im != nil {
			w := OvWidget{Kind: OvImage, A: im[1], GuardExpr: guard}
			if pendHave != 0 {
				w.X, w.Y = pendX, pendY
			}
			posFrom(im[2], &w)
			ov.Widgets = append(ov.Widgets, w)
			continue
		}
		if tx := ovTextRe.FindStringSubmatch(line); tx != nil {
			w := OvWidget{Kind: OvText, A: strings.TrimSpace(tx[1]), GuardExpr: guard}
			if pendHave != 0 {
				w.X, w.Y = pendX, pendY
			}
			posFrom(tx[1], &w)
			ov.Widgets = append(ov.Widgets, w)
			continue
		}
	}
	return ov
}

func negateAnd(priors []string, cond string) string {
	var parts []string
	for _, p := range priors {
		parts = append(parts, "not ("+p+")")
	}
	if cond != "" {
		parts = append(parts, "("+cond+")")
	}
	if len(parts) == 0 {
		return "True"
	}
	return strings.Join(parts, " and ")
}

func compileGuard(guards []guardPair, ir *IrProgram) int {
	var parts []string
	for _, g := range guards {
		if g.cond != "True" {
			parts = append(parts, "("+g.cond+")")
		}
	}
	if len(parts) == 0 {
		return -1
	}
	return ir.CompileExpr(strings.Join(parts, " and "))
}

func readPos(args string, x, y, have *int) {
	mx := ovXposRe.FindStringSubmatch(args)
	my := ovYposRe.FindStringSubmatch(args)
	*x, *y = 0, 0
	if mx != nil {
		*x, _ = strconv.Atoi(mx[1])
	}
	if my != nil {
		*y, _ = strconv.Atoi(my[1])
	}
	*have = 1
}

func posFrom(args string, w *OvWidget) {
	if mx := ovXposRe.FindStringSubmatch(args); mx != nil {
		w.X, _ = strconv.Atoi(mx[1])
	}
	if my := ovYposRe.FindStringSubmatch(args); my != nil {
		w.Y, _ = strconv.Atoi(my[1])
	}
}

func parseAction(line string) string {
	c := ovClickRe.FindStringSubmatch(line)
	if c == nil {
		return ""
	}
	fn, arg := c[1], c[2]
	if strings.HasSuffix(fn, "gamemenus") {
		return "menu:" + arg
	}
	return "call:" + arg
}
