package renpy

import (
	"strconv"
	"strings"
	"unicode"
)

type ExprOp byte

const (
	ExprPushInt ExprOp = iota
	ExprPushBool
	ExprPushNone
	ExprPushFloat
	ExprPushStr
	ExprLoadVar
	ExprNeg
	ExprNot
	ExprAdd
	ExprSub
	ExprMul
	ExprDiv
	ExprMod
	ExprEq
	ExprNe
	ExprLt
	ExprLe
	ExprGt
	ExprGe
	ExprAnd
	ExprOr
	ExprFloorDiv
	ExprMax
	ExprMin
)

type ExprInstr struct {
	Op  ExprOp
	Arg int
}

type ExprProgram struct {
	Ops []ExprInstr
}

func (p *ExprProgram) Emit(op ExprOp, arg int) {
	p.Ops = append(p.Ops, ExprInstr{Op: op, Arg: arg})
}

func (p *ExprProgram) Key() string {
	var b strings.Builder
	for _, i := range p.Ops {
		b.WriteString(strconv.Itoa(int(i.Op)))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(i.Arg))
		b.WriteByte(';')
	}
	return b.String()
}

type exprCompiler struct {
	s   string
	pos int
	ir  *IrProgram
	bad bool
}

func CompileExpr(src string, ir *IrProgram) *ExprProgram {
	if strings.TrimSpace(src) == "" {
		return nil
	}
	t := strings.TrimSpace(src)
	c := &exprCompiler{s: t, ir: ir}
	p := &ExprProgram{}
	c.parseOr(p)
	c.skipWs()
	if c.bad || c.pos < len(c.s) {
		return nil
	}
	if len(p.Ops) == 0 {
		return nil
	}
	return p
}

func (c *exprCompiler) skipWs() {
	for c.pos < len(c.s) {
		ch := c.s[c.pos]
		if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
			break
		}
		c.pos++
	}
}
func (c *exprCompiler) peek() byte {
	if c.pos < len(c.s) {
		return c.s[c.pos]
	}
	return 0
}
func (c *exprCompiler) peekAt(k int) byte {
	if c.pos+k < len(c.s) {
		return c.s[c.pos+k]
	}
	return 0
}

func (c *exprCompiler) matchWord(w string) bool {
	c.skipWs()
	if c.pos+len(w) > len(c.s) {
		return false
	}
	if c.s[c.pos:c.pos+len(w)] != w {
		return false
	}
	var after byte
	if c.pos+len(w) < len(c.s) {
		after = c.s[c.pos+len(w)]
	}
	if isIdentChar(after) {
		return false
	}
	c.pos += len(w)
	return true
}

func (c *exprCompiler) matchSym(sym string) bool {
	c.skipWs()
	if c.pos+len(sym) > len(c.s) {
		return false
	}
	if c.s[c.pos:c.pos+len(sym)] != sym {
		return false
	}
	c.pos += len(sym)
	return true
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}
func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '.'
}

func (c *exprCompiler) parseOr(p *ExprProgram) {
	c.parseAnd(p)
	for !c.bad && c.matchWord("or") {
		c.parseAnd(p)
		p.Emit(ExprOr, -1)
	}
}
func (c *exprCompiler) parseAnd(p *ExprProgram) {
	c.parseNot(p)
	for !c.bad && c.matchWord("and") {
		c.parseNot(p)
		p.Emit(ExprAnd, -1)
	}
}
func (c *exprCompiler) parseNot(p *ExprProgram) {
	if c.matchWord("not") {
		c.parseNot(p)
		p.Emit(ExprNot, -1)
		return
	}
	c.parseCmp(p)
}
func (c *exprCompiler) parseCmp(p *ExprProgram) {
	c.parseAdd(p)
	for !c.bad {
		c.skipWs()
		if c.matchWord("in") || c.matchWord("is") {
			c.bad = true
			return
		}
		var op ExprOp
		if c.matchSym("==") {
			op = ExprEq
		} else if c.matchSym("!=") {
			op = ExprNe
		} else if c.matchSym("<=") {
			op = ExprLe
		} else if c.matchSym(">=") {
			op = ExprGe
		} else if c.peek() == '<' && c.peekAt(1) != '<' {
			c.pos++
			op = ExprLt
		} else if c.peek() == '>' && c.peekAt(1) != '>' {
			c.pos++
			op = ExprGt
		} else {
			break
		}
		c.parseAdd(p)
		p.Emit(op, -1)
	}
}
func (c *exprCompiler) parseAdd(p *ExprProgram) {
	c.parseMul(p)
	for !c.bad {
		c.skipWs()
		if c.peek() == '+' {
			c.pos++
			c.parseMul(p)
			p.Emit(ExprAdd, -1)
		} else if c.peek() == '-' {
			c.pos++
			c.parseMul(p)
			p.Emit(ExprSub, -1)
		} else {
			break
		}
	}
}
func (c *exprCompiler) parseMul(p *ExprProgram) {
	c.parseUnary(p)
	for !c.bad {
		c.skipWs()
		if c.peek() == '*' && c.peekAt(1) == '*' {
			c.bad = true
			return
		}
		if c.peek() == '*' {
			c.pos++
			c.parseUnary(p)
			p.Emit(ExprMul, -1)
		} else if c.peek() == '/' && c.peekAt(1) == '/' {
			c.pos += 2
			c.parseUnary(p)
			p.Emit(ExprFloorDiv, -1)
		} else if c.peek() == '/' {
			c.pos++
			c.parseUnary(p)
			p.Emit(ExprDiv, -1)
		} else if c.peek() == '%' {
			c.pos++
			c.parseUnary(p)
			p.Emit(ExprMod, -1)
		} else {
			break
		}
	}
}
func (c *exprCompiler) parseUnary(p *ExprProgram) {
	c.skipWs()
	if c.peek() == '-' {
		c.pos++
		c.parseUnary(p)
		p.Emit(ExprNeg, -1)
		return
	}
	if c.peek() == '+' {
		c.pos++
		c.parseUnary(p)
		return
	}
	c.parseAtom(p)
}
func (c *exprCompiler) parseAtom(p *ExprProgram) {
	c.skipWs()
	ch := c.peek()
	if ch == 0 {
		c.bad = true
		return
	}
	if ch == '(' {
		c.pos++
		c.parseOr(p)
		c.skipWs()
		if c.peek() != ')' {
			c.bad = true
			return
		}
		c.pos++
		c.rejectTrailer()
		return
	}
	if ch == '\'' || ch == '"' {
		c.parseString(p, ch)
		c.rejectTrailer()
		return
	}
	if ch >= '0' && ch <= '9' {
		c.parseNumber(p)
		c.rejectTrailer()
		return
	}
	if ch == '.' && c.peekAt(1) >= '0' && c.peekAt(1) <= '9' {
		c.parseNumber(p)
		c.rejectTrailer()
		return
	}
	if isIdentStart(ch) {
		c.parseNameOrKeyword(p)
		return
	}
	c.bad = true
}

