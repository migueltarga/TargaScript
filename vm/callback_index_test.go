package vm

import (
	"bytes"
	"os"
	"testing"

	"github.com/migueltarga/TargaScript/evaluator"
)

func TestCallbacksGetTheIndex(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`["a", "b"].map((x, i) => "${i}:${x}")`, `["0:a", "1:b"]`},
		{"[10, 20, 30].filter((x, i) => i != 1)", "[10, 30]"},
		{"[5, 6, 7].reduce((acc, x, i) => acc + x * i, 0)", "20"},
		{"[5, 6].find((x, i) => i == 1)", "6"},
		{"[1, 1, 1].some((x, i) => i == 2)", "true"},
		{"[1, 1].every((x, i) => i < 2)", "true"},
		{"[1, 1, 1].count((x, i) => i > 0)", "2"},
		{"[3, 1, 2].sortBy((x, i) => -i)", "[2, 1, 3]"},
		{`["a", "b", "c"].groupBy((x, i) => i % 2)`, `{0: ["a", "c"], 1: ["b"]}`},
		// One parameter still gets only the element
		{"[1, 2].map(x => x * 10)", "[10, 20]"},
		{"[1, 2].map(fn(...xs) { xs })", "[[1, 0], [2, 1]]"},
		// The arguments of one call don't change those of another
		{"[[1, 2], [3]].map(a => a.map((x, i) => x + i))", "[[1, 3], [3]]"},
		{"let seen = []\n[3, 1, 2].sort((a, b) => { seen.insert([a, b])\na - b })\nseen.every(p => p.length == 2)", "true"},
		{"[1, 2, 3].reduce((acc, x) => acc + [1, 2].reduce((a, y) => a + x * y, 0), 0)", "18"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	// Builtins get only the element, so map(print) prints the elements
	var out bytes.Buffer
	evaluator.SetOutput(&out)
	defer evaluator.SetOutput(os.Stdout)
	testEval("[7, 8].map(print)")
	if out.String() != "7\n8\n" {
		t.Errorf("map(print) printed %q", out.String())
	}
}
