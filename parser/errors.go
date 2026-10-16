package parser

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/token"
)

// Error is a syntax error at a position in the source.
type Error struct {
	Line    int
	Column  int
	Message string
}

func (e Error) String() string {
	return fmt.Sprintf("line %d:%d: %s", e.Line, e.Column, e.Message)
}

// Errors returns the syntax errors as "line L:C: message" strings.
func (p *Parser) Errors() []string {
	messages := make([]string, len(p.errors))
	for i, err := range p.errors {
		messages[i] = err.String()
	}
	return messages
}

// Diagnostics returns the syntax errors with their positions.
func (p *Parser) Diagnostics() []Error {
	return p.errors
}

func (p *Parser) errorAt(line, column int, format string, a ...interface{}) {
	p.errors = append(p.errors, Error{Line: line, Column: column, Message: fmt.Sprintf(format, a...)})
}

// peekError reports that the next token isn't the expected one.
func (p *Parser) peekError(t token.TokenType) {
	if p.peekTokenIs(token.EOF) {
		line, column := p.afterCurrent()
		p.errorAt(line, column, "expected %s, but the file ended", describeType(t))
		return
	}
	p.errorAt(p.peekToken.Line, p.peekToken.Column, "expected %s, found %s", describeType(t), describe(p.peekToken))
}

// noPrefixParseFnError reports a token that can't start an expression.
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	cur := p.curToken

	if t == token.ILLEGAL {
		switch {
		case strings.HasPrefix(cur.Literal, "\""):
			p.errorAt(cur.Line, cur.Column, "this string is never closed, add a \" at its end")
		case cur.Literal == "/*":
			p.errorAt(cur.Line, cur.Column, "this comment is never closed, add */ at its end")
		default:
			p.errorAt(cur.Line, cur.Column, "unexpected %q", cur.Literal)
		}
		return
	}

	if expectsValue(p.prevToken) {
		// Point right after the operator when the problem is on a later
		// line, as in `let x = 1 +` followed by a new line
		if t == token.EOF || cur.Line != p.prevToken.Line {
			line, column := p.afterPrevious()
			p.errorAt(line, column, "expected a value after %q", p.prevToken.Literal)
			return
		}
		p.errorAt(cur.Line, cur.Column, "expected a value after %q, found %s", p.prevToken.Literal, describe(cur))
		return
	}

	if t == token.EOF {
		p.errorAt(cur.Line, cur.Column, "unexpected end of file")
		return
	}
	p.errorAt(cur.Line, cur.Column, "unexpected %s", describe(cur))
}

// afterCurrent and afterPrevious return the position right after the
// current or previous token.
func (p *Parser) afterCurrent() (int, int) {
	return p.curToken.Line, p.curToken.Column + utf8.RuneCountInString(p.curToken.Literal)
}

func (p *Parser) afterPrevious() (int, int) {
	return p.prevToken.Line, p.prevToken.Column + utf8.RuneCountInString(p.prevToken.Literal)
}

// expectsValue reports whether a value must follow tok, as after an
// operator, a comma or an opening bracket.
func expectsValue(tok token.Token) bool {
	switch tok.Type {
	case token.ASSIGN, token.EQ, token.NOT_EQ, token.PLUS, token.MINUS, token.ASTERISK,
		token.SLASH, token.BANG, token.LT, token.GT, token.LT_EQ, token.GT_EQ,
		token.AND, token.OR, token.RANGE, token.MODULO, token.XOR,
		token.COMMA, token.COLON, token.LPAREN, token.LBRACKET, token.RETURN:
		return true
	}
	return false
}

// describe names a token for an error message.
func describe(tok token.Token) string {
	switch tok.Type {
	case token.EOF:
		return "the end of the file"
	case token.STRING, token.TEMPLATE:
		return "a string"
	}
	return fmt.Sprintf("%q", tok.Literal)
}

// describeType names a kind of token for an error message.
func describeType(t token.TokenType) string {
	switch t {
	case token.IDENT:
		return "a name"
	case token.STRING:
		return "a string"
	case token.INT, token.FLOAT:
		return "a number"
	case token.EOF:
		return "the end of the file"
	case token.FUNCTION:
		return `"fn"`
	}
	if literal, ok := keywordLiterals[t]; ok {
		return fmt.Sprintf("%q", literal)
	}
	return fmt.Sprintf("%q", string(t))
}

var keywordLiterals = map[token.TokenType]string{
	token.LET: "let", token.CONST: "const", token.TRUE: "true", token.FALSE: "false",
	token.IF: "if", token.ELSE: "else", token.RETURN: "return", token.LOAD: "load",
	token.AS: "as", token.PRINT: "print", token.REPEAT: "repeat", token.IN: "in",
	token.NULL: "null", token.BREAK: "break", token.CONTINUE: "continue",
}
