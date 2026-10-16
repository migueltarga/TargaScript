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
	ASSIGN    = "="
	EQ        = "=="
	NOT_EQ    = "!="
	PLUS      = "+"
	MINUS     = "-"
	ASTERISK  = "*"
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
	PRINT    = "PRINT"
	REPEAT   = "REPEAT"
	IN       = "IN"
	NULL     = "NULL"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	AWAIT    = "AWAIT"
	TRY      = "TRY"
	CATCH    = "CATCH"
	THROW    = "THROW"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	"fn":       FUNCTION,
	"let":      LET,
	"const":    CONST,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"return":   RETURN,
	"load":     LOAD,
	"as":       AS,
	"print":    PRINT,
	"repeat":   REPEAT,
	"in":       IN,
	"null":     NULL,
	"break":    BREAK,
	"continue": CONTINUE,
	"await":    AWAIT,
	"try":      TRY,
	"catch":    CATCH,
	"throw":    THROW,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