func (c *exprCompiler) rejectTrailer() {
	c.skipWs()
	if c.peek() == '(' || c.peek() == '[' {
		c.bad = true
	}
}

func (c *exprCompiler) parseString(p *ExprProgram, quote byte) {
	c.pos++
	var b strings.Builder
	for c.pos < len(c.s) {
		ch := c.s[c.pos]
		c.pos++
		if ch == '\\' && c.pos < len(c.s) {
			e := c.s[c.pos]
			c.pos++
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(e)
			}
			continue
		}
		if ch == quote {
			p.Emit(ExprPushStr, c.ir.Intern(b.String()))
			return
		}
		b.WriteByte(ch)
	}
	c.bad = true
}

func (c *exprCompiler) parseNumber(p *ExprProgram) {
	start := c.pos
	isFloat := false
	for c.pos < len(c.s) {
		ch := c.s[c.pos]
		if ch >= '0' && ch <= '9' {
			c.pos++
		} else if ch == '.' {
			isFloat = true
			c.pos++
		} else if ch == 'e' || ch == 'E' {
			isFloat = true
			c.pos++
			if c.peek() == '+' || c.peek() == '-' {
				c.pos++
			}
		} else {
			break
		}
	}
	tok := c.s[start:c.pos]
	if isFloat {
		if _, err := strconv.ParseFloat(tok, 64); err != nil {
			c.bad = true
			return
		}
		p.Emit(ExprPushFloat, c.ir.Intern(tok))
	} else {
		lv, err := strconv.ParseInt(tok, 10, 64)
		if err != nil {
			c.bad = true
			return
		}
		p.Emit(ExprPushInt, int(lv))
	}
}

func (c *exprCompiler) parseNameOrKeyword(p *ExprProgram) {
	start := c.pos
	for c.pos < len(c.s) && isIdentChar(c.s[c.pos]) {
		c.pos++
	}
	name := c.s[start:c.pos]
	if name == "max" || name == "min" {
		c.skipWs()
		if c.peek() == '(' {
			c.pos++
			fold := ExprMax
			if name == "min" {
				fold = ExprMin
			}
			c.parseOr(p)
			args := 1
			for !c.bad && c.matchSym(",") {
				c.parseOr(p)
				p.Emit(fold, -1)
				args++
			}
			c.skipWs()
			if c.bad || args < 2 || c.peek() != ')' {
				c.bad = true
				return
			}
			c.pos++
			c.rejectTrailer()
			return
		}
	}
	if name == "True" {
		p.Emit(ExprPushBool, 1)
		c.rejectTrailer()
		return
	}
	if name == "False" {
		p.Emit(ExprPushBool, 0)
		c.rejectTrailer()
		return
	}
	if name == "None" {
		p.Emit(ExprPushNone, -1)
		c.rejectTrailer()
		return
	}
	if name == "and" || name == "or" || name == "not" || name == "in" || name == "is" {
		c.bad = true
		return
	}
	p.Emit(ExprLoadVar, c.ir.Intern(name))
	c.rejectTrailer()
}

var _ = unicode.IsLetter
