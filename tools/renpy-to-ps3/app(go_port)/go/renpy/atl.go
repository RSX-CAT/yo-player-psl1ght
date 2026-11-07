package renpy

import (
	"strconv"
	"strings"
)

type AtlWarper byte

const (
	AtlInstant AtlWarper = iota
	AtlLinear
	AtlPause
	AtlEase
	AtlEaseIn
	AtlEaseOut
)

type AtlProp byte

const (
	AtlXpos AtlProp = iota
	AtlYpos
	AtlXanchor
	AtlYanchor
	AtlXalign
	AtlYalign
	AtlZoom
	AtlXzoom
	AtlYzoom
	AtlAlpha
	AtlRotate
)

type AtlKey struct {
	Warper AtlWarper
	DurMs  int
	Props  []struct {
		Prop  AtlProp
		Value int
	}
}

type AtlProgram struct {
	Keys        []AtlKey
	RepeatCount int
}

func (p *AtlProgram) Key() string {
	var b strings.Builder
	b.WriteByte('R')
	b.WriteString(strconv.Itoa(p.RepeatCount))
	b.WriteByte(';')
	for _, k := range p.Keys {
		b.WriteString(strconv.Itoa(int(k.Warper)))
		b.WriteByte('@')
		b.WriteString(strconv.Itoa(k.DurMs))
		b.WriteByte(':')
		for _, pr := range k.Props {
			b.WriteString(strconv.Itoa(int(pr.Prop)))
			b.WriteByte('=')
			b.WriteString(strconv.Itoa(pr.Value))
			b.WriteByte(',')
		}
		b.WriteByte(';')
	}
	return b.String()
}

func CompileAtl(block *PyObject, notes *[]string) *AtlProgram {
	if block == nil {
		return nil
	}
	st := stateDict(block)
	if st == nil {
		return nil
	}
	stmtsAny, _ := st.Get("statements")
	stmts := asSeq(stmtsAny)
	if stmts == nil {
		return nil
	}
	prog := &AtlProgram{}
	for _, so := range stmts {
		s := asPy(so)
		if s == nil {
			continue
		}
		cls := s.ClassName
		if strings.HasSuffix(cls, ".RawMultipurpose") {
			ms := stateDict(s)
			if ms == nil {
				continue
			}
			exprsAny, _ := ms.Get("expressions")
			propsAny, _ := ms.Get("properties")
			exprs := asSeq(exprsAny)
			props := asSeq(propsAny)
			key := AtlKey{}
			warper, _ := ms.Get("warper")
			key.Warper = mapWarper(asStr(warper))
			durS, _ := AsText(func() any { v, _ := ms.Get("duration"); return v }())
			var dur float64
			tryNum(durS, &dur)
			key.DurMs = int(dur*1000.0 + 0.5)
			for _, po := range props {
				a, b, ok := pair(po)
				if !ok {
					continue
				}
				pname, _ := a.(string)
				if pname == "" {
					pname, _ = AsText(a)
				}
				pval, ok := AsText(b)
				if !ok {
					pval, _ = b.(string)
				}
				addAtlProp(&key, pname, pval, notes)
			}
			if len(exprs) > 0 && len(key.Props) == 0 && key.Warper != AtlPause {
				if notes != nil {
					*notes = append(*notes, "ATL displayable/contains form not animated")
				}
				continue
			}
			if len(key.Props) > 0 || key.DurMs > 0 || key.Warper == AtlPause {
				prog.Keys = append(prog.Keys, key)
			}
		} else if strings.HasSuffix(cls, ".RawRepeat") {
			rs := stateDict(s)
			var rep string
			if rs != nil {
				v, _ := rs.Get("repeats")
				rep, _ = AsText(v)
			}
			n, err := strconv.Atoi(strings.TrimSpace(rep))
			if err == nil {
				prog.RepeatCount = n
			} else {
				prog.RepeatCount = -1
			}
		} else if notes != nil {
			*notes = append(*notes, "ATL statement unsupported: "+cls)
		}
	}
	if len(prog.Keys) == 0 {
		return nil
	}
	return prog
}

func addAtlProp(key *AtlKey, name, val string, notes *[]string) {
	if name == "" {
		return
	}
	name = strings.TrimSpace(name)
	var v float64
	if !tryNum(val, &v) {
		if notes != nil {
			if val == "" {
				val = "?"
			}
			*notes = append(*notes, "ATL non-constant property: "+name+" = "+val)
		}
		return
	}
	adj := 0.5
	if v < 0 {
		adj = -0.5
	}
	milli := int(v*1000.0 + adj)
	add := func(p AtlProp) {
		key.Props = append(key.Props, struct {
			Prop  AtlProp
			Value int
		}{p, milli})
	}
	switch name {
	case "xpos":
		add(AtlXpos)
	case "ypos":
		add(AtlYpos)
	case "xanchor":
		add(AtlXanchor)
	case "yanchor":
		add(AtlYanchor)
	case "xalign":
		add(AtlXalign)
	case "yalign":
		add(AtlYalign)
	case "zoom":
		add(AtlZoom)
	case "xzoom":
		add(AtlXzoom)
	case "yzoom":
		add(AtlYzoom)
	case "alpha":
		add(AtlAlpha)
	case "rotate":
		add(AtlRotate)
	default:
		if notes != nil {
			*notes = append(*notes, "ATL property unsupported: "+name)
		}
	}
}

func mapWarper(w string) AtlWarper {
	switch w {
	case "":
		return AtlInstant
	case "linear":
		return AtlLinear
	case "pause":
		return AtlPause
	case "ease":
		return AtlEase
	case "easein":
		return AtlEaseIn
	case "easeout":
		return AtlEaseOut
	default:
		return AtlLinear
	}
}

func stateDict(p *PyObject) *Dict {
	if t, ok := p.State.(Tuple); ok && len(t) >= 2 {
		return asDict(t[1])
	}
	return asDict(p.State)
}

func asStr(o any) string {
	if s, ok := o.(string); ok {
		return s
	}
	s, _ := AsText(o)
	return s
}

func pair(o any) (any, any, bool) {
	if t, ok := o.(Tuple); ok && len(t) >= 2 {
		return t[0], t[1], true
	}
	if l := asSeq(o); len(l) >= 2 {
		return l[0], l[1], true
	}
	return nil, nil, false
}

func tryNum(s string, v *float64) bool {
	if s == "" {
		*v = 0
		return false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		*v = 0
		return false
	}
	*v = f
	return true
}
