package vm

import (
	"math"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestIntegerOverflowErrors(t *testing.T) {
	const max = "9223372036854775807"
	const min = "(-9223372036854775807 - 1)"

	tests := []string{
		max + " + 1",
		min + " - 1",
		max + " * 2",
		min + " * -1",
		"-1 * " + min,
		min + " / -1",
		"-" + min,
		"let x = " + max + "\nx++",
		"let x = " + min + "\nx--",
		"abs(" + min + ")",
		"toInt(10000000000000000000.0)",
		"round(-10000000000000000000.0)",
		"floor(" + max + ".0)",
		"ceil(pow(10, 300))",
	}

	for _, input := range tests {
		errObj, ok := testEval(input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error, got %s", input, testEval(input).Inspect())
			continue
		}
		if errObj.Message != "integer overflow" {
			t.Errorf("%q: wrong error message. got=%q", input, errObj.Message)
		}
	}
}

func TestIntegerArithmeticNearLimits(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"9223372036854775806 + 1", math.MaxInt64},
		{"(-9223372036854775807 - 1) + 1", math.MinInt64 + 1},
		{"-9223372036854775807 - 1", math.MinInt64},
		{"4611686018427387903 * 2", math.MaxInt64 - 1},
		{"(-9223372036854775807 - 1) / 1", math.MinInt64},
		{"(-9223372036854775807 - 1) % -1", 0},
		{"-5 * -3", 15},
		{"0 * (-9223372036854775807 - 1)", 0},
		{"toInt(-9200000000000000000.0)", -9200000000000000000},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected)
	}
}
