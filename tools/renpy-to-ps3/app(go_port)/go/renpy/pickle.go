package renpy

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// LoadPickle unpickles a Python pickle (protocols 0/2-5 subset that Ren'Py emits).
func LoadPickle(b []byte) (any, error) {
	m := &pickleMachine{memo: map[int]any{}}
	v, err := m.run(b)
	if err != nil {
		return nil, err
	}
	return unwrapDeep(v), nil
}

type pickleMachine struct {
	stack []any
	memo  map[int]any
}

var pickleMark = &struct{ mark bool }{true}

func (m *pickleMachine) top() any   { return m.stack[len(m.stack)-1] }
func (m *pickleMachine) push(o any) { m.stack = append(m.stack, o) }
func (m *pickleMachine) pop() any {
	v := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return v
}
func (m *pickleMachine) memoGet(idx int) any { return m.memo[idx] }

func (m *pickleMachine) popMarkIndex() (int, error) {
	for k := len(m.stack) - 1; k >= 0; k-- {
		if m.stack[k] == pickleMark {
			return k, nil
		}
	}
	return -1, fmt.Errorf("pickle: mark not found")
}

func (m *pickleMachine) popToMarkTuple() ([]any, error) {
	idx, err := m.popMarkIndex()
	if err != nil {
		return nil, err
	}
	items := append([]any{}, m.stack[idx+1:]...)
	m.stack = m.stack[:idx]
	return items, nil
}

func (m *pickleMachine) run(b []byte) (res any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("pickle: %v", rec)
		}
	}()
	i, n := 0, len(b)
	for i < n {
		op := b[i]
		i++
		switch op {
		case 0x80:
			i++
		case 0x95:
			i += 8
		case '.':
			return m.top(), nil
		case '(':
			m.push(pickleMark)
		case '}':
			m.push(NewDict())
		case ']':
			m.push(wrapList(nil))
		case ')':
			m.push(Tuple{})
		case 0x8f:
			m.push(&Set{})
		case 'N':
			m.push(nil)
		case 0x88:
			m.push(true)
		case 0x89:
			m.push(false)
		case 'J':
			m.push(int(i32(b, &i)))
		case 'K':
			m.push(int(b[i]))
			i++
		case 'M':
			m.push(int(b[i]) | int(b[i+1])<<8)
			i += 2
		case 'I':
			m.push(parseIntText(readLine(b, &i)))
		case 'L':
			s := readLine(b, &i)
			s = strings.TrimSuffix(s, "L")
			m.push(parseIntText(s))
		case 0x8a:
			ln := int(b[i])
			i++
			m.push(longLE(b, &i, ln))
		case 0x8b:
			ln := i32(b, &i)
			m.push(longLE(b, &i, ln))
		case 'G':
			m.push(binFloat(b, &i))
		case 'F':
			f, e := strconv.ParseFloat(readLine(b, &i), 64)
			if e != nil {
				return nil, e
			}
			m.push(f)
		case 'U':
			ln := int(b[i])
			i++
			m.push(takeLatin1(b, &i, ln))
		case 'T':
			ln := i32(b, &i)
			m.push(takeLatin1(b, &i, ln))
		case 'S':
			m.push(unquote(readLine(b, &i)))
		case 0x8c:
			ln := int(b[i])
			i++
			m.push(takeUtf8(b, &i, ln))
		case 'X':
			ln := i32(b, &i)
			m.push(takeUtf8(b, &i, ln))
		case 0x8d:
			ln := int(i64(b, &i))
			m.push(takeUtf8(b, &i, ln))
		case 'V':
			m.push(readLine(b, &i))
		case 'C':
			ln := int(b[i])
			i++
			m.push(takeBytes(b, &i, ln))
		case 'B':
			ln := i32(b, &i)
			m.push(takeBytes(b, &i, ln))
		case 0x8e, 0x96:
			ln := int(i64(b, &i))
			m.push(takeBytes(b, &i, ln))
		case 't':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			m.push(Tuple(items))
		case 0x85:
			a := m.pop()
			m.push(Tuple{a})
		case 0x86:
			y := m.pop()
			x := m.pop()
			m.push(Tuple{x, y})
		case 0x87:
			z := m.pop()
			y := m.pop()
			x := m.pop()
			m.push(Tuple{x, y, z})
		case 'l':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			m.push(wrapList(items))
		case 'a':
			v := m.pop()
			appendOne(m.top(), v)
		case 'e':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			appendMany(m.top(), items)
		case 'd':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			dd := NewDict()
			for k := 0; k+1 < len(items); k += 2 {
				dd.Set(items[k], items[k+1])
			}
			m.push(dd)
		case 's':
			v := m.pop()
			k := m.pop()
			setItem(m.top(), k, v)
		case 'u':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			setItems(m.top(), items)
		case 0x90:
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			if hs, ok := m.top().(*Set); ok {
				for _, it := range items {
					hs.Add(it)
				}
			}
		case 0x94:
			m.memo[len(m.memo)] = m.top()
		case 'q':
			m.memo[int(b[i])] = m.top()
			i++
		case 'r':
			idx := i32(b, &i)
			m.memo[idx] = m.top()
		case 'p':
			e := newlineEnd(b, i)
			idx := int(parseLong(latin1(b[i:e])))
			m.memo[idx] = m.top()
			i = e + 1
		case 'h':
			m.push(m.memoGet(int(b[i])))
			i++
		case 'j':
			m.push(m.memoGet(i32(b, &i)))
		case 'g':
			e := newlineEnd(b, i)
			m.push(m.memoGet(int(parseLong(latin1(b[i:e])))))
			i = e + 1
		case 'c':
			mod := readLine(b, &i)
			nm := readLine(b, &i)
			m.push(&GlobalRef{Module: mod, Name: nm})
		case 0x93:
			nm := m.pop()
			mod := m.pop()
			m.push(&GlobalRef{Module: toString(mod), Name: toString(nm)})
		case 'R':
			args := m.pop()
			callable := m.pop()
			m.push(construct(callable, args))
		case 0x81:
			args := m.pop()
			cls := m.pop()
			m.push(construct(cls, args))
		case 0x92:
			m.pop()
			args := m.pop()
			cls := m.pop()
			m.push(construct(cls, args))
		case 'o':
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			var cls any
			if len(items) > 0 {
				cls = items[0]
			}
			m.push(construct(cls, subArray(items, 1)))
		case 'i':
			mod := readLine(b, &i)
			nm := readLine(b, &i)
			items, e := m.popToMarkTuple()
			if e != nil {
				return nil, e
			}
			m.push(construct(&GlobalRef{Module: mod, Name: nm}, items))
		case 'b':
			state := m.pop()
			build(m.top(), state)
		case 0x82:
			i++
		case 0x83:
			i += 2
		case 0x84:
			i += 4
		default:
			return nil, fmt.Errorf("pickle: unhandled opcode 0x%02X at %d", op, i-1)
		}
	}
	return nil, fmt.Errorf("pickle: no STOP opcode")
}

