package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunFmt(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"messy.tg": "fn f() {\nreturn 1   \n}\n\n\n",
		"tidy.tg":  "fn f() {\n    return 1\n}\n",
	})
	if code := runFmt([]string{"--check", dir}); code != 1 {
		t.Errorf("--check with a messy file: exit code %d, want 1", code)
	}
	if source, _ := os.ReadFile(filepath.Join(dir, "messy.tg")); string(source) != "fn f() {\nreturn 1   \n}\n\n\n" {
		t.Error("--check changed the file")
	}

	if code := runFmt([]string{dir}); code != 0 {
		t.Errorf("fmt: exit code %d", code)
	}
	if source, _ := os.ReadFile(filepath.Join(dir, "messy.tg")); string(source) != "fn f() {\n    return 1\n}\n" {
		t.Errorf("formatted to %q", source)
	}
	if code := runFmt([]string{"--check", dir}); code != 0 {
		t.Errorf("--check after fmt: exit code %d, want 0", code)
	}
}
