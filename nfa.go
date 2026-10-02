package thompson

import "fmt"

type StateKind int

const (
	KindRune StateKind = iota
	KindAny
	KindClass
	KindSplit
	KindEmpty
	KindMatch
)

func (k StateKind) String() string {
	return [...]string{"rune", "any", "class", "split", "empty", "match"}[k]
}

type State struct {
	ID    int
	Kind  StateKind
	Rune  rune
	Class *charClass
	Out   int
	Out1  int
}

func (s *State) consumes(r rune) bool {
	switch s.Kind {
	case KindRune:
		return s.Rune == r
	case KindAny:
		return r != '\n'
	case KindClass:
		return s.Class.matches(r)
	}
	return false
}

func (s *State) Label() string {
	switch s.Kind {
	case KindRune:
		return displayRune(s.Rune)
	case KindAny:
		return "."
	case KindClass:
		return s.Class.label
	}
	return "ε"
}

func displayRune(r rune) string {
	switch r {
	case ' ':
		return "␣"
	case '\n':
		return `\n`
	case '\t':
		return `\t`
	case '\r':
		return `\r`
	}
	return string(r)
}

type NFA struct {
	Pattern string
	States  []*State
	Start   int
	Accept  int
}

const MaxStates = 20000

func Compile(pattern string) (*NFA, error) {
	ast, err := parse(pattern)
	if err != nil {
		return nil, err
	}
	b := &builder{}
	f := b.build(ast)
	accept := b.newState(KindMatch)
	f.patch(accept.ID)
	if len(b.states) > MaxStates {
		return nil, fmt.Errorf("regex: pattern %q needs %d states, limit is %d", pattern, len(b.states), MaxStates)
	}
	n := &NFA{Pattern: pattern, States: b.states, Start: f.start, Accept: accept.ID}
	n.renumber()
	return n, nil
}

func MustCompile(pattern string) *NFA {
	n, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return n
}

type frag struct {
	start int
	out   []*int
}

func (f frag) patch(target int) {
	for _, p := range f.out {
		*p = target
	}
}

type builder struct{ states []*State }

func (b *builder) newState(kind StateKind) *State {
	s := &State{ID: len(b.states), Kind: kind, Out: -1, Out1: -1}
	b.states = append(b.states, s)
	return s
}

func (b *builder) build(n *node) frag {
	switch n.kind {
	case nLit, nAny, nClass:
		kind := map[nodeKind]StateKind{nLit: KindRune, nAny: KindAny, nClass: KindClass}[n.kind]
		s := b.newState(kind)
		s.Rune, s.Class = n.r, n.class
		return frag{s.ID, []*int{&s.Out}}

	case nEmpty:
		s := b.newState(KindEmpty)
		return frag{s.ID, []*int{&s.Out}}

	case nConcat:
		e1, e2 := b.build(n.left), b.build(n.right)
		e1.patch(e2.start)
		return frag{e1.start, e2.out}

	case nAlt:
		e1, e2 := b.build(n.left), b.build(n.right)
		s := b.newState(KindSplit)
		s.Out, s.Out1 = e1.start, e2.start
		return frag{s.ID, append(e1.out, e2.out...)}

	case nStar:
		e := b.build(n.left)
		s := b.newState(KindSplit)
		s.Out = e.start
		e.patch(s.ID)
		return frag{s.ID, []*int{&s.Out1}}

	case nPlus:
		e := b.build(n.left)
		s := b.newState(KindSplit)
		s.Out = e.start
		e.patch(s.ID)
		return frag{e.start, []*int{&s.Out1}}

	case nQuest:
		e := b.build(n.left)
		s := b.newState(KindSplit)
		s.Out = e.start
		return frag{s.ID, append(e.out, &s.Out1)}
	}
	panic("unreachable")
}
func (n *NFA) renumber() {
	newID := make([]int, len(n.States))
	for i := range newID {
		newID[i] = -1
	}
	var order []*State
	stack := []int{n.Start}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if id < 0 || newID[id] >= 0 {
			continue
		}
		newID[id] = len(order)
		s := n.States[id]
		order = append(order, s)
		stack = append(stack, s.Out1, s.Out)
	}
	remap := func(id int) int {
		if id < 0 {
			return -1
		}
		return newID[id]
	}
	for _, s := range order {
		s.ID, s.Out, s.Out1 = remap(s.ID), remap(s.Out), remap(s.Out1)
	}
	n.Start, n.Accept, n.States = remap(n.Start), remap(n.Accept), order
}