func subArray(a []any, start int) []any {
	if start >= len(a) {
		return []any{}
	}
	return append([]any{}, a[start:]...)
}

type listBox struct{ items []any }

func wrapList(items []any) *listBox {
	if items == nil {
		items = []any{}
	}
	return &listBox{items: items}
}

func appendOne(target, value any) {
	switch t := target.(type) {
	case *listBox:
		t.items = append(t.items, value)
	case *PyObject:
		t.ListItems = append(t.AsList(), value)
	}
}

func appendMany(target any, items []any) {
	switch t := target.(type) {
	case *listBox:
		t.items = append(t.items, items...)
	case *PyObject:
		t.ListItems = append(t.AsList(), items...)
	}
}

func setItem(target, key, value any) {
	switch t := target.(type) {
	case *Dict:
		t.Set(key, value)
	case *PyObject:
		t.AsDict().Set(key, value)
	}
}

func setItems(target any, items []any) {
	for k := 0; k+1 < len(items); k += 2 {
		setItem(target, items[k], items[k+1])
	}
}

func asArgs(argsObj any) []any {
	switch t := argsObj.(type) {
	case nil:
		return []any{}
	case []any:
		return t
	case Tuple:
		return []any(t)
	case List:
		return []any(t)
	case *listBox:
		return t.items
	default:
		return []any{argsObj}
	}
}

func construct(callable, argsObj any) any {
	args := asArgs(argsObj)
	if g, ok := callable.(*GlobalRef); ok {
		full := g.Full()
		if full == "copy_reg._reconstructor" || full == "copyreg._reconstructor" {
			var cg *GlobalRef
			if len(args) > 0 {
				cg, _ = args[0].(*GlobalRef)
			}
			name := "object"
			if cg != nil {
				name = cg.Full()
			}
			obj := &PyObject{ClassName: name}
			var bg *GlobalRef
			if len(args) > 1 {
				bg, _ = args[1].(*GlobalRef)
			}
			if bg != nil {
				switch bg.Name {
				case "list":
					obj.AsList()
				case "dict", "OrderedDict", "defaultdict":
					obj.AsDict()
				}
			}
			return obj
		}
		if full == "copy_reg.__newobj__" || full == "copyreg.__newobj__" {
			var rest any = []any{}
			if len(args) > 1 {
				rest = subArray(args, 1)
			}
			var first any
			if len(args) > 0 {
				first = args[0]
			}
			return construct(first, rest)
		}
		if full == "builtins.list" || full == "__builtin__.list" {
			l := []any{}
			if len(args) > 0 {
				if _, isStr := args[0].(string); !isStr {
					if seq := asSeq(args[0]); seq != nil {
						l = append(l, seq...)
					}
				}
			}
			return wrapList(l)
		}
		if full == "builtins.dict" || full == "__builtin__.dict" || full == "collections.OrderedDict" || full == "collections.defaultdict" {
			return NewDict()
		}
		if full == "builtins.set" || full == "__builtin__.set" {
			return &Set{}
		}
		return &PyObject{ClassName: g.Full(), Args: args}
	}
	name := "?"
	if callable != nil {
		name = toString(callable)
	}
	return &PyObject{ClassName: name, Args: args}
}

