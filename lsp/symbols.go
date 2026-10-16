package lsp

import (
	"strconv"
	"strings"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/token"
)

// declaration is a name a file declares with fn, let or const.
type declaration struct {
	name         string
	kind         int    // the LSP symbol kind: 12 function, 13 variable, 14 constant
	line, column int    // 0-based, of the name
	blocks       string // the blocks it's in, like "/1/4", from the outermost
}

func (d declaration) topLevel() bool { return d.blocks == "" }

// scan reads the tokens of a file, calling found for each declaration and
// returning the blocks open at a 0-based line and column. It reads tokens,
// so it works while the file has syntax errors too.
func scan(source string, line, column int, found func(declaration)) string {
	l := lexer.New(source)
	open := []string{}
	next := 0
	at := ""
	reached := false
	previous := token.Token{}
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if !reached && (tok.Line-1 > line || tok.Line-1 == line && tok.Column-1 >= column) {
			at, reached = strings.Join(open, ""), true
		}
		switch tok.Type {
		case token.LBRACE:
			next++
			open = append(open, "/"+strconv.Itoa(next))
		case token.RBRACE:
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
		case token.IDENT:
			kind := map[token.TokenType]int{token.FUNCTION: 12, token.LET: 13, token.CONST: 14}[previous.Type]
			if kind != 0 && found != nil {
				found(declaration{tok.Literal, kind, tok.Line - 1, tok.Column - 1, strings.Join(open, "")})
			}
		}
		previous = tok
	}
	if !reached {
		at = strings.Join(open, "")
	}
	return at
}

// declarations finds the names a file declares, in order.
func declarations(source string) []declaration {
	found := []declaration{}
	scan(source, -1, 0, func(d declaration) { found = append(found, d) })
	return found
}

// definition returns where name is declared for a use at line and column:
// among the declarations whose block is open there, the last one before
// it, or else the first one after, as functions can be called before
// they're declared.
func definition(source, name string, line, column int) (declaration, bool) {
	all := []declaration{}
	blocks := scan(source, line, column, func(d declaration) {
		if d.name == name {
			all = append(all, d)
		}
	})
	var best declaration
	found := false
	for _, d := range all {
		// Visible where its block, or a block inside it, is open
		if !strings.HasPrefix(blocks+"/", d.blocks+"/") {
			continue
		}
		before := d.line < line || d.line == line && d.column < column
		if before || !found {
			best, found = d, true
		}
	}
	return best, found
}
