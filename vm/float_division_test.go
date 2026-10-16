package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestFloatDivisionByZero(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0 / 0", "Inf"},
		{"-1 / 0.0", "-Inf"},
		{"0.0 / 0", "NaN"},
		{"5.5 % 0.0", "NaN"},
		{"1.0 / 0 > 1000000", "true"},
		{"let n = 0.0 / 0\nn == n", "false"},
		{`"result: ${1.0 / 0}"`, "result: Inf"},
		{`toString(-1.0 / 0)`, "-Inf"},
		{`"x" + 1.0 / 0`, "xInf"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if isError(evaluated) {
			t.Errorf("%q: unexpected error %s", tt.input, evaluated.Inspect())
			continue
		}
		got := evaluated.Inspect()
		if s, ok := evaluated.(*object.String); ok {
			got = s.Value
		}
		if got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestIntegerDivisionByZeroIsAnError(t *testing.T) {
	tests := map[string]string{
		"1 / 0": "division by zero",
		"1 % 0": "modulo by zero",
	}

	for input, expected := range tests {
		errObj, ok := testEval(input).(*object.Error)
		if !ok || errObj.Message != expected {
			t.Errorf("%q: expected error %q, got %s", input, expected, testEval(input).Inspect())
		}
	}
}
