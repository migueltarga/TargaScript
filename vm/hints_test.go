package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestHintsForOtherLanguages(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`len([1])`, `"len" is not defined. In TargaScript, use .length`},
		{`parseInt("1")`, "use toInt(value)"},
		{`console.log(1)`, "use print(...)"},
		{`let x = None`, "use null"},
		{`[1].push(2)`, `arrays have no method "push". In TargaScript, use insert(value)`},
		{`[1].includes(1)`, "use contains(value) or the in operator"},
		{`"a".toUpperCase()`, `strings have no method "toUpperCase". In TargaScript, use upper()`},
		{`{a: 1}.hasOwnProperty("a")`, `use has(key) or "key" in object`},
		// Typos still get the closest name, now from every method
		{`[1].sortby(x => x)`, `Did you mean "sortBy"?`},
		{`"a".padstart(2)`, `Did you mean "padStart"?`},
		{`{a: 1}.entris()`, `Did you mean "entries"?`},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		err, ok := got.(*object.Error)
		if !ok || !strings.Contains(err.Message, tt.expected) {
			t.Errorf("%s: got %v, want an error with %q", tt.input, got, tt.expected)
		}
	}
}

func TestOwnNamesWinOverHints(t *testing.T) {
	if got := testEval("let len = fn(x) { 7 }; len([1])").Inspect(); got != "7" {
		t.Errorf("got %s, want 7", got)
	}
}
