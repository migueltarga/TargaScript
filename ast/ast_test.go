package ast

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestString(t *testing.T) {
	// Test a simple program: let x = 5;
	program := &Program{
		Statements: []Statement{
			&LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "x"},
					Value: "x",
				},
				Value: &IntegerLiteral{
					Token: token.Token{Type: token.INT, Literal: "5"},
					Value: 5,
				},
			},
		},
	}

	if program.String() != "let x = 5" {
		t.Errorf("program.String() wrong. got=%q", program.String())
	}
}

func TestPostfixExpression(t *testing.T) {
	// Test increment: x++
	increment := &PostfixExpression{
		Token: token.Token{Type: token.INCREMENT, Literal: "++"},
		Left: &Identifier{
			Token: token.Token{Type: token.IDENT, Literal: "x"},
			Value: "x",
		},
		Operator: "++",
	}

	if increment.String() != "(x++)" {
		t.Errorf("increment.String() wrong. got=%q, want=%q", increment.String(), "(x++)")
	}

	// Test decrement: y--
	decrement := &PostfixExpression{
		Token: token.Token{Type: token.DECREMENT, Literal: "--"},
		Left: &Identifier{
			Token: token.Token{Type: token.IDENT, Literal: "y"},
			Value: "y",
		},
		Operator: "--",
	}

	if decrement.String() != "(y--)" {
		t.Errorf("decrement.String() wrong. got=%q, want=%q", decrement.String(), "(y--)")
	}
}

func TestComplexExpression(t *testing.T) {
	// Test a more complex expression: let z = (x++) + (y--);
	program := &Program{
		Statements: []Statement{
			&LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "z"},
					Value: "z",
				},
				Value: &InfixExpression{
					Token: token.Token{Type: token.PLUS, Literal: "+"},
					Left: &PostfixExpression{
						Token: token.Token{Type: token.INCREMENT, Literal: "++"},
						Left: &Identifier{
							Token: token.Token{Type: token.IDENT, Literal: "x"},
							Value: "x",
						},
						Operator: "++",
					},
					Operator: "+",
					Right: &PostfixExpression{
						Token: token.Token{Type: token.DECREMENT, Literal: "--"},
						Left: &Identifier{
							Token: token.Token{Type: token.IDENT, Literal: "y"},
							Value: "y",
						},
						Operator: "--",
					},
				},
			},
		},
	}

	expected := "let z = ((x++) + (y--))"
	if program.String() != expected {
		t.Errorf("program.String() wrong.\ngot=%q\nwant=%q", program.String(), expected)
	}
}
