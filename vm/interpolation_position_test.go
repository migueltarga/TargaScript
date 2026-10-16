package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestErrorsInsideInterpolationsHaveTheirPosition(t *testing.T) {
	tests := []struct {
		input        string
		line, column int
	}{
		{"let a = null\n\nprint(\"value: ${a.x}\")", 3, 18},
		{"let a = null\nprint(\"one\\ttwo ${a.x}\")", 2, 20},
		{"let a = null\nprint(\"line one\nline two ${a.x}\")", 3, 13},
		{"let a = null\nprint(\"${1} and ${a.x:.2f}\")", 2, 20},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok || err.Line != tt.line || err.Column != tt.column {
			t.Errorf("%q: got %v, want line %d, column %d", tt.input, testEval(tt.input), tt.line, tt.column)
		}
	}
}
