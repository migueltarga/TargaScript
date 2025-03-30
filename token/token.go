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
	DOT       = "."
	INCREMENT = "++"
	DECREMENT = "--"

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
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	"fn":     FUNCTION,
	"let":    LET,
	"const":  CONST,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"load":   LOAD,
	"as":     AS,
	"print":  PRINT,
	"repeat": REPEAT,
	"in":     IN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
