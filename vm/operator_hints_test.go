package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestOperatorHints(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`3 + " items"`, `put the number in the string: "${n} items"`},
		{"[1] + 2", "use list.insert(x), or list + [x]"},
		{"null + 1", "give it a default with ??"},
		{"let user = {}; user.age * 2", "give it a default with ??"},
		{"{a: 1} + {b: 2}", "use {...a, ...b}"},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want a hint with %q", tt.input, testEval(tt.input), tt.expected)
		}
	}
	// Other mix-ups keep the plain message
	if err, _ := testEval(`true - 1`).(*object.Error); err == nil || strings.Contains(err.Message, ". ") {
		t.Errorf("true - 1: %v", err)
	}
}
