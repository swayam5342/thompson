//go:build js && wasm

// Command wasm exposes the engine to JavaScript as globalThis.thompson:
//
//	thompson.compile(pattern)       -> JSON {graph} or {error}
//	thompson.trace(pattern, input)  -> JSON {graph, trace} or {error}
//
// Build: GOOS=js GOARCH=wasm go build -o web/main.wasm ./wasm
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/swayam5342/thompson"
)

type result struct {
	Graph *thompson.Graph `json:"graph,omitempty"`
	Trace *thompson.Trace `json:"trace,omitempty"`
	Error string          `json:"error,omitempty"`
	Pos   int             `json:"pos"`
}

func run(pattern string, input *string) string {
	var r result
	re, err := thompson.Compile(pattern)
	if err != nil {
		r.Error, r.Pos = err.Error(), -1
		if se, ok := err.(*thompson.SyntaxError); ok {
			r.Error, r.Pos = se.Msg, se.Pos
		}
	} else {
		g := re.Graph()
		r.Graph = &g
		if input != nil {
			r.Trace = re.Trace(*input)
		}
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("compile", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return run(args[0].String(), nil)
	}))
	api.Set("trace", js.FuncOf(func(_ js.Value, args []js.Value) any {
		in := args[1].String()
		return run(args[0].String(), &in)
	}))
	js.Global().Set("thompson", api)
	select {} // keep the Go runtime alive for callbacks
}
