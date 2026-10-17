package lexer

import (
	"testing"
	"unsafe"

	"github.com/migueltarga/TargaScript/token"
)

// Tokens don't point into the input
func TestTokensOwnTheirText(t *testing.T) {
	input := `let count = 12; count += 3.5; print("${count} items")`
	start := uintptr(unsafe.Pointer(unsafe.StringData(input)))
	end := start + uintptr(len(input))
	l := New(input)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if tok.Literal == "" {
			continue
		}
		at := uintptr(unsafe.Pointer(unsafe.StringData(tok.Literal)))
		if at >= start && at < end {
			t.Errorf("%s %q points into the input", tok.Type, tok.Literal)
		}
	}
}
