package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"
	BOOL   = "BOOL"

	// TEMPLATE is a string containing ${...}. Its literal is the raw source
	// between the quotes, escapes included.
	TEMPLATE = "TEMPLATE"

	// Operators
	ASSIGN   = "="
	EQ       = "=="
	NOT_EQ   = "!="
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	POWER    = "**"

	// OP_ASSIGN is a compound assignment, like += or ??=
	OP_ASSIGN = "OP="
	SLASH     = "/"
	BANG      = "!"
	LT        = "<"
	GT        = ">"
	LT_EQ     = "<="
	GT_EQ     = ">="
	AND       = "&&"
	OR        = "||"
	RANGE     = ".."
	SPREAD    = "..."
	DOT       = "."
	INCREMENT = "++"
	DECREMENT = "--"
	MODULO    = "%"
	XOR       = "^"
	NULLISH   = "??"
	PIPE      = "|>"
	OPTIONAL  = "?."

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	CONST    = "CONST"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	LOAD     = "LOAD"
	AS       = "AS"
	REPEAT   = "REPEAT"
	IN       = "IN"
	NULL     = "NULL"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	AWAIT    = "AWAIT"
	TRY      = "TRY"
	CATCH    = "CATCH"
	FINALLY  = "FINALLY"
	THROW    = "THROW"
	MATCH    = "MATCH"
	ARROW    = "=>"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// keyword is the token type of a keyword.
func keyword(k string) (TokenType, bool) {
	switch k {
	case "fn":
		return FUNCTION, true
	case "let":
		return LET, true
	case "const":
		return CONST, true
	case "true":
		return TRUE, true
	case "false":
		return FALSE, true
	case "if":
		return IF, true
	case "else":
		return ELSE, true
	case "return":
		return RETURN, true
	case "load":
		return LOAD, true
	case "as":
		return AS, true
	case "repeat":
		return REPEAT, true
	case "in":
		return IN, true
	case "null":
		return NULL, true
	case "break":
		return BREAK, true
	case "continue":
		return CONTINUE, true
	case "await":
		return AWAIT, true
	case "try":
		return TRY, true
	case "catch":
		return CATCH, true
	case "finally":
		return FINALLY, true
	case "throw":
		return THROW, true
	case "match":
		return MATCH, true
	}
	return "", false
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keyword(ident); ok {
		return tok
	}
	return IDENT
}

// IsKeyword reports whether a token is a keyword, like repeat or in.
func IsKeyword(tok Token) bool {
	kind, ok := keyword(tok.Literal)
	return ok && kind == tok.Type
}
