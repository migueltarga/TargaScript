package lexer

import (
	"github.com/migueltarga/TargaScript/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	l.skipWhitespace()

	// Store current position information for the token
	tokenLine := l.line
	tokenColumn := l.column

	switch l.ch {
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.AND, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch), tokenLine, tokenColumn)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.OR, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch), tokenLine, tokenColumn)
		}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.EQ, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.ASSIGN, string(l.ch), tokenLine, tokenColumn)
		}
	case ':':
		tok = l.newToken(token.COLON, string(l.ch), tokenLine, tokenColumn)
	case '+':
		tok = l.newToken(token.PLUS, string(l.ch), tokenLine, tokenColumn)
	case '-':
		tok = l.newToken(token.MINUS, string(l.ch), tokenLine, tokenColumn)
	case '*':
		tok = l.newToken(token.ASTERISK, string(l.ch), tokenLine, tokenColumn)
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok = l.newToken(token.RANGE, "..", tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.DOT, string(l.ch), tokenLine, tokenColumn)
		}
	case '/':
		if l.peekChar() == '/' {
			l.skipLineComment()
			return l.NextToken()
		}
		tok = l.newToken(token.SLASH, string(l.ch), tokenLine, tokenColumn)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.NOT_EQ, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.BANG, string(l.ch), tokenLine, tokenColumn)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.LT_EQ, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.LT, string(l.ch), tokenLine, tokenColumn)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.GT_EQ, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.GT, string(l.ch), tokenLine, tokenColumn)
		}
	case ',':
		tok = l.newToken(token.COMMA, string(l.ch), tokenLine, tokenColumn)
	case '{':
		tok = l.newToken(token.LBRACE, string(l.ch), tokenLine, tokenColumn)
	case '}':
		tok = l.newToken(token.RBRACE, string(l.ch), tokenLine, tokenColumn)
	case '(':
		tok = l.newToken(token.LPAREN, string(l.ch), tokenLine, tokenColumn)
	case ')':
		tok = l.newToken(token.RPAREN, string(l.ch), tokenLine, tokenColumn)
	case '[':
		tok = l.newToken(token.LBRACKET, string(l.ch), tokenLine, tokenColumn)
	case ']':
		tok = l.newToken(token.RBRACKET, string(l.ch), tokenLine, tokenColumn)
	case '"':
		literal := l.readString()
		tok = l.newToken(token.STRING, literal, tokenLine, tokenColumn)
		l.readChar()
		return tok
	case 0:
		tok = l.newToken(token.EOF, "", tokenLine, tokenColumn)
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			tokenType := token.LookupIdent(literal)
			tok = l.newToken(tokenType, literal, tokenLine, tokenColumn)
			return tok
		} else if isDigit(l.ch) {
			tokenType, literal := l.readNumber()
			tok = l.newToken(tokenType, literal, tokenLine, tokenColumn)
			return tok
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch), tokenLine, tokenColumn)
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
	if l.ch == '\n' {
		l.line++
		l.column = 1
	}

	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		r := []rune(l.input[l.readPosition:])[0]
		l.ch = r
	}
	l.position = l.readPosition

	if l.ch != 0 {
		l.readPosition += len(string(l.ch))
		if l.ch != '\n' {
			l.column++
		}
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

		if l.ch == 0 {
			break
		}
		if l.ch == '"' {
			break
		}
		if l.ch == '\\' {
			if l.peekChar() == '"' {
				l.readChar()
			}
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

func (l *Lexer) newToken(tokenType token.TokenType, literal string, line int, column int) token.Token {
	return token.Token{
		Type:    tokenType,
		Literal: literal,
		Line:    line,
		Column:  column,
	}
}
