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
	// A first line like #!/usr/bin/env targascript lets the system run
	// the file as a program
	if l.ch == '#' && l.peekChar() == '!' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
	}
	return l
}

// NewAt returns a lexer for code that starts at a line and column of a
// larger source, like the code inside ${...} in a string, so its tokens
// have positions in that source.
func NewAt(input string, line, column int) *Lexer {
	l := &Lexer{input: input, line: line, column: column - 1}
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
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = l.newToken(token.PIPE, "|>", tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch), tokenLine, tokenColumn)
		}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.EQ, string(ch)+string(l.ch), tokenLine, tokenColumn)
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = l.newToken(token.ARROW, "=>", tokenLine, tokenColumn)
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
		if l.peekChar() == '*' {
			l.readChar()
			tok = l.newToken(token.POWER, "**", tokenLine, tokenColumn)
		} else {
			tok = l.newToken(token.ASTERISK, string(l.ch), tokenLine, tokenColumn)
		}
	case '%':
		tok = l.newToken(token.MODULO, string(l.ch), tokenLine, tokenColumn)
	case '?':
		switch l.peekChar() {
		case '?':
			l.readChar()
			tok = l.newToken(token.NULLISH, "??", tokenLine, tokenColumn)
		case '.':
			l.readChar()
			tok = l.newToken(token.OPTIONAL, "?.", tokenLine, tokenColumn)
		default:
			tok = l.newToken(token.ILLEGAL, string(l.ch), tokenLine, tokenColumn)
		}
	case '^':
		tok = l.newToken(token.XOR, string(l.ch), tokenLine, tokenColumn)
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			if l.peekChar() == '.' {
				l.readChar()
				tok = l.newToken(token.SPREAD, "...", tokenLine, tokenColumn)
			} else {
				tok = l.newToken(token.RANGE, "..", tokenLine, tokenColumn)
			}
		} else {
			tok = l.newToken(token.DOT, string(l.ch), tokenLine, tokenColumn)
		}
	case '/':
		if l.peekChar() == '/' {
			l.skipLineComment()
			return l.NextToken()
		}
		if l.peekChar() == '*' {
			if !l.skipBlockComment() {
				return l.newToken(token.ILLEGAL, "/*", tokenLine, tokenColumn)
			}
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
	case '`':
		literal, template, interpolated, terminated := l.readRawString()
		switch {
		case !terminated:
			tok = l.newToken(token.ILLEGAL, "`"+literal, tokenLine, tokenColumn)
		case interpolated:
			tok = l.newToken(token.TEMPLATE, template, tokenLine, tokenColumn)
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

	// x += 1 and the like: an operator followed by =
	if compoundable[tok.Type] && l.peekChar() == '=' {
		l.readChar()
		tok = l.newToken(token.OP_ASSIGN, tok.Literal+"=", tokenLine, tokenColumn)
	}

	l.readChar()
	return tok
}

// compoundable are the operators that form compound assignments, like +=.
var compoundable = map[token.TokenType]bool{
	token.PLUS: true, token.MINUS: true, token.ASTERISK: true, token.SLASH: true,
	token.MODULO: true, token.POWER: true, token.NULLISH: true,
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

// skipBlockComment skips a /* ... */ comment and reports whether its closing
// */ was found.
func (l *Lexer) skipBlockComment() bool {
	l.readChar()
	l.readChar()
	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return true
		}
		l.readChar()
	}
	return false
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
	column := l.column

	if l.ch == '0' && strings.ContainsRune("xXbBoO", l.peekChar()) {
		return false
	}
	for isDigit(l.ch) || (l.ch == '_' && isDigit(l.peekChar())) {
		l.readChar()
	}

	isInvalidIdent := isLetter(l.ch) && !l.atExponent()

	l.position = pos
	l.readPosition = readPos
	l.ch = ch
	l.column = column

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

	// 0x1F, 0b1010 and 0o17
	if l.ch == '0' && strings.ContainsRune("xXbBoO", l.peekChar()) {
		l.readChar()
		l.readChar()
		for isHexDigit(l.ch) || (l.ch == '_' && isHexDigit(l.peekChar())) {
			l.readChar()
		}
		return token.INT, l.input[start:l.position]
	}

	// Underscores separate digits: 1_000_000
	for isDigit(l.ch) || (l.ch == '.' && !hasDot) || (l.ch == '_' && isDigit(l.peekChar())) {
		if l.ch == '.' {
			if l.peekChar() == '.' {
				break
			}
			hasDot = true
		}
		l.readChar()
	}
	hasExponent := l.atExponent()
	if hasExponent {
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	lit := l.input[start:l.position]
	if hasDot || hasExponent {
		return token.FLOAT, lit
	}
	return token.INT, lit
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

// atExponent reports whether the lexer is at the exponent of a number, an e
// or E followed by digits with an optional sign, as in 1e3 or 2.5E-4.
func (l *Lexer) atExponent() bool {
	if l.ch != 'e' && l.ch != 'E' {
		return false
	}
	rest := l.input[l.readPosition:]
	if len(rest) > 0 && (rest[0] == '+' || rest[0] == '-') {
		rest = rest[1:]
	}
	return len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9'
}

// readString reads a string literal, processing escapes. It reports whether
// the string contains ${...} interpolations, which are skipped over without
// being processed, and whether its closing quote was found.
// readRawString reads a string in backticks, which can span lines and
// keeps backslashes as written: only \` and \${ stand for ` and ${. It
// returns the text, and for one with ${...} the same string written the
// way a template in double quotes would be, which the parser reads.
func (l *Lexer) readRawString() (text, template string, interpolated, terminated bool) {
	var value, quoted strings.Builder
	for {
		l.readChar()
		switch {
		case l.ch == 0:
			return value.String(), "", interpolated, false
		case l.ch == '`':
			return value.String(), quoted.String(), interpolated, true
		case l.ch == '\\' && l.peekChar() == '`':
			l.readChar()
			value.WriteRune('`')
			quoted.WriteRune('`')
		case l.ch == '\\' && l.peekChar() == '$':
			l.readChar()
			value.WriteRune('$')
			quoted.WriteString("\\$")
		case l.ch == '\\' || l.ch == '"':
			value.WriteRune(l.ch)
			quoted.WriteRune('\\')
			quoted.WriteRune(l.ch)
		case l.ch == '$' && l.peekChar() == '{':
			interpolated = true
			start := l.position
			l.readChar()
			if !l.skipInterpolation() {
				return value.String(), "", interpolated, false
			}
			quoted.WriteString(l.input[start : l.position+1])
		default:
			value.WriteRune(l.ch)
			quoted.WriteRune(l.ch)
		}
	}
}

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
			case '$':
				// \${ is a literal ${
				result.WriteRune('$')
			default:
				// Other backslashes stay, so patterns like "\d+" read as written
				result.WriteRune('\\')
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
