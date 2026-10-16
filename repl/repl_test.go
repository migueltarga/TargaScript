package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartKeepsStateBetweenLines(t *testing.T) {
	in := strings.NewReader("let x = 41\nfn inc(n) { n + 1 }\ninc(x)\n")
	var out bytes.Buffer

	Start(in, &out, true, false)

	if !strings.Contains(out.String(), "42") {
		t.Fatalf("expected output to contain 42, got %q", out.String())
	}
	if strings.Contains(out.String(), "is not defined") {
		t.Fatalf("variables were lost between lines: %q", out.String())
	}
}
