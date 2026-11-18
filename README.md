<div align="center">

<img src="docs/icon.svg" width="88" height="88" alt="thompson">

# thompson

**Regex that can't blow up.**

`thompson` is a regular expression engine written in **Go**, built the way Ken Thompson described in 1968: **parse the pattern**, **compile it to an NFA** by patching together small fragments, then **simulate every path at once**. Matching is linear in the input for every pattern, so `(x+x+)+y` can't blow up the way it does in a backtracking engine.

[![Go](https://img.shields.io/badge/Go-1.22%2B-4cb3f0?style=flat-square&logo=go&logoColor=white&labelColor=0d3a55)](go.mod)
[![Matching](https://img.shields.io/badge/matching-linear%20time-4cb3f0?style=flat-square&logo=speedtest&logoColor=white&labelColor=0d3a55)](#supported-syntax)
[![Tested](https://img.shields.io/badge/fuzzed-3%2C000%20patterns%20vs%20regexp-4cb3f0?style=flat-square&logo=go&logoColor=white&labelColor=0d3a55)](#verification--testing)
[![Wasm](https://img.shields.io/badge/runs%20in-WebAssembly-4cb3f0?style=flat-square&logo=webassembly&logoColor=white&labelColor=0d3a55)](cmd/wasm)

[**Interactive demo**](https://blog.swayam.li/projects/thompson) &nbsp;&middot;&nbsp; [Supported Syntax](#supported-syntax) &nbsp;&middot;&nbsp; [Library](#library) &nbsp;&middot;&nbsp; [CLI](#cli)

</div>

<br>

> [!TIP]
> [blog.swayam.li/projects/thompson](https://blog.swayam.li/projects/thompson) runs this exact engine, compiled to WebAssembly, and animates the state machine reading a string one character at a time.

<br>

## Table of Contents

| | |
| :--- | :--- |
| **Overview** | [How It Works](#how-it-works) &middot; [Supported Syntax](#supported-syntax) |
| **Use** | [Library](#library) &middot; [CLI](#cli) |
| **Reference** | [Layout](#layout) |
| **Testing** | [Verification & Testing](#verification--testing) |

<br>

## How It Works

| | Step | File | What happens |
| :---: | :--- | :--- | :--- |
| 1 | **Parse** | `parse.go` | Recursive descent turns the pattern into a syntax tree. |
| 2 | **Compile** | `nfa.go` | Thompson's construction turns the tree into an NFA. |
| 3 | **Simulate** | `match.go` | The engine tracks the set of every active state, advancing all of them on each character. |

<br>

## Supported Syntax

| Syntax | Meaning |
| :--- | :--- |
| `abc` | Literals (full Unicode). |
| `.` | Any character except `\n`. |
| `a\|b` | Alternation. |
| `a*` `a+` `a?` | Zero or more, one or more, zero or one. |
| `( … )` | Grouping. |
| `[a-z_]` `[^0-9]` | Character classes, negated classes. |
| `\d \w \s` | Digit, word, space (and `\D \W \S`). |
| `\n \t \r`, `\.` | Escapes; any punctuation can be escaped. |

> [!NOTE]
> Matching is anchored at both ends. Use `.*(…).*` to search inside a string.

<br>

## Library

```go
re, err := thompson.Compile(`(a|b)*abb`)
re.Match("babaabb")             // true
re.Trace("babaabb")             // every state set, step by step
re.DOT()                        // Graphviz source for the NFA
```

<br>

## CLI

| Flag | Meaning |
| :--- | :--- |
| `-trace` | Print the active state set after every rune. |
| `-dot` | Print the NFA as Graphviz DOT. |

<details open>
<summary><b>Match some strings</b></summary>

```sh
go run ./cmd/thompson '(a|b)*abb' abb abab
```

</details>

<details>
<summary><b>Trace a match</b> (<code>-trace</code>)</summary>

```sh
go run ./cmd/thompson -trace '(a|b)*abb' aabb
```

</details>

<details>
<summary><b>Draw the NFA</b> (<code>-dot</code>, needs Graphviz)</summary>

```sh
go run ./cmd/thompson -dot '(a|b)*abb' | dot -Tsvg > nfa.svg
```

</details>

<br>

## Layout

| File | What it does |
| :--- | :--- |
| `parse.go` | Recursive-descent parser, pattern → syntax tree. |
| `nfa.go` | Thompson's construction, syntax tree → NFA. |
| `match.go` | Set-of-states simulation, plus `Trace` for the visualizer. |
| `graph.go` | NFA as JSON-ready nodes and edges, and DOT. |
| `thompson_test.go` | Unit tests, and 3,000 random patterns diffed against `regexp`. |
| `cmd/thompson/` | The command-line tool. |
| `cmd/wasm/` | Exposes `thompson.compile` and `thompson.trace` to JavaScript. |
| `cmd/bundle/` | Inlines a `web/` page into one standalone HTML file. |

<br>

## Verification & Testing

```sh
go test ./...      # or: make test
```

The suite covers matching, syntax errors, trace shape and pathological patterns, and diffs 3,000 random patterns against Go's `regexp`.
