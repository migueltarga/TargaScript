package diag

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	source := "let a = 1\nlet x = 1 +\n"
	got := Format(source, Diagnostic{Line: 2, Column: 12, Message: `expected a value after "+"`})
	expected := strings.Join([]string{
		`Error: expected a value after "+"`,
		`  --> line 2, column 12`,
		``,
		` 2 | let x = 1 +`,
		`   |            ^`,
	}, "\n")
	if got != expected {
		t.Errorf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestFormatKeepsTabsAndWideLineNumbers(t *testing.T) {
	source := strings.Repeat("\n", 9) + "\tprint(é, x)"
	got := Format(source, Diagnostic{Line: 10, Column: 11, Message: "oops"})
	if !strings.HasSuffix(got, " 10 | \tprint(é, x)\n    | \t         ^") {
		t.Errorf("got:\n%q", got)
	}
}

func TestFormatWithoutPosition(t *testing.T) {
	if got := Format("x", Diagnostic{Message: "boom"}); got != "Error: boom" {
		t.Errorf("got %q", got)
	}
	if got := Format("x", Diagnostic{Line: 5, Column: 2, Message: "boom"}); got != "Error: boom (line 5, column 2)" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAllLimitsErrors(t *testing.T) {
	diagnostics := []Diagnostic{{Message: "a"}, {Message: "b"}, {Message: "c"}, {Message: "d"}, {Message: "e"}}
	got := FormatAll("", diagnostics)
	if !strings.HasSuffix(got, "Error: c\n\n...and 2 more errors") || strings.Contains(got, "Error: d") {
		t.Errorf("got:\n%s", got)
	}
}

func TestFormatWithTrace(t *testing.T) {
	source := "fn f() {\n  1 / 0\n}\nf()"
	got := Format(source, Diagnostic{Line: 2, Column: 5, Message: "division by zero",
		Trace: []Call{{"f", 2}, {"", 4}}})
	want := "Error: division by zero\n  --> line 2, column 5\n\n 2 |   1 / 0\n   |     ^\n\n  in f (line 2)\n  called from line 4"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	// An error at the top level has nothing to trace
	if top := Format("1 / 0", Diagnostic{Line: 1, Column: 3, Message: "x", Trace: []Call{{"", 1}}}); strings.Contains(top, "called") || strings.Contains(top, " in ") {
		t.Errorf("top-level error shows a trace:\n%s", top)
	}
}

func TestLongTraces(t *testing.T) {
	// Runaway recursion: the same call thousands of times is one line
	trace := []Call{}
	for range 5000 {
		trace = append(trace, Call{"f", 1})
	}
	trace = append(trace, Call{"", 2})
	got := Format("fn f() { f() }\nf()", Diagnostic{Line: 1, Column: 10, Message: "too deep", Trace: trace})
	if !strings.HasSuffix(got, "\n\n  in f (line 1), 5000 times\n  called from line 2") {
		t.Errorf("got:\n%s", got)
	}

	// Alternating calls keep the innermost and the outermost
	trace = []Call{}
	for i := range 40 {
		trace = append(trace, Call{[]string{"a", "b"}[i%2], i%2 + 1})
	}
	trace = append(trace, Call{"", 3})
	got = Format("x\ny\nz", Diagnostic{Line: 1, Column: 1, Message: "deep", Trace: trace})
	lines := strings.Count(got, "called from") + strings.Count(got, "\n  in ")
	if lines != 11 || !strings.Contains(got, "... 30 more calls") || !strings.HasSuffix(got, "called from line 3") {
		t.Errorf("%d calls shown in:\n%s", lines, got)
	}
}
