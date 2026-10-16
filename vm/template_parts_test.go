package vm

import (
	"strings"
	"testing"
)

// Templates show every kind of value like print does
func TestTemplateParts(t *testing.T) {
	long := strings.Repeat("é", 50)
	tests := []struct{ input, expected string }{
		{`let i = -42` + "\n" + `"n=${i}!"`, "n=-42!"},
		{`let f = 1.5` + "\n" + `"${f} ${true} ${false} ${null}"`, "1.5 true false null"},
		{`let xs = [1, "a"]` + "\n" + `"${xs} ${ {a: 1} }"`, `[1, "a"] {"a": 1}`},
		{`let s = "` + long + `"` + "\n" + `"<${s}>${s}"`, "<" + long + ">" + long},
		{`let n = 9223372036854775807` + "\n" + `"${n}${n}"`, "92233720368547758079223372036854775807"},
		{`""`, ""},
		{`let a = "x"` + "\n" + `"${a}"`, "x"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %q, want %q", tt.input, got, tt.expected)
		}
	}
}
