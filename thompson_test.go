package thompson

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern string
		yes, no []string
	}{
		{"abc", []string{"abc"}, []string{"", "ab", "abcd", "xbc"}},
		{"a|b", []string{"a", "b"}, []string{"", "ab"}},
		{"(a|b)*abb", []string{"abb", "aabb", "babb", "ababb"}, []string{"ab", "abba", ""}},
		{"colou?r", []string{"color", "colour"}, []string{"colouur", "colr"}},
		{"a+", []string{"a", "aaaa"}, []string{""}},
		{"(ab)*", []string{"", "ab", "abab"}, []string{"a", "aba"}},
		{"a.c", []string{"abc", "a c", "aéc"}, []string{"ac", "a\nc"}},
		{"[a-c]+x", []string{"ax", "cbax"}, []string{"dx", "x"}},
		{"[^0-9]*", []string{"", "abc"}, []string{"a1"}},
		{"[]a]", []string{"]", "a"}, []string{"b"}},
		{"[a-]", []string{"a", "-"}, []string{"b"}},
		{`\d+\.\d+`, []string{"3.14", "10.0"}, []string{"3.", ".5", "3x14"}},
		{`\w+@\w+`, []string{"me@site"}, []string{"me@", "a b@c"}},
		{`[\d_]+`, []string{"1_2"}, []string{"a"}},
		{`\D`, []string{"x"}, []string{"7"}},
		{"", []string{""}, []string{"a"}},
		{"a|", []string{"a", ""}, []string{"b"}},
		{"()", []string{""}, []string{"a"}},
		{"(a*)*", []string{"", "aaa"}, []string{"b"}},
		{"(a|b|)+c", []string{"c", "abc"}, []string{"ab"}},
		{"(0|1(01*0)*1)*", []string{"", "0", "11", "110", "1001"}, []string{"1", "10", "100"}}, // multiples of 3
		{"日本+", []string{"日本", "日本本"}, []string{"日"}},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern)
		if err != nil {
			t.Fatalf("Compile(%q): %v", c.pattern, err)
		}
		for _, s := range c.yes {
			if !re.Match(s) {
				t.Errorf("%q should match %q", c.pattern, s)
			}
		}
		for _, s := range c.no {
			if re.Match(s) {
				t.Errorf("%q should not match %q", c.pattern, s)
			}
		}
	}
}

// TestAgainstStdlib generates thousands of random patterns and checks
// every string over {a,b,c} up to length 5 against Go's regexp package.
func TestAgainstStdlib(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	inputs := allStrings("abc", 5)
	for i := 0; i < 3000; i++ {
		pattern := randomPattern(rng, 4)
		std, err := regexp.Compile("^(?:" + pattern + ")$")
		if err != nil {
			continue
		}
		re, err := Compile(pattern)
		if err != nil {
			t.Fatalf("Compile(%q): %v", pattern, err)
		}
		for _, s := range inputs {
			if got, want := re.Match(s), std.MatchString(s); got != want {
				t.Fatalf("pattern %q, input %q: got %v, regexp says %v", pattern, s, got, want)
			}
			if tr := re.Trace(s); tr.Matched != re.Match(s) {
				t.Fatalf("pattern %q, input %q: Trace and Match disagree", pattern, s)
			}
		}
	}
}

func randomPattern(rng *rand.Rand, depth int) string {
	if depth == 0 || rng.Intn(4) == 0 {
		return []string{"a", "b", "c", ".", "[ab]", "[^a]", "[a-b]"}[rng.Intn(7)]
	}
	switch rng.Intn(5) {
	case 0:
		return randomPattern(rng, depth-1) + randomPattern(rng, depth-1)
	case 1:
		return "(" + randomPattern(rng, depth-1) + "|" + randomPattern(rng, depth-1) + ")"
	case 2:
		return "(" + randomPattern(rng, depth-1) + ")" + []string{"*", "+", "?"}[rng.Intn(3)]
	case 3:
		return "(" + randomPattern(rng, depth-1) + ")"
	default:
		return randomPattern(rng, depth-1) + "(|" + randomPattern(rng, depth-1) + ")"
	}
}

func allStrings(alphabet string, maxLen int) []string {
	out := []string{""}
	level := []string{""}
	for n := 0; n < maxLen; n++ {
		var next []string
		for _, p := range level {
			for _, c := range alphabet {
				next = append(next, p+string(c))
			}
		}
		out = append(out, next...)
		level = next
	}
	return out
}

func TestPathological(t *testing.T) {
	n := 40
	pattern := strings.Repeat("a?", n) + strings.Repeat("a", n)
	re := MustCompile(pattern)
	start := time.Now()
	if !re.Match(strings.Repeat("a", n)) {
		t.Fatal("should match")
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("took %v; simulation should be linear", d)
	}
}

func TestSyntaxErrors(t *testing.T) {
	for _, p := range []string{"(", "(a", "a)", "*a", "a|*", "[a", "[z-a]", `a\`, `\q`, "[]", "[^]"} {
		if _, err := Compile(p); err == nil {
			t.Errorf("Compile(%q) should fail", p)
		} else if _, ok := err.(*SyntaxError); !ok {
			t.Errorf("Compile(%q): want *SyntaxError, got %T", p, err)
		}
	}
}

func TestTraceShape(t *testing.T) {
	tr := MustCompile("ab").Trace("ax")
	if len(tr.Steps) != 3 || tr.DeadAt != 1 || tr.Matched {
		t.Fatalf("unexpected trace: %+v", tr)
	}
	if len(tr.Steps[2].Active) != 0 {
		t.Fatalf("set should be empty after 'x'")
	}
}

func TestRenumberIsLeftToRight(t *testing.T) {
	re := MustCompile("abc")
	if re.Start != 0 || re.Accept != len(re.States)-1 {
		t.Fatalf("start=%d accept=%d", re.Start, re.Accept)
	}
	for i, s := range re.States {
		if s.ID != i {
			t.Fatalf("state %d has ID %d", i, s.ID)
		}
	}
}

func BenchmarkPathological(b *testing.B) {
	for _, n := range []int{10, 20, 40} {
		re := MustCompile(strings.Repeat("a?", n) + strings.Repeat("a", n))
		in := strings.Repeat("a", n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				re.Match(in)
			}
		})
	}
}
