// Command thompson matches strings against a pattern from the shell.
//
//	thompson '(a|b)*abb' abb abab      match each argument
//	thompson -trace '(a|b)*abb' aabb   print the state set after each rune
//	thompson -dot '(a|b)*abb' | dot -Tsvg > nfa.svg
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/swayam5342/thompson"
)

func main() {
	dot := flag.Bool("dot", false, "print the NFA as Graphviz DOT")
	trace := flag.Bool("trace", false, "print the active state set after every rune")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: thompson [-dot] [-trace] pattern [input ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	re, err := thompson.Compile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *dot {
		fmt.Print(re.DOT())
		return
	}

	failed := false
	for _, in := range flag.Args()[1:] {
		if *trace {
			t := re.Trace(in)
			fmt.Printf("%q against %q\n", re.Pattern, in)
			for _, s := range t.Steps {
				read := "start"
				if s.Pos >= 0 {
					read = fmt.Sprintf("read %-3q", s.Rune)
				}
				fmt.Printf("  %-10s {%s}\n", read, join(s.Active))
			}
			fmt.Printf("  => %s\n\n", verdict(t.Matched))
			failed = failed || !t.Matched
			continue
		}
		ok := re.Match(in)
		fmt.Printf("%-8s %q\n", verdict(ok), in)
		failed = failed || !ok
	}
	if failed {
		os.Exit(1)
	}
}

func verdict(ok bool) string {
	if ok {
		return "match"
	}
	return "no match"
}

func join(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	return strings.Join(parts, " ")
}
