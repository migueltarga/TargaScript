package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/token"
)

// tokens lists the type and text of every token, which formatting must
// keep: it only changes spaces between them.
func tokens(source string) []token.Token {
	l := lexer.New(source)
	list := []token.Token{}
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		list = append(list, token.Token{Type: tok.Type, Literal: tok.Literal})
	}
	return list
}

// FuzzSource checks that formatting never changes a program's tokens, and
// that formatting twice changes nothing more.
func FuzzSource(f *testing.F) {
	files, _ := filepath.Glob("../examples/*.tg")
	more, _ := filepath.Glob("../tests/*.tg")
	for _, file := range append(files, more...) {
		if source, err := os.ReadFile(file); err == nil {
			f.Add(string(source))
		}
	}
	for _, seed := range []string{
		"fn f() {\nlet s = `a\n  b ${ {x: \"}\"}.x } c`\n}", "\"a\\\"b\n{\" {", "/* {\n} */ x", "a\r\nb", "`${`${1}`}`",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		once := Source(source)
		before, after := tokens(source), tokens(once)
		if len(before) != len(after) {
			t.Fatalf("formatting changed the number of tokens from %d to %d\n%q\n%q", len(before), len(after), source, once)
		}
		for i := range before {
			if before[i] != after[i] {
				t.Fatalf("formatting changed token %d from %v to %v\n%q\n%q", i, before[i], after[i], source, once)
			}
		}
		if twice := Source(once); twice != once {
			t.Fatalf("formatting twice changed it\n%q\n%q", once, twice)
		}
	})
}
