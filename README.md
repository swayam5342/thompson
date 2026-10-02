# thompson

A regular expression engine in Go, built the way Ken Thompson described in
1968: parse the pattern, compile it to an NFA by patching together small
fragments, then simulate every path at once. Matching is linear in the input
for every pattern, so `(x+x+)+y` can't blow up the way it does in a
backtracking engine.

`web/` is an interactive page that runs this exact engine (compiled to
WebAssembly) and animates the state machine reading a string one character
at a time.

## Supported syntax

| Syntax            | Meaning                                    |
|-------------------|--------------------------------------------|
| `abc`             | literals (full Unicode)                    |
| `.`               | any character except `\n`                  |
| `a\|b`            | alternation                                |
| `a*` `a+` `a?`    | zero or more, one or more, zero or one     |
| `( … )`           | grouping                                   |
| `[a-z_]` `[^0-9]` | character classes, negated classes         |
| `\d \w \s`        | digit, word, space (and `\D \W \S`)        |
| `\n \t \r`, `\.`  | escapes; any punctuation can be escaped    |

Matching is anchored at both ends. Use `.*(…).*` to search inside a string.

## Use it

```go
re, err := thompson.Compile(`(a|b)*abb`)
re.Match("babaabb")             // true
re.Trace("babaabb")             // every state set, step by step
re.DOT()                        // Graphviz source for the NFA
```

```sh
go run ./cmd/thompson '(a|b)*abb' abb abab
go run ./cmd/thompson -trace '(a|b)*abb' aabb
go run ./cmd/thompson -dot '(a|b)*abb' | dot -Tsvg > nfa.svg
```

## Layout

| File               | What it does                                                    |
|--------------------|-----------------------------------------------------------------|
| `parse.go`         | recursive-descent parser, pattern → syntax tree                 |
| `nfa.go`           | Thompson's construction, syntax tree → NFA                      |
| `match.go`         | set-of-states simulation, plus `Trace` for the visualizer       |
| `graph.go`         | NFA as JSON-ready nodes/edges, and DOT                          |
| `thompson_test.go` | unit tests, 3,000 random patterns diffed against `regexp`       |
| `wasm/`            | exposes `thompson.compile` / `thompson.trace` to JavaScript     |
| `cmd/bundle/`      | inlines everything into one standalone HTML file                |

## Put it on your blog

```sh
make            # test, build web/main.wasm, build dist/thompson-nfa.html
```

Pick one:

1. **One file.** Upload `dist/thompson-nfa.html` anywhere and link to it or
   embed it: `<iframe src="/thompson-nfa.html" style="width:100%;height:1400px;border:0"></iframe>`
2. **Folder.** Upload `web/` as is (`index.html`, `main.wasm`, `wasm_exec.js`,
   `vendor/`). Smaller first load because the server can cache `main.wasm`.
   Serve `.wasm` with any MIME type; the page reads it as bytes.

Link to a specific example with the URL hash:
`thompson-nfa.html#p=(0|1(01*0)*1)*&s=1001`.

If a host blocks WebAssembly, the page falls back to a JavaScript port of the
same engine (checked identical against the Go build on 78,000 pattern/input
pairs) and says so in its footer.

`web/vendor/dagre.min.js` is [dagre](https://github.com/dagrejs/dagre)
(MIT), used for graph layout.
