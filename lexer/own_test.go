package lexer

import (
	"testing"
	"unsafe"

	"github.com/migueltarga/TargaScript/token"
)

// Tokens don't point into the input, and a name is made once
func TestTokensOwnTheirText(t *testing.T) {
	input := `let count = 12; count += 3.5; print("${count} items")`
	start := uintptr(unsafe.Pointer(unsafe.StringData(input)))
	end := start + uintptr(len(input))
	l := New(input)
	var counts []string
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		if tok.Literal == "" {
			continue
		}
		at := uintptr(unsafe.Pointer(unsafe.StringData(tok.Literal)))
		if at >= start && at < end {
			t.Errorf("%s %q points into the input", tok.Type, tok.Literal)
		}
		if tok.Literal == "count" {
			counts = append(counts, tok.Literal)
		}
	}
	if len(counts) != 2 || unsafe.StringData(counts[0]) != unsafe.StringData(counts[1]) {
		t.Errorf("count is made more than once")
	}
}
