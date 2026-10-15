package lexer

import (
	"strings"
	"unicode/utf8"

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
		column: 0,
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
	case ';':
		tok = l.newToken(token.SEMICOLON, string(l.ch), tokenLine, tokenColumn)
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.INCREMENT, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.PLUS, string(l.ch), tokenLine, tokenColumn)
		}
	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.DECREMENT, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.MINUS, string(l.ch), tokenLine, tokenColumn)
		}
	case '*':
		tok = l.newToken(token.ASTERISK, string(l.ch), tokenLine, tokenColumn)
	case '%':
		tok = l.newToken(token.MODULO, string(l.ch), tokenLine, tokenColumn)
	case '^':
		tok = l.newToken(token.XOR, string(l.ch), tokenLine, tokenColumn)
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
		start := l.position + 1
		literal, interpolated, terminated := l.readString()
		switch {
		case !terminated:
			tok = l.newToken(token.ILLEGAL, "\""+literal, tokenLine, tokenColumn)
		case interpolated:
			tok = l.newToken(token.TEMPLATE, l.input[start:l.position], tokenLine, tokenColumn)
		default:
			tok = l.newToken(token.STRING, literal, tokenLine, tokenColumn)
		}
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
			if l.isStartingWithDigitFollowedByLetter() {
				invalidIdent := l.readInvalidIdentifier()
				tok = l.newToken(token.ILLEGAL, invalidIdent, tokenLine, tokenColumn)
				return tok
			}
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
		l.column = 0
	}

	width := 1
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch, width = utf8.DecodeRuneInString(l.input[l.readPosition:])
	}
	l.position = l.readPosition
	l.readPosition += width

	if l.ch != 0 && l.ch != '\n' {
		l.column++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || (isDigit(l.ch) && l.position > position) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) isStartingWithDigitFollowedByLetter() bool {
	pos := l.position
	readPos := l.readPosition
	ch := l.ch

	for isDigit(l.ch) {
		l.readChar()
	}

	isInvalidIdent := isLetter(l.ch)

	l.position = pos
	l.readPosition = readPos
	l.ch = ch

	return isInvalidIdent
}

func (l *Lexer) readInvalidIdentifier() string {
	position := l.position
	l.readChar()
	for isLetter(l.ch) || isDigit(l.ch) {
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

// readString reads a string literal, processing escapes. It reports whether
// the string contains ${...} interpolations, which are skipped over without
// being processed, and whether its closing quote was found.
func (l *Lexer) readString() (string, bool, bool) {
	interpolated := false
	escaped := false
	var result strings.Builder

	for {
		l.readChar()

		if l.ch == 0 {
			return result.String(), interpolated, false
		}

		if escaped {
			// Handle escaped characters
			switch l.ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case 'r':
				result.WriteRune('\r')
			case '"':
				result.WriteRune('"')
			case '\\':
				result.WriteRune('\\')
			default:
				result.WriteRune(l.ch)
			}
			escaped = false
		} else if l.ch == '\\' {
			escaped = true
		} else if l.ch == '"' {
			break
		} else if l.ch == '$' && l.peekChar() == '{' {
			interpolated = true
			l.readChar()
			if !l.skipInterpolation() {
				return result.String(), interpolated, false
			}
		} else {
			result.WriteRune(l.ch)
		}
	}

	return result.String(), interpolated, true
}

// skipInterpolation advances from the "{" of "${" to its matching "}",
// skipping nested braces and strings. Reports false if the input ends first.
func (l *Lexer) skipInterpolation() bool {
	depth := 1
	for depth > 0 {
		l.readChar()
		switch l.ch {
		case 0:
			return false
		case '{':
			depth++
		case '}':
			depth--
		case '"':
			for {
				l.readChar()
				if l.ch == 0 {
					return false
				}
				if l.ch == '\\' {
					l.readChar()
				} else if l.ch == '"' {
					break
				}
			}
		}
	}
	return true
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
