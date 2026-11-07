package renpy

import (
	"fmt"
	"strconv"
)

// GlobalRef is a Python class/callable encountered via GLOBAL / STACK_GLOBAL.
type GlobalRef struct {
	Module string
	Name   string
}

func (g *GlobalRef) Full() string { return g.Module + "." + g.Name }
func (g *GlobalRef) String() string {
	if g == nil {
		return ""
	}
	return g.Full()
}

// PyObject is a captured Python instance (unknown class).
type PyObject struct {
	ClassName string
	Args      []any
	State     any
	ListItems []any
	DictItems *Dict
}

func (p *PyObject) AsList() []any {
	if p.ListItems == nil {
		p.ListItems = []any{}
	}
	return p.ListItems
}

func (p *PyObject) AsDict() *Dict {
	if p.DictItems == nil {
		p.DictItems = NewDict()
	}
	return p.DictItems
}

func (p *PyObject) String() string { return "<" + p.ClassName + ">" }

// Tuple is a pickle tuple (distinct from a list).
type Tuple []any

// List is a pickle list.
type List []any

// Set is a pickle set.
type Set struct {
	items []any
}

func (s *Set) Add(v any) {
	for _, it := range s.items {
		if same(it, v) {
			return
		}
	}
	s.items = append(s.items, v)
}

// Dict is a pickle dict. String keys use a map; other keys fall back to a linear list
// so unhashable Python keys (tuples) don't panic.
type Dict struct {
	str   map[string]any
	extra []kv
	order []any
}

type kv struct{ k, v any }

func NewDict() *Dict { return &Dict{str: make(map[string]any)} }

func (d *Dict) Len() int {
	if d == nil {
		return 0
	}
	return len(d.str) + len(d.extra)
}

func (d *Dict) Set(k, v any) {
	if d.str == nil {
		d.str = make(map[string]any)
	}
	if s, ok := k.(string); ok {
		if _, exists := d.str[s]; !exists {
			d.order = append(d.order, s)
		}
		d.str[s] = v
		return
	}
	for i, e := range d.extra {
		if same(e.k, k) {
			d.extra[i].v = v
			return
		}
	}
	d.extra = append(d.extra, kv{k, v})
	d.order = append(d.order, k)
}

func (d *Dict) Get(k any) (any, bool) {
	if d == nil {
		return nil, false
	}
	if s, ok := k.(string); ok {
		v, ok := d.str[s]
		return v, ok
	}
	for _, e := range d.extra {
		if same(e.k, k) {
			return e.v, true
		}
	}
	return nil, false
}

func (d *Dict) Has(k any) bool {
	_, ok := d.Get(k)
	return ok
}

func (d *Dict) Items() []kv {
	if d == nil {
		return nil
	}
	out := make([]kv, 0, d.Len())
	seen := map[string]bool{}
	for _, k := range d.order {
		if s, ok := k.(string); ok {
			if v, exists := d.str[s]; exists {
				out = append(out, kv{s, v})
				seen[s] = true
			}
			continue
		}
		if v, ok := d.Get(k); ok {
			out = append(out, kv{k, v})
		}
	}
	for s, v := range d.str {
		if !seen[s] {
			out = append(out, kv{s, v})
		}
	}
	return out
}

func same(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	return fmt.Sprintf("%T:%v", a, a) == fmt.Sprintf("%T:%v", b, b)
}

// asSeq flattens list/tuple/Go slices into a sequential view (IList).
func asSeq(v any) []any {
	switch x := v.(type) {
	case nil:
		return nil
	case List:
		return []any(x)
	case Tuple:
		return []any(x)
	case []any:
		return x
	case *listBox:
		return x.items
	}
	return nil
}

func asDict(v any) *Dict {
	d, _ := v.(*Dict)
	return d
}

func asPy(v any) *PyObject {
	p, _ := v.(*PyObject)
	return p
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	}
	return 0, false
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case *GlobalRef:
		return x.Full()
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}
