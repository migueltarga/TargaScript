package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestMoreMath(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"PI", "3.141592653589793"},
		{"E", "2.718281828459045"},
		{"round(sin(PI / 2), 6)", "1"},
		{"cos(0)", "1"},
		{"round(atan2(1, 1) * 4, 6)", "3.141593"},
		{"log(E)", "1"},
		{"log10(1000)", "3"},
		{"log2(8)", "3"},
		{"exp(0)", "1"},
		{"hypot(3, 4)", "5"},
		{"sign(-2.5)", "-1"},
		{"sign(0)", "0"},
		{"sign(7)", "1"},
		{"clamp(15, 0, 10)", "10"},
		{"clamp(-3, 0, 10)", "0"},
		{"clamp(5, 0, 10)", "5"},
		{"trunc(-2.7)", "-2"},
		// A program's own PI comes first
		{"let PI = 3; PI", "3"},
		{"fn f() { let E = 1; E }; [f(), E]", "[1, 2.718281828459045]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	for input, expected := range map[string]string{
		`sin("a")`:    "arguments to `sin` must be numbers, got a string",
		"hypot(1)":    "wrong number of arguments for hypot",
		"clamp(1, 2)": "want=3 (value, min, max)",
	} {
		err, ok := testEval(input).(*object.Error)
		if !ok || !strings.Contains(err.Message, expected) {
			t.Errorf("%s: got %v, want an error with %q", input, testEval(input), expected)
		}
	}
}
