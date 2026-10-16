package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestEvalMixedNumberExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1 + 2.5", 3.5},
		{"2.5 + 1", 3.5},
		{"10 - 0.5", 9.5},
		{"3 * 1.5", 4.5},
		{"7 / 2.0", 3.5},
		{"5.5 % 2.0", 1.5},
		{"7.0 % 0.5", 0},
		{"7 % 2.5", 2},
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

func TestEvalMixedNumberComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1 == 1.0", true},
		{"1 < 1.5", true},
		{"2.5 >= 3", false},
	}

	for _, tt := range tests {
		testBooleanObject(t, testEval(tt.input), tt.expected)
	}
}
