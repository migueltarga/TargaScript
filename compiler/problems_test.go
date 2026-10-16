package compiler

import (
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/parser"
)

func TestProblems(t *testing.T) {
	source := `let total = 1
print(totl)
fn f() {
  let counter = 0
  fn inc() { counter++ }
  missing()
  break
}
const max = 3
max = 4
print(len([1]))`

	c := New()
	if _, err := c.Compile(parser.New(lexer.New(source)).ParseProgram()); err != nil {
		t.Fatal(err)
	}

	expected := []Problem{
		{2, 7, `"totl" is not defined. Did you mean "total"?`},
		{6, 3, `"missing" is not defined`},
		{7, 3, "break outside of a loop"},
		{10, 5, "cannot reassign constant: max"},
		{11, 7, `"len" is not defined`},
	}
	problems := c.Problems()
	if len(problems) != len(expected) {
		t.Fatalf("got %d problems: %+v", len(problems), problems)
	}
	for i, p := range problems {
		if p != expected[i] {
			t.Errorf("problem %d: got %+v, want %+v", i, p, expected[i])
		}
	}

	// A new Compile starts over
	c.Compile(parser.New(lexer.New("let ok = 1")).ParseProgram())
	if len(c.Problems()) != 0 {
		t.Errorf("problems from the previous program: %+v", c.Problems())
	}
}
