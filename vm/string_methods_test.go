package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestMoreStringMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"ab".repeat(3)`, "ababab"},
		{`"ab".repeat(0)`, ""},
		{`"7".padStart(3, "0")`, "007"},
		{`"7".padStart(3)`, "  7"},
		{`"abc".padStart(2)`, "abc"},
		{`"ab".padEnd(6, "xy")`, "abxyxy"},
		{`"ab".padEnd(5, "xy")`, "abxyx"},
		{`"héllo".reverse()`, "olléh"},
		{`"héllo".slice(1, 3)`, "él"},
		{`"hello".slice(-3)`, "llo"},
		{`"hello".slice(10)`, ""},
		{`"héy".chars()`, `["h", "é", "y"]`},
		{`"a\nb\r\nc\n".lines()`, `["a", "b", "c"]`},
		{`"".lines()`, "[]"},
		{`"  hi  ".trimStart() + "|"`, "hi  |"},
		{`"|" + "  hi  ".trimEnd()`, "|  hi"},
		{`"banana".count("an")`, "2"},
		{`"élan".capitalize()`, "Élan"},
		{`"".capitalize()`, ""},
		{`"ab".repeat(-1)`, "ERROR: string.repeat() needs a count of 0 or more"},
		{`"ab".repeat(100000000)`, "ERROR: string.repeat() would make a string too long"},
		{`"ab".padStart(5, "")`, "ERROR: fill for string.padStart() must be a non-empty string"},
		{`"ab".slice("1")`, "ERROR: arguments to string.slice() must be integers, got a string"},
		{`"ab".count("")`, "ERROR: string.count() needs a non-empty string to count"},
		{`"ab".revers()`, `ERROR: strings have no method "revers". Did you mean "reverse"?`},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		got := result.Inspect()
		if s, ok := result.(*object.String); ok {
			got = s.Value
		}
		if err, ok := result.(*object.Error); ok {
			got = "ERROR: " + err.Message
		}
		if got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
