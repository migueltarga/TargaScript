package repl

import (
	"bytes"
	"testing"
)

func TestArgs(t *testing.T) {
	SetArgs([]string{"a", "b c"})
	defer SetArgs([]string{})

	var out bytes.Buffer
	if err := EvalSource(`print(args.length, args[1])`, &out, true, false, false); err != nil {
		t.Fatal(err)
	}
	if out.String() != "2 b c\n" {
		t.Errorf("got %q", out.String())
	}
}
