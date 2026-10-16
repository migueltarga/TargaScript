package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestCompareArrays(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"[1, 2] < [1, 3]", "true"},
		{"[1, 2] > [1, 3]", "false"},
		{"[2] > [1, 9]", "true"},
		{"[1] < [1, 0]", "true"},
		{"[1, 0] <= [1, 0]", "true"},
		{"[1, 0] < [1, 0]", "false"},
		{`["b", 1] > ["a", 2]`, "true"},
		{"[[1, 2]] < [[1, 3]]", "true"},
		{"[1.5] < [2]", "true"},
		{"[[2, \"b\"], [1, \"z\"], [2, \"a\"]].sort()", `[[1, "z"], [2, "a"], [2, "b"]]`},
		{`[{n: "Bia", c: "Rio"}, {n: "Ana", c: "SP"}, {n: "Caio", c: "Rio"}].sortBy(p => [p.c, p.n]).map(p => p.n)`, `["Bia", "Caio", "Ana"]`},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	err, ok := testEval(`[1] < ["a"]`).(*object.Error)
	if !ok || !strings.Contains(err.Message, `cannot use "<" with an integer and a string`) {
		t.Errorf("mixed elements: got %v", err)
	}
}
