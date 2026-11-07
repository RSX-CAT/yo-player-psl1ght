package renpy

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func Compile(statements []any) *IrProgram {
	return CompileUnits([][]any{statements}, false)
}

func CompileUnits(units [][]any, normalizeText bool) *IrProgram {
	p := NewIrProgram()
	for _, u := range units {
		for _, n := range ToNodes(u) {
			s := n.Short()
			if s == "Init" || s == "EarlyPython" || s == "Python" {
				p.BeginInit()
				lower(n, p)
				p.EndInit()
			} else {
				lower(n, p)
			}
		}
	}
	resolveCharacters(p)
	if normalizeText {
		normalizeDisplay(p)
	}
	if len(p.InitCode) > 0 {
		p.SetLabel("__init__", p.Here())
		p.Emit(NewInstr(IrLabel, p.Intern("__init__")))
		p.Code = append(p.Code, p.InitCode...)
		p.Emit(NewInstr(IrReturn))
	}
	p.Emit(NewInstr(IrEnd))
	resolveJumps(p)
	return p
}

func normalizeDisplay(p *IrProgram) {
	for i := range p.Code {
		ins := &p.Code[i]
		if ins.Op == IrSay {
			if ins.A >= 0 {
				ins.A = p.Intern(normalizeASCII(p.Str(ins.A)))
			}
			ins.B = p.Intern(normalizeASCII(p.Str(ins.B)))
		} else if ins.Op == IrChoice {
			ins.A = p.Intern(normalizeASCII(p.Str(ins.A)))
		}
	}
}

