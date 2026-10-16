package compiler

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

// Top-level variables only the program's own code uses become its locals
func TestLocalizeTopLevel(t *testing.T) {
	tests := []struct {
		source string
		local  map[string]bool
	}{
		{"let a = 1\na += 2\nprint(a)", map[string]bool{"a": true}},
		{"const a = 1\nlet [b, c] = [a, 2]", map[string]bool{"a": true, "b": true, "c": true}},
		// Functions use globals
		{"let a = 1\nfn f() { a }", map[string]bool{"a": false, "f": false}},
		{"let a = 1\nlet g = () => a + 1", map[string]bool{"a": false, "g": true}},
		// Used or assigned before its declaration
		{"print(a)\nlet a = 1", map[string]bool{"a": false}},
		{"a = 5\nlet a = 1", map[string]bool{"a": false}},
		{"let a = a", map[string]bool{"a": false}},
	}
	for _, tt := range tests {
		program := parser.New(lexer.New(tt.source)).ParseProgram()
		c := New()
		c.LocalizeTopLevel(program)
		for name, want := range tt.local {
			if got := c.mainLocals[name]; got != want {
				t.Errorf("%q: %s local %v, want %v", tt.source, name, got, want)
			}
		}
		if _, err := c.Compile(program); err != nil {
			t.Errorf("%q: %v", tt.source, err)
		}
	}
}
