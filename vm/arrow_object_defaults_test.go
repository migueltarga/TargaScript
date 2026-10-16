package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestArrowObjectPatternDefaults(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`let f = ({host, port = 80}) => "${host}:${port}"; f({host: "a"})`, "a:80"},
		{`let f = ({host, port = 80}) => "${host}:${port}"; f({host: "a", port: 1})`, "a:1"},
		{`[{n: 1}, {}].map(({n = 0}) => n)`, "[1, 0]"},
		{`let f = ({a = 1, ...rest}) => [a, rest]; f({b: 2})`, `[1, {"b": 2}]`},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}

	err, ok := testEval("let o = {a = 1}").(*object.Error)
	if !ok || !strings.Contains(err.Message, "in an object, write {a: ...}") {
		t.Errorf("{a = 1} as an object: got %v", err)
	}
}
