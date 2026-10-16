package repl

import (
	"io"
	"os"
	"regexp"
	"testing"
)

// TestReferenceExamples runs every code block marked targascript in
// docs/REFERENCE.md, so the reference can't fall out of date.
func TestReferenceExamples(t *testing.T) {
	doc, err := os.ReadFile("../docs/REFERENCE.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := regexp.MustCompile("(?s)```targascript\n(.*?)```").FindAllStringSubmatch(string(doc), -1)
	if len(blocks) == 0 {
		t.Fatal("no examples found")
	}
	for i, block := range blocks {
		if err := EvalSource(block[1], io.Discard, true, false, false); err != nil {
			t.Errorf("example %d failed: %v\n%s", i+1, err, block[1])
		}
	}
}
