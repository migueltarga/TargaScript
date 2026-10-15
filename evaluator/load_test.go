package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

// setupModules writes files into a temporary directory used as the main
// program's directory, and clears modules loaded by previous tests.
func setupModules(t *testing.T, files map[string]string) {
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

	loadedModules = map[string]*object.Environment{}
	loadingModules = map[string]bool{}
	SetMainDir(dir)
	t.Cleanup(func() { SetMainDir("") })
}

func TestLoad(t *testing.T) {
	setupModules(t, map[string]string{
		"lib/math.tg": "fn square(x) { x * x }\nconst PI = 3\nlet loads = 0",
		"lib/geo.tg":  "load \"math.tg\"\nfn area(r) { PI * square(r) }\nfn later() {\n  load \"math.tg\" as m\n  return m.square(4)\n}",
		"counter.tg":  "let count = 0\nfn inc() { count = count + 1\nreturn count }",
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"load \"lib/math.tg\"\nsquare(3)", "9"},
		{"load \"lib/math.tg\"\nPI", "3"},
		{"load \"lib/math.tg\" as m\nm.square(5) + m.PI", "28"},
		{"load \"lib/math.tg\" as m\nm.keys()", "[PI, loads, square]"},
		// Paths inside a module are relative to that module's file
		{"load \"lib/geo.tg\" as geo\ngeo.area(2)", "12"},
		{"load \"lib/geo.tg\" as geo\ngeo.later()", "16"},
		// A module runs once and keeps its state between loads
		{"load \"counter.tg\" as a\nload \"counter.tg\" as b\na.inc()\nb.inc()", "2"},
		{"load \"lib/math.tg\";\nsquare(2)", "4"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	setupModules(t, map[string]string{
		"math.tg":   "const PI = 3",
		"a.tg":      "load \"b.tg\"",
		"b.tg":      "load \"a.tg\"",
		"broken.tg": "let x = 1\nprint(missing)",
		"syntax.tg": "let = 1",
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"load \"math.tg\"\nPI = 4", "cannot reassign constant: PI"},
		{"const PI = 1\nload \"math.tg\"", "it defines PI, which is a constant here"},
		{"load \"nope.tg\"", `cannot load "nope.tg": file not found`},
		{"load \"a.tg\"", `circular load of "a.tg"`},
		{"load \"broken.tg\"", "broken.tg:2:7: identifier not found: missing"},
		{"load \"syntax.tg\"", `cannot load "syntax.tg": line 1:5`},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error", tt.input)
			continue
		}
		if !strings.Contains(errObj.Message, tt.expected) {
			t.Errorf("%q: error %q does not contain %q", tt.input, errObj.Message, tt.expected)
		}
	}
}
