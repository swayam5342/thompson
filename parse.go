package thompson

import (
	"fmt"
	"sort"
	"unicode"
)

type nodeKind int

const (
	nLit nodeKind = iota
	nAny
	nClass
	nEmpty
	nConcat
	nAlt
	nStar
	nPlus
	nQuest
)

type node struct {
	kind        nodeKind
	r           rune
	class       *charClass
	left, right *node
}

type SyntaxError struct {
	Pattern string
	Pos     int
	Msg     string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("regex: %s at position %d in %q", e.Msg, e.Pos, e.Pattern)
}

type parser struct {
	pattern string
	src     []rune
	pos     int
}

func parse(pattern string) (*node, error) {
	p := &parser{pattern: pattern, src: []rune(pattern)}
	n, err := p.parseAlt()
	if err != nil {
		return nil, err
	}
	if p.more() {
		return nil, p.errAt(p.pos, "unmatched ')'")
	}
	return n, nil
}

func (p *parser) more() bool { return p.pos < len(p.src) }
func (p *parser) peek() rune { return p.src[p.pos] }
func (p *parser) errAt(pos int, format string, args ...any) error {
	return &SyntaxError{Pattern: p.pattern, Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (p *parser) parseAlt() (*node, error) {
	left, err := p.parseConcat()
	if err != nil {
		return nil, err
	}
	for p.more() && p.peek() == '|' {
		p.pos++
		right, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		left = &node{kind: nAlt, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseConcat() (*node, error) {
	var result *node
	for p.more() && p.peek() != '|' && p.peek() != ')' {
		n, err := p.parseRepeat()
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = n
		} else {
			result = &node{kind: nConcat, left: result, right: n}
		}
	}
	if result == nil {
		return &node{kind: nEmpty}, nil
	}
	return result, nil
}

func (p *parser) parseRepeat() (*node, error) {
	n, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	for p.more() {
		switch p.peek() {
		case '*':
			n = &node{kind: nStar, left: n}
		case '+':
			n = &node{kind: nPlus, left: n}
		case '?':
			n = &node{kind: nQuest, left: n}
		default:
			return n, nil
		}
		p.pos++
	}
	return n, nil
}

func (p *parser) parseAtom() (*node, error) {
	start := p.pos
	c := p.peek()
	switch c {
	case '(':
		p.pos++
		inner, err := p.parseAlt()
		if err != nil {
			return nil, err
		}
		if !p.more() || p.peek() != ')' {
			return nil, p.errAt(start, "missing ')' for this '('")
		}
		p.pos++
		return inner, nil
	case '*', '+', '?':
		return nil, p.errAt(start, "nothing before '%c' to repeat", c)
	case '.':
		p.pos++
		return &node{kind: nAny}, nil
	case '[':
		cls, err := p.parseClass()
		if err != nil {
			return nil, err
		}
		return &node{kind: nClass, class: cls}, nil
	case '\\':
		r, cls, err := p.parseEscape()
		if err != nil {
			return nil, err
		}
		if cls != nil {
			cls.label = string(p.src[start:p.pos])
			return &node{kind: nClass, class: cls}, nil
		}
		return &node{kind: nLit, r: r}, nil
	}
	p.pos++
	return &node{kind: nLit, r: c}, nil
}

func (p *parser) parseEscape() (rune, *charClass, error) {
	start := p.pos
	p.pos++ // the backslash
	if !p.more() {
		return 0, nil, p.errAt(start, "trailing backslash")
	}
	c := p.peek()
	p.pos++
	switch c {
	case 'n':
		return '\n', nil, nil
	case 't':
		return '\t', nil, nil
	case 'r':
		return '\r', nil, nil
	case 'd', 'D', 'w', 'W', 's', 'S':
		ranges := perlClasses[unicode.ToLower(c)]
		return 0, &charClass{ranges: ranges, negate: unicode.IsUpper(c)}, nil
	}
	if unicode.IsLetter(c) || unicode.IsDigit(c) {
		return 0, nil, p.errAt(start, "unknown escape '\\%c'", c)
	}
	return c, nil, nil
}

var perlClasses = map[rune][]runeRange{
	'd': {{'0', '9'}},
	'w': {{'0', '9'}, {'A', 'Z'}, {'_', '_'}, {'a', 'z'}},
	's': {{'\t', '\n'}, {'\f', '\r'}, {' ', ' '}},
}

func (p *parser) parseClass() (*charClass, error) {
	start := p.pos
	p.pos++ // '['
	cls := &charClass{}
	if p.more() && p.peek() == '^' {
		cls.negate = true
		p.pos++
	}
	first := true
	for {
		if !p.more() {
			return nil, p.errAt(start, "missing ']' for this '['")
		}
		c := p.peek()
		if c == ']' && !first {
			p.pos++
			break
		}
		first = false

		lo, ranges, err := p.classAtom()
		if err != nil {
			return nil, err
		}
		if ranges != nil {
			cls.ranges = append(cls.ranges, ranges...)
			continue
		}
		hi := lo
		if p.pos+1 < len(p.src) && p.peek() == '-' && p.src[p.pos+1] != ']' {
			dash := p.pos
			p.pos++
			var hr []runeRange
			hi, hr, err = p.classAtom()
			if err != nil {
				return nil, err
			}
			if hr != nil {
				return nil, p.errAt(dash, "a range can't end in a class escape")
			}
			if hi < lo {
				return nil, p.errAt(dash, "range %c-%c is backwards", lo, hi)
			}
		}
		cls.ranges = append(cls.ranges, runeRange{lo, hi})
	}
	cls.label = string(p.src[start:p.pos])
	return cls, nil
}

func (p *parser) classAtom() (rune, []runeRange, error) {
	if p.peek() != '\\' {
		r := p.peek()
		p.pos++
		return r, nil, nil
	}
	r, cls, err := p.parseEscape()
	if err != nil {
		return 0, nil, err
	}
	if cls == nil {
		return r, nil, nil
	}
	if cls.negate {
		return 0, complement(cls.ranges), nil
	}
	return 0, cls.ranges, nil
}

type runeRange struct{ lo, hi rune }

type charClass struct {
	ranges []runeRange
	negate bool
	label  string
}

func (c *charClass) matches(r rune) bool {
	in := false
	for _, rg := range c.ranges {
		if rg.lo <= r && r <= rg.hi {
			in = true
			break
		}
	}
	return in != c.negate
}

func complement(rs []runeRange) []runeRange {
	sorted := append([]runeRange(nil), rs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].lo < sorted[j].lo })
	var out []runeRange
	next := rune(0)
	for _, rg := range sorted {
		if rg.lo > next {
			out = append(out, runeRange{next, rg.lo - 1})
		}
		if rg.hi+1 > next {
			next = rg.hi + 1
		}
	}
	if next <= unicode.MaxRune {
		out = append(out, runeRange{next, unicode.MaxRune})
	}
	return out
}
