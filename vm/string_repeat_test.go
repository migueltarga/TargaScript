package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestStringRepeat(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"ab" * 3`, "ababab"},
		{`3 * "ab"`, "ababab"},
		{`"ab" * 0`, ""},
		{`"" * 5`, ""},
		{`let line = "-"; line *= 4; line`, "----"},
	}
	for _, tt := range tests {
		str, ok := testEval(tt.input).(*object.String)
		if !ok || str.Value != tt.expected {
			t.Errorf("%s: got %v, want %q", tt.input, testEval(tt.input), tt.expected)
		}
	}
}

func TestStringRepeatErrors(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"ab" * -1`, "negative number of times"},
		{`"ab" * 9999999999`, "too long"},
	}
	for _, tt := range tests {
		err, ok := testEval(tt.input).(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, testEval(tt.input), tt.expected)
		}
	}
}
