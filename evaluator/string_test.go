package evaluator

import (
	"testing"
)

func TestStringLength(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`"hello".length`, 5},
		{`"".length`, 0},
		{`"héllo".length`, 5},
		{`"olá 🚀".length`, 5},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(tt.input), tt.expected, tt.input)
	}
}
