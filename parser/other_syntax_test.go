package parser

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
)

func TestHintsForOtherLanguagesSyntax(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"for i in 1..3 { print(i) }", "TargaScript loops with repeat: repeat item in list"},
		{"for (let i = 0; i < 3; i++) { }", "TargaScript loops with repeat"},
		{"while true { }", "repeat condition { ... }"},
		{"while (x < 3) { }", "repeat condition { ... }"},
		{"function add(a, b) { a + b }", "functions are declared with fn"},
		{"def add(a, b): return a + b", "functions are declared with fn"},
		{"var x = 5", "variables are declared with let"},
		{`print "hi"`, "calls need parentheses: print(...)"},
		{"if a === b { }", "TargaScript compares with =="},
		{"if a !== b { }", "TargaScript compares with !="},
	}
	for _, tt := range tests {
		p := New(lexer.New(tt.input))
		p.ParseProgram()
		if len(p.Errors()) == 0 || !strings.Contains(p.Errors()[0], tt.expected) {
			t.Errorf("%q: errors %v, want the first to contain %q", tt.input, p.Errors(), tt.expected)
		}
	}
}

func TestOtherLanguagesWordsAsNames(t *testing.T) {
	// The words are only hints at the start of a statement that looks
	// like another language; as names they still work
	for _, input := range []string{
		"let def = 1\ndef",
		"let var = fn(x) { x }\nvar(2)",
		"let while = 3\nwhile + 1",
		"let for = [1]\nfor.length",
		"let print = 1\nprint",
		"x\n\"next line\"",
		"a == b",
	} {
		p := New(lexer.New(input))
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Errorf("%q: unexpected errors %v", input, p.Errors())
		}
	}
}