func normalizeASCII(s string) string {
	if s == "" {
		return s
	}
	any := false
	for _, c := range s {
		if c > 127 {
			any = true
			break
		}
	}
	if !any {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		switch c {
		case 0x2018, 0x2019, 0x2032:
			b.WriteByte('\'')
		case 0x201C, 0x201D, 0x2033:
			b.WriteByte('"')
		case 0x2013, 0x2014:
			b.WriteByte('-')
		case 0x2026:
			b.WriteString("...")
		case 0x00A0:
			b.WriteByte(' ')
		case 0x2022:
			b.WriteByte('*')
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

func lower(n AstNode, p *IrProgram) {
	switch n.Short() {
	case "Label":
		name := n.LabelName()
		if !n.HasLabelName() {
			name = "@" + strconv.Itoa(p.Here())
		}
		p.SetLabel(name, p.Here())
		p.Emit(NewInstr(IrLabel, p.Intern(n.LabelName())))
		for _, c := range n.Block("block") {
			lower(c, p)
		}
	case "Say":
		who, whoOk := AsText(n.Raw("who"))
		what, whatOk := AsText(n.Raw("what"))
		if !whatOk {
			what = ""
		}
		if seq := asSeq(n.Raw("attributes")); len(seq) > 0 {
			w := "?"
			if whoOk {
				w = who
			}
			p.Notes = append(p.Notes, "say-attributes ignored: "+w+" (+"+strconv.Itoa(len(seq))+" attr)")
		}
		whoId := -1
		if whoOk {
			whoId = p.Intern(who)
		}
		p.Emit(NewInstr(IrSay, whoId, p.Intern(what)))
	case "Scene":
		nn := n
		p.Emit(NewInstr(IrScene, p.Intern(n.ImspecName()), -1, p.CompileAtl(&nn)))
	case "Show":
		nn := n
		p.Emit(NewInstr(IrShow, p.Intern(n.ImspecName()), p.Intern(n.ImspecAtList()), p.CompileAtl(&nn)))
	case "Hide":
		p.Emit(NewInstr(IrHide, p.Intern(n.ImspecName())))
	case "With":
		e, ok := AsText(n.Raw("expr"))
		if !ok {
			e = ""
		}
		p.Emit(NewInstr(IrWith, p.Intern(e)))
	case "Jump":
		ins := NewInstr(IrJump)
		ins.Sym = n.Text("target")
		p.Emit(ins)
	case "Call":
		ins := NewInstr(IrCall)
		ins.Sym = n.Text("label")
		p.Emit(ins)
	case "Return":
		p.Emit(NewInstr(IrReturn))
	case "Pass":
		p.Emit(NewInstr(IrNop))
	case "Python", "EarlyPython", "Init":
		src := n.PyCodeSource()
		if src != "" {
			p.Emit(NewInstr(IrPyExec, p.Intern(src)))
			emitNativeCalls(src, p)
			emitAssignments(src, p)
		} else {
			for _, c := range n.Block("block") {
				lower(c, p)
			}
		}
	case "Default", "Define":
		varname, ok := AsText(n.Raw("varname"))
		if !ok {
			vn := n.Raw("varname")
			if vn == nil {
				varname = ""
			} else {
				varname = toString(vn)
			}
		}
		code := codeOf(n)
		valExpr := p.CompileExpr(code)
		if code == "" {
			p.Emit(NewInstr(IrDefault, p.Intern(varname), p.Intern(""), valExpr))
		} else {
			p.Emit(NewInstr(IrDefault, p.Intern(varname), p.Intern(code), valExpr))
		}
	case "Image":
		code := codeOf(n)
		nn := n
		c := ""
		if code != "" {
			c = code
		}
		p.Emit(NewInstr(IrImage, p.Intern(n.ImageName()), p.Intern(c), p.CompileAtl(&nn)))
	case "UserStatement":
		line, _ := AsText(n.Raw("line"))
		p.Emit(NewInstr(IrUser, p.Intern(line)))
	case "Menu":
		lowerMenu(n, p)
	case "If":
		lowerIf(n, p)
	case "While":
		lowerWhile(n, p)
	default:
		p.Unsupported = append(p.Unsupported, n.Short())
		p.Emit(NewInstr(IrNop))
	}
}

type menuChoice struct {
	cap, cond int
	block     []AstNode
}

func lowerMenu(n AstNode, p *IrProgram) {
	items := asSeq(n.Raw("items"))
	var choices []menuChoice
	captionId := -1
	for _, it := range items {
		t, ok := it.(Tuple)
		if !ok {
			seq := asSeq(it)
			if len(seq) < 2 {
				continue
			}
			t = Tuple(seq)
		}
		if len(t) < 2 {
			continue
		}
		capTxt, _ := AsText(t[0])
		if !(len(t) > 2 && t[2] != nil) {
			captionId = p.Intern(capTxt)
			continue
		}
		cap := p.Intern(capTxt)
		condTxt, condOk := AsText(t[1])
		cond := -1
		if condOk && condTxt != "True" {
			cond = p.CompileExpr(condTxt)
			if cond < 0 {
				p.Unsupported = append(p.Unsupported, "menu-condition: "+condTxt)
			}
		}
		var block []AstNode
		if len(t) > 2 {
			block = ToNodes(t[2])
		}
		choices = append(choices, menuChoice{cap, cond, block})
	}
	p.Emit(NewInstr(IrMenuStart, len(choices), captionId))
	var choiceInstrs []int
	for _, c := range choices {
		choiceInstrs = append(choiceInstrs, p.Emit(NewInstr(IrChoice, c.cap, c.cond)))
	}
	p.Emit(NewInstr(IrMenuEnd))
	var endJumps []int
	for idx, ch := range choices {
		p.at(choiceInstrs[idx]).C = p.Here()
		for _, cn := range ch.block {
			lower(cn, p)
		}
		endJumps = append(endJumps, p.Emit(NewInstr(IrJump)))
	}
	end := p.Here()
	for _, j := range endJumps {
		p.at(j).A = end
	}
}

func lowerIf(n AstNode, p *IrProgram) {
	entries := asSeq(n.Raw("entries"))
	var endJumps []int
	for _, e := range entries {
		t, ok := e.(Tuple)
		if !ok {
			seq := asSeq(e)
			if len(seq) < 2 {
				continue
			}
			t = Tuple(seq)
		}
		if len(t) < 2 {
			continue
		}
		cond, _ := AsText(t[0])
		block := ToNodes(t[1])
		skip := -999999
		if cond != "" && cond != "True" {
			condExpr := p.CompileExpr(cond)
			if condExpr >= 0 {
				skip = p.Emit(NewInstr(IrIfFalseGoto, condExpr))
			} else {
				p.Unsupported = append(p.Unsupported, "if-condition: "+cond)
			}
		}
		for _, c := range block {
			lower(c, p)
		}
		endJumps = append(endJumps, p.Emit(NewInstr(IrJump)))
		if skip != -999999 {
			p.at(skip).C = p.Here()
		}
	}
	end := p.Here()
	for _, j := range endJumps {
		p.at(j).A = end
	}
}

func lowerWhile(n AstNode, p *IrProgram) {
	cond, condOk := AsText(n.Raw("condition"))
	condExpr := -1
	if condOk && cond != "True" {
		condExpr = p.CompileExpr(cond)
	}
	if condOk && cond != "True" && condExpr < 0 {
		p.Unsupported = append(p.Unsupported, "while-condition: "+cond)
		over := p.Emit(NewInstr(IrJump))
		for _, c := range n.Block("block") {
			lower(c, p)
		}
		p.at(over).A = p.Here()
		return
	}
	top := p.Here()
	skip := -999999
	if condExpr >= 0 {
		skip = p.Emit(NewInstr(IrIfFalseGoto, condExpr))
	}
	for _, c := range n.Block("block") {
		lower(c, p)
	}
	back := NewInstr(IrJump)
	back.A = top
	p.Emit(back)
	if skip != -999999 {
		p.at(skip).C = p.Here()
	}
}

func codeOf(n AstNode) string {
	if code := n.NodeAt("code"); code != nil {
		return code.PyCodeSource()
	}
	s, _ := AsText(n.Raw("code"))
	return s
}

var assignStmt = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_.]*)\s*(\+=|-=|\*=|/=|%=|=)\s*(.+?)\s*$`)
var imageMapAssign = regexp.MustCompile(`(?s)(\w+)\s*=\s*renpy\.imagemap\(`)
var jumpCall = regexp.MustCompile(`renpy\.jump(?:_out_of_context)?\(\s*[uU]?["']([^"']+)["']\s*\)`)
var quitCall = regexp.MustCompile(`renpy\.quit\(\s*\)`)
var overlayAppend = regexp.MustCompile(`config\.overlay_functions\.append\(\s*([A-Za-z_]\w*)\s*\)`)
var overlayRemove = regexp.MustCompile(`config\.overlay_functions\.remove\(\s*([A-Za-z_]\w*)\s*\)`)

func emitNativeCalls(src string, p *IrProgram) {
	if src == "" {
		return
	}
	if im := imageMapAssign.FindStringSubmatch(src); im != nil {
		id := p.CompileImageMap(src)
		if id >= 0 {
			p.Emit(NewInstr(IrImageMap, p.Intern(im[1]), id))
		}
	}
	p.CollectThemedImageMaps(src)
	for _, jm := range jumpCall.FindAllStringSubmatch(src, -1) {
		ins := NewInstr(IrJump)
		ins.Sym = jm[1]
		p.Emit(ins)
	}
	if quitCall.MatchString(src) {
		p.Emit(NewInstr(IrEnd))
	}
	for _, am := range overlayAppend.FindAllStringSubmatch(src, -1) {
		p.Emit(NewInstr(IrOverlayShow, p.Intern(am[1])))
	}
	for _, rm := range overlayRemove.FindAllStringSubmatch(src, -1) {
		p.Emit(NewInstr(IrOverlayHide, p.Intern(rm[1])))
	}
	p.RegisterOverlays(src)
}

func emitAssignments(src string, p *IrProgram) {
	if src == "" {
		return
	}
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n"), "\n")
	defIndent := -1
	for _, rawLine := range lines {
		line := rawLine
		if hash := indexOfComment(line); hash >= 0 {
			line = line[:hash]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		ind := 0
		for ind < len(line) && (line[ind] == ' ' || line[ind] == '\t') {
			ind++
		}
		if defIndent >= 0 {
			if ind > defIndent {
				continue
			}
			defIndent = -1
		}
		head := line[ind:]
		if strings.HasPrefix(head, "def ") || strings.HasPrefix(head, "class ") {
			defIndent = ind
			continue
		}
		for _, stmt := range strings.Split(line, ";") {
			m := assignStmt.FindStringSubmatch(stmt)
			if m == nil {
				continue
			}
			variable, opTok, rhs := m[1], m[2], m[3]
			if opTok == "=" && strings.HasPrefix(rhs, "=") {
				continue
			}
			valExpr := p.CompileExpr(rhs)
			if valExpr < 0 {
				p.Notes = append(p.Notes, "assign-rhs not executable: "+strings.TrimSpace(stmt))
				continue
			}
			kind := -1
			switch opTok {
			case "=":
				kind = 0
			case "+=":
				kind = 1
			case "-=":
				kind = 2
			case "*=":
				kind = 3
			case "/=":
				kind = 4
			case "%=":
				kind = 5
			default:
				continue
			}
			p.Emit(NewInstr(IrAssign, p.Intern(variable), kind, valExpr))
		}
	}
}

func indexOfComment(s string) int {
	var q byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if q != 0 {
			if c == q && (i == 0 || s[i-1] != '\\') {
				q = 0
			}
		} else if c == '\'' || c == '"' {
			q = c
		} else if c == '#' {
			return i
		}
	}
	return -1
}

var charAssign = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*Character\(\s*([^,)]*)`)
var charFirstArg = regexp.MustCompile(`Character\(\s*([^,)]*)`)
var quotedStr = regexp.MustCompile(`["']([^"']*)["']`)
var charVarDef = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*Character\(`)
var whatColorRe = regexp.MustCompile(`what_color\s*=\s*["']([^"']+)["']`)
var whatPrefixRe = regexp.MustCompile(`what_prefix\s*=\s*["']([^"']*)["']`)

func charNameOf(arg1 string) (string, bool) {
	arg1 = strings.TrimSpace(arg1)
	if strings.HasPrefix(arg1, "None") {
		return "", true
	}
	if q := quotedStr.FindStringSubmatch(arg1); q != nil {
		return q[1], true
	}
	return "", false
}

func isNvlDef(s string) bool {
	return s != "" && strings.Contains(strings.ReplaceAll(s, " ", ""), "kind=nvl")
}

func balancedCall(s string, openIdx int) string {
	depth := 0
	for k := openIdx; k < len(s); k++ {
		if s[k] == '(' {
			depth++
		} else if s[k] == ')' {
			depth--
			if depth == 0 {
				return s[openIdx : k+1]
			}
		}
	}
	return s[openIdx:]
}

func resolveCharacters(p *IrProgram) {
	cmap := map[string]string{}
	nvlVars := map[string]struct{}{}
	whatColor := map[string]string{}
	whatPrefix := map[string]string{}
	ingameVars := map[string]struct{}{}

	defs := append([]Instr{}, p.Code...)
	defs = append(defs, p.InitCode...)
	for _, ins := range defs {
		if ins.Op == IrDefault {
			val := p.Str(ins.B)
			if strings.Contains(val, "Character(") {
				if m := charFirstArg.FindStringSubmatch(val); m != nil {
					if nm, ok := charNameOf(m[1]); ok {
						cmap[p.Str(ins.A)] = nm
					}
				}
				if isNvlDef(val) {
					nvlVars[p.Str(ins.A)] = struct{}{}
				}
				scanCharStyle(p.Str(ins.A), val, whatColor, whatPrefix, ingameVars)
			}
		} else if ins.Op == IrPyExec {
			src := p.Str(ins.A)
			for _, m := range charAssign.FindAllStringSubmatch(src, -1) {
				if nm, ok := charNameOf(m[2]); ok {
					cmap[m[1]] = nm
				}
			}
			// full call for style: find each Character( after var =
			idx := 0
			for {
				m := charAssign.FindStringSubmatchIndex(src[idx:])
				if m == nil {
					break
				}
				abs := idx + m[0]
				open := strings.IndexByte(src[abs:], '(')
				if open >= 0 {
					varname := src[idx+m[2] : idx+m[3]]
					scanCharStyle(varname, balancedCall(src, abs+open), whatColor, whatPrefix, ingameVars)
				}
				idx = idx + m[1]
			}
			for _, line := range strings.Split(src, "\n") {
				if !strings.Contains(line, "Character(") || !isNvlDef(line) {
					continue
				}
				if v := charVarDef.FindStringSubmatch(line); v != nil {
					nvlVars[v[1]] = struct{}{}
				}
			}
		}
	}
	if len(cmap) == 0 && len(nvlVars) == 0 && len(whatColor) == 0 && len(whatPrefix) == 0 && len(ingameVars) == 0 {
		return
	}
	for i := range p.Code {
		ins := &p.Code[i]
		if ins.Op != IrSay || ins.A < 0 {
			continue
		}
		who := p.Str(ins.A)
		if _, ok := nvlVars[who]; ok {
			ins.C = 1
		} else if _, ok := ingameVars[who]; ok {
			ins.C = 2
		}
		pfx, hasPfx := whatPrefix[who]
		clr, hasClr := whatColor[who]
		hasPfx = hasPfx && pfx != ""
		hasClr = hasClr && clr != ""
		if hasPfx || hasClr {
			txt := p.Str(ins.B)
			if hasPfx {
				txt = pfx + txt
			}
			if hasClr {
				txt = "{color=" + clr + "}" + txt + "{/color}"
			}
			ins.B = p.Intern(txt)
		}
		if name, ok := cmap[who]; ok {
			if name == "" {
				ins.A = -1
			} else {
				ins.A = p.Intern(name)
			}
		}
	}
}

func scanCharStyle(variable, call string, whatColor, whatPrefix map[string]string, ingameVars map[string]struct{}) {
	if variable == "" || call == "" {
		return
	}
	if c := whatColorRe.FindStringSubmatch(call); c != nil {
		whatColor[variable] = c[1]
	}
	if pf := whatPrefixRe.FindStringSubmatch(call); pf != nil {
		whatPrefix[variable] = pf[1]
	}
	if strings.Contains(call, "window_background") {
		ingameVars[variable] = struct{}{}
	}
}

func resolveJumps(p *IrProgram) {
	for i := range p.Code {
		ins := &p.Code[i]
		if ins.Sym == "" {
			continue
		}
		if ins.Op == IrJump {
			ins.B = p.Intern(ins.Sym)
		}
		if addr, ok := p.Labels[ins.Sym]; ok {
			ins.A = addr
		} else {
			ins.A = -1
			p.Unresolved = append(p.Unresolved, ins.Sym)
		}
	}
}

var _ = unicode.IsSpace
