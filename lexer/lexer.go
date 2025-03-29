package lexer

import (
	"github.com/migueltarga/TargaScript/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	l.skipWhitespace()

	switch l.ch {
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case ':':
		tok = newToken(token.COLON, l.ch)
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '*':
		tok = newToken(token.ASTERISK, l.ch)
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok = token.Token{Type: token.RANGE, Literal: ".."}
		} else {
			tok = newToken(token.DOT, l.ch)
		}
	case '/':
		if l.peekChar() == '/' {
			l.skipLineComment()
			return l.NextToken()
		}
		tok = newToken(token.SLASH, l.ch)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LT_EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GT_EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(token.GT, l.ch)
		}
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case 0:
		tok = token.Token{Type: token.EOF, Literal: ""}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		r := []rune(l.input[l.readPosition:])[0]
		l.ch = r
	}
	l.position = l.readPosition

	if l.ch != 0 {
		l.readPosition += len(string(l.ch))
	} else {
		l.readPosition += 1
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return []rune(l.input[l.readPosition:])[0]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || (isDigit(l.ch) && l.position > position) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (token.TokenType, string) {
	start := l.position
	hasDot := false

	for isDigit(l.ch) || (l.ch == '.' && !hasDot) {
		if l.ch == '.' {
			if l.peekChar() == '.' {
				break
			}
			hasDot = true
		}
		l.readChar()
	}
	lit := l.input[start:l.position]
	if hasDot {
		return token.FLOAT, lit
	}
	return token.INT, lit
}

func (l *Lexer) readString() string {
	start := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[start:l.position]
}

func isLetter(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_' ||
		(ch >= 0x80 && ch <= 0x10FFFF)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch rune) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}
