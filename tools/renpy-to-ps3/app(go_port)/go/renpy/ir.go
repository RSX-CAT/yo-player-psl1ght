package renpy

// IrOp is the flat instruction set the PS3 runtime interprets.
type IrOp byte

const (
	IrLabel IrOp = iota
	IrSay
	IrScene
	IrShow
	IrHide
	IrWith
	IrJump
	IrCall
	IrReturn
	IrMenuStart
	IrChoice
	IrMenuEnd
	IrIfFalseGoto
	IrPyExec
	IrDefault
	IrImage
	IrUser
	IrPause
	IrAssign
	IrNop
	IrEnd
	IrImageMap
	IrOverlayShow
	IrOverlayHide
)

func (op IrOp) String() string {
	names := []string{
		"Label", "Say", "Scene", "Show", "Hide", "With", "Jump", "Call", "Return",
		"MenuStart", "Choice", "MenuEnd", "IfFalseGoto", "PyExec", "Default", "Image",
		"User", "Pause", "Assign", "Nop", "End", "ImageMap", "OverlayShow", "OverlayHide",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

type Instr struct {
	Op      IrOp
	A, B, C int
	Sym     string
}

func NewInstr(op IrOp, abc ...int) Instr {
	in := Instr{Op: op, A: -1, B: -1, C: -1}
	if len(abc) > 0 {
		in.A = abc[0]
	}
	if len(abc) > 1 {
		in.B = abc[1]
	}
	if len(abc) > 2 {
		in.C = abc[2]
	}
	return in
}

type IrProgram struct {
	Code         []Instr
	Strings      []string
	Labels       map[string]int
	labelOrder   []string
	Unsupported  []string
	Unresolved   []string
	Notes        []string
	Exprs        []ExprProgram
	exprIntern   map[string]int
	Atls         []AtlProgram
	atlIntern    map[string]int
	ImageMaps    []ImageMapDef
	imIntern     map[string]int
	Overlays     map[string]*OverlayDef
	overlayOrder []string
	intern       map[string]int
	InitCode     []Instr
	toInit       bool
}

func NewIrProgram() *IrProgram {
	return &IrProgram{
		Labels:     map[string]int{},
		Overlays:   map[string]*OverlayDef{},
		exprIntern: map[string]int{},
		atlIntern:  map[string]int{},
		imIntern:   map[string]int{},
		intern:     map[string]int{},
	}
}

func (p *IrProgram) Intern(s string) int {
	if s == "" {
		// still intern empty
	}
	if i, ok := p.intern[s]; ok {
		return i
	}
	i := len(p.Strings)
	p.Strings = append(p.Strings, s)
	p.intern[s] = i
	return i
}

func (p *IrProgram) Str(id int) string {
	if id >= 0 && id < len(p.Strings) {
		return p.Strings[id]
	}
	return ""
}

func (p *IrProgram) SetLabel(name string, addr int) {
	if _, exists := p.Labels[name]; !exists {
		p.labelOrder = append(p.labelOrder, name)
	}
	p.Labels[name] = addr
}

func (p *IrProgram) LabelPairs() [][2]any {
	out := make([][2]any, 0, len(p.labelOrder))
	for _, n := range p.labelOrder {
		out = append(out, [2]any{n, p.Labels[n]})
	}
	for n, addr := range p.Labels {
		found := false
		for _, o := range p.labelOrder {
			if o == n {
				found = true
				break
			}
		}
		if !found {
			out = append(out, [2]any{n, addr})
		}
	}
	return out
}

func (p *IrProgram) RegisterOverlays(pySrc string) {
	ParseOverlays(pySrc, p)
}

func (p *IrProgram) PutOverlay(ov *OverlayDef) {
	if _, exists := p.Overlays[ov.Name]; !exists {
		p.overlayOrder = append(p.overlayOrder, ov.Name)
	}
	p.Overlays[ov.Name] = ov
}

func (p *IrProgram) OverlayList() []*OverlayDef {
	out := make([]*OverlayDef, 0, len(p.overlayOrder))
	for _, n := range p.overlayOrder {
		if ov, ok := p.Overlays[n]; ok {
			out = append(out, ov)
		}
	}
	return out
}

func (p *IrProgram) BeginInit() { p.toInit = true }
func (p *IrProgram) EndInit()   { p.toInit = false }

// Emit appends instr and returns its index in Code (or -1-initIndex when buffering init).
func (p *IrProgram) Emit(instr Instr) int {
	if p.toInit {
		p.InitCode = append(p.InitCode, instr)
		return -len(p.InitCode)
	}
	p.Code = append(p.Code, instr)
	return len(p.Code) - 1
}

func (p *IrProgram) at(idx int) *Instr {
	if idx < 0 {
		return &p.InitCode[-idx-1]
	}
	return &p.Code[idx]
}

func (p *IrProgram) Here() int { return len(p.Code) }

func (p *IrProgram) CompileExpr(src string) int {
	ep := CompileExpr(src, p)
	if ep == nil {
		return -1
	}
	key := ep.Key()
	if i, ok := p.exprIntern[key]; ok {
		return i
	}
	i := len(p.Exprs)
	p.Exprs = append(p.Exprs, *ep)
	p.exprIntern[key] = i
	return i
}

func (p *IrProgram) CompileAtl(node *AstNode) int {
	if node == nil {
		return -1
	}
	ap := CompileAtl(node.AtlObj(), &p.Notes)
	if ap == nil {
		return -1
	}
	key := ap.Key()
	if i, ok := p.atlIntern[key]; ok {
		return i
	}
	i := len(p.Atls)
	p.Atls = append(p.Atls, *ap)
	p.atlIntern[key] = i
	return i
}

func (p *IrProgram) CompileImageMap(src string) int {
	d := ParseImageMap(src)
	if d == nil {
		return -1
	}
	return p.internImageMap(*d)
}

func (p *IrProgram) CollectThemedImageMaps(src string) {
	for _, d := range ParseThemedImageMaps(src) {
		p.internImageMap(d)
	}
}

func (p *IrProgram) internImageMap(d ImageMapDef) int {
	key := d.Key()
	if i, ok := p.imIntern[key]; ok {
		return i
	}
	i := len(p.ImageMaps)
	p.ImageMaps = append(p.ImageMaps, d)
	p.imIntern[key] = i
	return i
}
