package thompson

import (
	"fmt"
	"strings"
)

type GraphNode struct {
	ID     int    `json:"id"`
	Kind   string `json:"kind"`
	Start  bool   `json:"start,omitempty"`
	Accept bool   `json:"accept,omitempty"`
}

type GraphEdge struct {
	ID      int    `json:"id"`
	From    int    `json:"from"`
	To      int    `json:"to"`
	Label   string `json:"label"`
	Epsilon bool   `json:"epsilon"`
}

type Graph struct {
	Pattern string      `json:"pattern"`
	Start   int         `json:"start"`
	Accept  int         `json:"accept"`
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
}

func (n *NFA) Graph() Graph {
	g := Graph{Pattern: n.Pattern, Start: n.Start, Accept: n.Accept}
	for _, s := range n.States {
		g.Nodes = append(g.Nodes, GraphNode{
			ID: s.ID, Kind: s.Kind.String(), Start: s.ID == n.Start, Accept: s.ID == n.Accept,
		})
		eps := s.Kind == KindSplit || s.Kind == KindEmpty
		for slot, to := range []int{s.Out, s.Out1} {
			if to >= 0 {
				g.Edges = append(g.Edges, GraphEdge{
					ID: EdgeID(s.ID, slot), From: s.ID, To: to, Label: s.Label(), Epsilon: eps,
				})
			}
		}
	}
	return g
}

func (n *NFA) DOT() string {
	var b strings.Builder
	fmt.Fprintf(&b, "digraph nfa {\n  rankdir=LR;\n  label=%q;\n  node [shape=circle fontname=monospace];\n", n.Pattern)
	fmt.Fprintf(&b, "  start [shape=point];\n  start -> %d;\n  %d [shape=doublecircle];\n", n.Start, n.Accept)
	for _, e := range n.Graph().Edges {
		style := ""
		if e.Epsilon {
			style = " style=dashed"
		}
		fmt.Fprintf(&b, "  %d -> %d [label=%q%s];\n", e.From, e.To, e.Label, style)
	}
	b.WriteString("}\n")
	return b.String()
}
