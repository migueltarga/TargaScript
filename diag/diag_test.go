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
