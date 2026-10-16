package evaluator

import (
	"bytes"
	"os"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// typedLines is a System with lines to read, as if a user typed them.
type typedLines struct {
	fakeFiles
	lines []string
}

func (t *typedLines) ReadLine() (string, bool) {
	if len(t.lines) == 0 {
		return "", false
	}
	line := t.lines[0]
	t.lines = t.lines[1:]
	return line, true
}

func TestInput(t *testing.T) {
	var out bytes.Buffer
	SetOutput(&out)
	defer SetOutput(os.Stdout)
	SetSystem(&typedLines{lines: []string{"Ana", ""}})
	defer SetSystem(nil)

	for _, want := range []object.Object{&object.String{Value: "Ana"}, &object.String{Value: ""}, NULL} {
		got := inputBuiltin(&object.String{Value: "? "})
		if got.Inspect() != want.Inspect() || got.Type() != want.Type() {
			t.Errorf("got %s, want %s", got.Inspect(), want.Inspect())
		}
	}
	if out.String() != "? ? ? " {
		t.Errorf("printed %q", out.String())
	}

	// Without a terminal, as in the browser
	SetSystem(fakeFiles{})
	if err, ok := inputBuiltin().(*object.Error); !ok || err.Message != "input is only available in the command line" {
		t.Errorf("got %s", inputBuiltin().Inspect())
	}
}
