package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestValuesThatContainThemselves(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let a = [1]; a.insert(a); a", "[1, [...]]"},
		{`let o = {name: "x"}; o.self = o; o`, `{"name": "x", "self": {...}}`},
		{`let o = {}; let a = [o]; o.list = a; a`, `[{"list": [...]}]`},
		{"let a = [1]; a.insert(a); \"${a}\"", "[1, [...]]"},
		{"let x = [1]; [x, x]", "[[1], [1]]"},
		{"let a = []; a.insert(a); a == a", "true"},
		{"let a = []; a.insert(a); a.insert(a); let b = []; b.insert(b); b.insert(b); a == b", "false"},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		inspected := got.Inspect()
		if str, ok := got.(*object.String); ok {
			inspected = str.Value
		}
		if inspected != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, inspected, tt.expected)
		}
	}

	err, ok := testEval("let o = {}; o.self = o; toJSON(o)").(*object.Error)
	if !ok || !strings.Contains(err.Message, "contains itself") {
		t.Errorf("toJSON of a value inside itself: got %v", err)
	}
}
