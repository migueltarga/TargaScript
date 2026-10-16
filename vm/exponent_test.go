package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestEvalExponentLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1e3", 1000},
		{"2.5e-1", 0.25},
		{"1E+2 + 1", 101},
		{"-3e2", -300},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		result, ok := evaluated.(*object.Float)
		if !ok {
			t.Errorf("%s: object is not Float. got=%T (%+v)", tt.input, evaluated, evaluated)
			continue
		}
		if result.Value != tt.expected {
			t.Errorf("%s: got=%g, want=%g", tt.input, result.Value, tt.expected)
		}
	}
}