func build(target, state any) {
	if p, ok := target.(*PyObject); ok {
		p.State = state
		return
	}
	d := asDict(target)
	sd := asDict(state)
	if d != nil && sd != nil {
		for _, e := range sd.Items() {
			d.Set(e.k, e.v)
		}
	}
}

func unwrapDeep(v any) any {
	return unwrapVisit(v, map[any]any{})
}

// unwrapVisit walks a pickle graph. Ren'Py statements point at each other
// (next, parent, block), so a visit map stops those cycles.
func unwrapVisit(v any, seen map[any]any) any {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case *listBox, *Dict, *PyObject:
		if prev, ok := seen[v]; ok {
			return prev
		}
	}
	switch x := v.(type) {
	case *listBox:
		out := make(List, len(x.items))
		seen[x] = out
		for i, it := range x.items {
			out[i] = unwrapVisit(it, seen)
		}
		return out
	case List:
		out := make(List, len(x))
		for i, it := range x {
			out[i] = unwrapVisit(it, seen)
		}
		return out
	case Tuple:
		out := make(Tuple, len(x))
		for i, it := range x {
			out[i] = unwrapVisit(it, seen)
		}
		return out
	case *Dict:
		seen[x] = x
		for _, e := range x.Items() {
			x.Set(e.k, unwrapVisit(e.v, seen))
		}
		return x
	case *PyObject:
		seen[x] = x
		if x.Args != nil {
			for i, a := range x.Args {
				x.Args[i] = unwrapVisit(a, seen)
			}
		}
		x.State = unwrapVisit(x.State, seen)
		if x.ListItems != nil {
			for i, a := range x.ListItems {
				x.ListItems[i] = unwrapVisit(a, seen)
			}
		}
		if x.DictItems != nil {
			unwrapVisit(x.DictItems, seen)
		}
		return x
	default:
		return v
	}
}

func i32(b []byte, i *int) int {
	v := int(int32(binary.LittleEndian.Uint32(b[*i:])))
	*i += 4
	return v
}

func i64(b []byte, i *int) int64 {
	v := int64(binary.LittleEndian.Uint64(b[*i:]))
	*i += 8
	return v
}

func longLE(b []byte, i *int, length int) any {
	if length == 0 {
		return 0
	}
	if length > 8 {
		panic("pickle LONG larger than 64-bit not supported")
	}
	var v int64
	for k := length - 1; k >= 0; k-- {
		v = (v << 8) | int64(b[*i+k])
	}
	if length < 8 && b[*i+length-1]&0x80 != 0 {
		for k := length; k < 8; k++ {
			v |= int64(0xFF) << (k * 8)
		}
	}
	*i += length
	if v >= math.MinInt32 && v <= math.MaxInt32 {
		return int(v)
	}
	return v
}

func binFloat(b []byte, i *int) float64 {
	var tmp [8]byte
	for k := 0; k < 8; k++ {
		tmp[k] = b[*i+7-k]
	}
	*i += 8
	return math.Float64frombits(binary.LittleEndian.Uint64(tmp[:]))
}

func takeLatin1(b []byte, i *int, n int) string {
	s := latin1(b[*i : *i+n])
	*i += n
	return s
}
func takeUtf8(b []byte, i *int, n int) string {
	s := utf8Slice(b[*i : *i+n])
	*i += n
	return s
}
func takeBytes(b []byte, i *int, n int) []byte {
	r := append([]byte{}, b[*i:*i+n]...)
	*i += n
	return r
}

func newlineEnd(b []byte, i int) int {
	for i < len(b) && b[i] != '\n' {
		i++
	}
	return i
}

func readLine(b []byte, i *int) string {
	start := *i
	for *i < len(b) && b[*i] != '\n' {
		*i++
	}
	s := latin1(b[start:*i])
	*i++
	return strings.TrimSuffix(s, "\r")
}

func parseIntText(s string) any {
	l, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("pickle: integer too large for 64-bit")
	}
	if l >= math.MinInt32 && l <= math.MaxInt32 {
		return int(l)
	}
	return l
}

func parseLong(s string) int64 {
	l, _ := strconv.ParseInt(s, 10, 64)
	return l
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\'`, "'")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
