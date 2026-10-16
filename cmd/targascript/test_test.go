package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/migueltarga/TargaScript/evaluator"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestTestFiles(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a_test.tg":         "",
		"helper.tg":         "",
		"sub/b_test.tg":     "",
		".hidden/c_test.tg": "",
	})
	got, err := testFiles([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "a_test.tg"), filepath.Join(dir, "sub/b_test.tg")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunTestsExitCode(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"passing", map[string]string{"a_test.tg": `test("ok", fn() { assert(true) })`}, 0},
		{"failing", map[string]string{"a_test.tg": `test("no", fn() { assert(false) })`}, 1},
		{"error outside a test", map[string]string{"a_test.tg": "test(\"ok\", fn() { assert(true) })\nnope()"}, 1},
		{"no tests", map[string]string{"a_test.tg": `print("hi")`}, 1},
		{"no files", map[string]string{"a.tg": ""}, 1},
	}
	for _, tt := range tests {
		evaluator.ResetTests()
		if got := runTests([]string{writeFiles(t, tt.files)}, true); got != tt.want {
			t.Errorf("%s: exit code %d, want %d", tt.name, got, tt.want)
		}
	}
	evaluator.ResetTests()
}
