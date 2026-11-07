package renpy

import (
	"strings"
)

// AstNode is a typed, read-only view over a captured Ren'Py AST node.
type AstNode struct {
	Obj *PyObject
}

func (n AstNode) Short() string {
	c := n.Obj.ClassName
	if strings.HasPrefix(c, "renpy.ast.") {
		return c[len("renpy.ast."):]
	}
	return c
}

func (n AstNode) attrs() *Dict {
	if t, ok := n.Obj.State.(Tuple); ok && len(t) >= 2 {
		return asDict(t[1])
	}
	return asDict(n.Obj.State)
}

func (n AstNode) Raw(key string) any {
	a := n.attrs()
	if a == nil {
		return nil
	}
	v, _ := a.Get(key)
	return v
}

func (n AstNode) Text(key string) string {
	s, _ := AsText(n.Raw(key))
	return s
}

// AsText normalizes a value to text: plain string, or the source of a PyExpr.
// ok is false when the value is nil / not text.
func AsText(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	if p := asPy(v); p != nil && strings.HasSuffix(p.ClassName, ".PyExpr") {
		if len(p.Args) > 0 {
			if s, ok := p.Args[0].(string); ok {
				return s, true
			}
		}
		if st, ok := p.State.(Tuple); ok && len(st) > 0 {
			if s, ok := st[0].(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

func textOrEmpty(v any) string {
	s, ok := AsText(v)
	if ok {
		return s
	}
	return ""
}

func (n AstNode) NodeAt(key string) *AstNode {
	p := asPy(n.Raw(key))
	if p == nil {
		return nil
	}
	return &AstNode{Obj: p}
}

func (n AstNode) Block(key string) []AstNode { return ToNodes(n.Raw(key)) }

func ToNodes(listish any) []AstNode {
	var r []AstNode
	for _, it := range asSeq(listish) {
		if p := asPy(it); p != nil {
			r = append(r, AstNode{Obj: p})
		}
	}
	return r
}

func (n AstNode) PyCodeSource() string {
	if t, ok := n.Obj.State.(Tuple); ok && len(t) > 1 {
		if s, ok := t[1].(string); ok {
			return s
		}
	}
	if code := n.NodeAt("code"); code != nil {
		return code.PyCodeSource()
	}
	return ""
}

func (n AstNode) AtlObj() *PyObject {
	return asPy(n.Raw("atl"))
}

func (n AstNode) ImageName() string {
	parts := asSeq(n.Raw("imgname"))
	if parts == nil {
		return ""
	}
	var b strings.Builder
	for _, x := range parts {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(toString(x))
	}
	return b.String()
}

func (n AstNode) ImspecAtList() string { return n.ImspecAtListKey("imspec") }

func (n AstNode) ImspecAtListKey(key string) string {
	spec, _ := n.Raw(key).(Tuple)
	if spec == nil {
		if seq := asSeq(n.Raw(key)); seq != nil {
			spec = Tuple(seq)
		}
	}
	if spec == nil {
		return ""
	}
	var at []any
	if len(spec) == 3 {
		at = asSeq(spec[1])
	} else if len(spec) >= 6 {
		at = asSeq(spec[3])
	}
	if at == nil {
		return ""
	}
	var b strings.Builder
	for _, x := range at {
		s, ok := AsText(x)
		if !ok {
			s = toString(x)
		}
		if s == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s)
	}
	return b.String()
}

func (n AstNode) ImspecName() string { return n.ImspecNameKey("imspec") }

func (n AstNode) ImspecNameKey(key string) string {
	spec, _ := n.Raw(key).(Tuple)
	if spec == nil {
		if seq := asSeq(n.Raw(key)); len(seq) > 0 {
			spec = Tuple(seq)
		}
	}
	if len(spec) > 0 {
		parts := asSeq(spec[0])
		if parts != nil {
			var b strings.Builder
			for _, x := range parts {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(toString(x))
			}
			return b.String()
		}
	}
	return ""
}

func (n AstNode) LabelName() string {
	nm := n.Raw("name")
	if s, ok := nm.(string); ok {
		return s
	}
	if s, ok := n.Raw("_name").(string); ok {
		return s
	}
	if t, ok := AsText(nm); ok {
		return t
	}
	serial := n.Raw("name_serial")
	if serial != nil {
		fn := n.Raw("filename")
		ver := n.Raw("name_version")
		fnS, verS := "?", "0"
		if fn != nil {
			fnS = toString(fn)
		}
		if ver != nil {
			verS = toString(ver)
		}
		return fnS + ":" + verS + ":" + toString(serial)
	}
	return ""
}

func (n AstNode) HasLabelName() bool {
	return n.Raw("name") != nil || n.Raw("_name") != nil || n.Raw("name_serial") != nil
}
