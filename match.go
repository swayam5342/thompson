package thompson

func (n *NFA) Match(input string) bool {
	return n.run(input, false).Matched
}

func MatchString(pattern, input string) (bool, error) {
	n, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return n.Match(input), nil
}

func EdgeID(state, slot int) int { return state*2 + slot }

type Step struct {
	Pos    int    `json:"pos"`
	Rune   string `json:"rune"`
	Fired  []int  `json:"fired"`
	Eps    []int  `json:"eps"`
	Active []int  `json:"active"`
}

type Trace struct {
	Pattern string   `json:"pattern"`
	Input   []string `json:"input"`
	Steps   []Step   `json:"steps"`
	Matched bool     `json:"matched"`
	DeadAt  int      `json:"deadAt"`
}

func (n *NFA) Trace(input string) *Trace {
	t := n.run(input, true)
	return &t
}

type sim struct {
	n      *NFA
	mark   []int
	gen    int
	record bool
	eps    []int
}

func (m *sim) add(list []int, id, via int) []int {
	if id < 0 || m.mark[id] == m.gen {
		return list
	}
	m.mark[id] = m.gen
	if m.record && via >= 0 {
		m.eps = append(m.eps, via)
	}
	list = append(list, id)
	s := m.n.States[id]
	switch s.Kind {
	case KindSplit:
		list = m.add(list, s.Out, EdgeID(id, 0))
		list = m.add(list, s.Out1, EdgeID(id, 1))
	case KindEmpty:
		list = m.add(list, s.Out, EdgeID(id, 0))
	}
	return list
}

func (m *sim) takeEps() []int {
	e := m.eps
	m.eps = nil
	return e
}

func (n *NFA) run(input string, record bool) Trace {
	m := &sim{n: n, mark: make([]int, len(n.States)), record: record}
	for i := range m.mark {
		m.mark[i] = -1
	}
	t := Trace{Pattern: n.Pattern, DeadAt: -1}

	clist := m.add(make([]int, 0, len(n.States)), n.Start, -1)
	nlist := make([]int, 0, len(n.States))
	if record {
		t.Steps = append(t.Steps, Step{Pos: -1, Eps: m.takeEps(), Active: clone(clist)})
	}

	pos := 0
	for _, r := range input {
		if record {
			t.Input = append(t.Input, string(r))
		}
		if t.DeadAt >= 0 {
			pos++
			continue
		}
		m.gen++
		nlist = nlist[:0]
		var fired []int
		for _, id := range clist {
			s := n.States[id]
			if s.consumes(r) {
				if record {
					fired = append(fired, EdgeID(id, 0))
				}
				nlist = m.add(nlist, s.Out, -1)
			}
		}
		clist, nlist = nlist, clist
		if record {
			t.Steps = append(t.Steps, Step{
				Pos: pos, Rune: string(r), Fired: fired, Eps: m.takeEps(), Active: clone(clist),
			})
		}
		if len(clist) == 0 {
			t.DeadAt = pos
			if !record {
				return t
			}
		}
		pos++
	}

	if t.DeadAt < 0 {
		for _, id := range clist {
			if id == n.Accept {
				t.Matched = true
			}
		}
	}
	return t
}

func clone(s []int) []int { return append([]int{}, s...) }
