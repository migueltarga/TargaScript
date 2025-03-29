package lexer

import (
	"testing"

	"github.com/migueltarga/TargaScript/token"
)

func TestNextToken(t *testing.T) {
	input := `load math
load fs as file

let name = "Targa"
const version = 1.0
let isCool = true

fn greet(person) {
  print("Hello, " + person)
}

fn greet2(person) {
  return "Hello, " + person + "! 😀"
}

if name == "Targa" {
  greet(name)
} else {
  print("Who are you?")
}

fn add(a, b) {
  return a + b
}

let result = add(5, 10)
print("5 + 10 = " + result)


let app = {
  name: name,
  version: version,
  greeting: greet(name)
}

repeat i in 1..3 {
  print("number: " + i)
}

`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LOAD, "load"},
		{token.IDENT, "math"},
		{token.LOAD, "load"},
		{token.IDENT, "fs"},
		{token.AS, "as"},
		{token.IDENT, "file"},
		{token.LET, "let"},
		{token.IDENT, "name"},
		{token.ASSIGN, "="},
		{token.STRING, "Targa"},
		{token.CONST, "const"},
		{token.IDENT, "version"},
		{token.ASSIGN, "="},
		{token.FLOAT, "1.0"},
		{token.LET, "let"},
		{token.IDENT, "isCool"},
		{token.ASSIGN, "="},
		{token.TRUE, "true"},
		{token.FUNCTION, "fn"},
		{token.IDENT, "greet"},
		{token.LPAREN, "("},
		{token.IDENT, "person"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.PRINT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "Hello, "},
		{token.PLUS, "+"},
		{token.IDENT, "person"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.FUNCTION, "fn"},
		{token.IDENT, "greet2"},
		{token.LPAREN, "("},
		{token.IDENT, "person"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.STRING, "Hello, "},
		{token.PLUS, "+"},
		{token.IDENT, "person"},
		{token.PLUS, "+"},
		{token.STRING, "! 😀"},
		{token.RBRACE, "}"},
		{token.IF, "if"},
		{token.IDENT, "name"},
		{token.EQ, "=="},
		{token.STRING, "Targa"},
		{token.LBRACE, "{"},
		{token.IDENT, "greet"},
		{token.LPAREN, "("},
		{token.IDENT, "name"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.PRINT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "Who are you?"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.FUNCTION, "fn"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "a"},
		{token.COMMA, ","},
		{token.IDENT, "b"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.IDENT, "a"},
		{token.PLUS, "+"},
		{token.IDENT, "b"},
		{token.RBRACE, "}"},
		{token.LET, "let"},
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.INT, "5"},
		{token.COMMA, ","},
		{token.INT, "10"},
		{token.RPAREN, ")"},
		{token.PRINT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "5 + 10 = "},
		{token.PLUS, "+"},
		{token.IDENT, "result"},
		{token.RPAREN, ")"},
		{token.LET, "let"},
		{token.IDENT, "app"},
		{token.ASSIGN, "="},
		{token.LBRACE, "{"},
		{token.IDENT, "name"},
		{token.COLON, ":"},
		{token.IDENT, "name"},
		{token.COMMA, ","},
		{token.IDENT, "version"},
		{token.COLON, ":"},
		{token.IDENT, "version"},
		{token.COMMA, ","},
		{token.IDENT, "greeting"},
		{token.COLON, ":"},
		{token.IDENT, "greet"},
		{token.LPAREN, "("},
		{token.IDENT, "name"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.REPEAT, "repeat"},
		{token.IDENT, "i"},
		{token.IN, "in"},
		{token.INT, "1"},
		{token.RANGE, ".."},
		{token.INT, "3"},
		{token.LBRACE, "{"},
		{token.PRINT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "number: "},
		{token.PLUS, "+"},
		{token.IDENT, "i"},
		{token.RPAREN, ")"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		// Add debug print statement to see tokens during test execution
		t.Logf("Token %d: Type=%q, Literal=%q", i, tok.Type, tok.Literal)

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
