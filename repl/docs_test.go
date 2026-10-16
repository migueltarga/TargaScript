package repl

import (
	"bytes"
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

// TestCookbookExamples runs the code blocks marked targascript in
// docs/COOKBOOK.md, in a directory of their own, and compares what they
// print with the output block after them.
func TestCookbookExamples(t *testing.T) {
	doc, err := os.ReadFile("../docs/COOKBOOK.md")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	blocks := regexp.MustCompile("(?s)```targascript\n(.*?)```(?:\\s*```output\n(.*?)```)?").FindAllStringSubmatch(string(doc), -1)
	if len(blocks) == 0 {
		t.Fatal("no examples found")
	}
	for i, block := range blocks {
		var out bytes.Buffer
		if err := EvalSource(block[1], &out, true, false, false); err != nil {
			t.Errorf("example %d failed: %v\n%s", i+1, err, block[1])
			continue
		}
		if block[2] != "" && out.String() != block[2] {
			t.Errorf("example %d printed:\n%s\nwant:\n%s", i+1, out.String(), block[2])
		}
	}
}
