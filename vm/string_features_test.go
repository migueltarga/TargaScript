package vm

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestStringIndexing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"[0]`, "h"},
		{`"hello"[4]`, "o"},
		{`"olá"[2]`, "á"},
		{`"🚀x"[1]`, "x"},
		{`let s = "abc"
s[s.length - 1]`, "c"},
		{`"abc"[3]`, "null"},
		{`"abc"[-1]`, "null"},
		{`""[0]`, "null"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestRepeatOverString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let out = []
repeat c in "abc" { out = out.insert(c) }
out`, `["a", "b", "c"]`},
		{`let n = 0
repeat c in "olá 🚀" { n = n + 1 }
n`, "5"},
		{`let out = ""
repeat c in "a-b-c" {
  if c == "-" { continue }
  out = out + c
}
out`, "abc"},
		{`let n = 0
repeat c in "" { n = n + 1 }
n`, "0"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestStringSearchMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello".contains("ell")`, "true"},
		{`"hello".contains("xyz")`, "false"},
		{`"hello".contains("")`, "true"},
		{`"hello".startsWith("he")`, "true"},
		{`"hello".startsWith("lo")`, "false"},
		{`"hello".endsWith("lo")`, "true"},
		{`"hello".indexOf("l")`, "2"},
		{`"hello".indexOf("z")`, "-1"},
		{`"olá mundo".indexOf("m")`, "4"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}

	errorTests := []struct {
		input    string
		expected string
	}{
		{`"a".contains(1)`, "argument to string.contains() must be a string, got an integer"},
		{`"a".indexOf()`, "wrong number of arguments for string.indexOf(): got 0, want 1"},
		{`"abc"["x"]`, `cannot use [] on a string`},
	}

	for _, tt := range errorTests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error", tt.input)
			continue
		}
		if errObj.Message != tt.expected {
			t.Errorf("%q: got=%q, want=%q", tt.input, errObj.Message, tt.expected)
		}
	}
}
