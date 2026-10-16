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

func TestStartReadsUnfinishedCode(t *testing.T) {
	in := strings.NewReader(strings.Join([]string{
		"fn add(a, b) {",
		"  a + b",
		"}",
		"let total = add(1,",
		"  2)",
		`let s = "multi`,
		`line"`,
		"total + 1 +",
		"1",
		"let broken = [1, 2",
		"",
		"s.length",
	}, "\n") + "\n")
	var out bytes.Buffer

	Start(in, &out, true, false)

	output := out.String()
	for _, want := range []string{"5\n", "10\n", `expected "]", but the file ended`} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
	if !strings.Contains(output, CONTINUE) {
		t.Errorf("never showed the continuation prompt:\n%s", output)
	}
	if strings.Contains(output, "never closed") {
		t.Errorf("ran the function before it was finished:\n%s", output)
	}
}
