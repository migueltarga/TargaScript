package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestStringEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"a\tb\nc"`, "a\tb\nc"},
		{`"say \"hi\""`, `say "hi"`},
		{`"back\\slash"`, `back\slash`},
		{`"\d+\s"`, `\d+\s`},
		{`"\${not interpolated}"`, "${not interpolated}"},
		{`"${1 + 1}\w"`, `2\w`},
		{`"${1}\"${2}\""`, `1"2"`},
		{`"\$"`, "$"},
	}
	for _, tt := range tests {
		result := testEval(tt.input)
		s, ok := result.(*object.String)
		if !ok || s.Value != tt.expected {
			t.Errorf("%s: got %q, want %q", tt.input, result.Inspect(), tt.expected)
		}
	}
}
