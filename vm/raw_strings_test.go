package vm

import (
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestBacktickStrings(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"`plain`", "plain"},
		{"`two\nlines`", "two\nlines"},
		{"`say \"hi\"`", `say "hi"`},
		{"`\\d+\\n`", `\d+\n`},
		{"`a \\` b`", "a ` b"},
		{"let n = 2; `${n} + ${n} = ${n + n}`", "2 + 2 = 4"},
		{"`\\${kept}`", "${kept}"},
		// A backslash before a $ that doesn't start ${ is text, as in a shell prompt
		{"`PS1='\\w \\$ '`", `PS1='\w \$ '`},
		{"let n = 1; `${n} \\$`", `1 \$`},
		{"let who = \"Ana\"; `<p class=\"x\">${who}</p>`", `<p class="x">Ana</p>`},
		{"`${\"in quotes\"}`", "in quotes"},
		{"`${`nested`}`", "nested"},
		{"parseJSON(`{\"a\": [1, \"x\\\"y\"]}`).a[1]", `x"y`},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		str, ok := got.(*object.String)
		if !ok || str.Value != tt.expected {
			t.Errorf("%q: got %s, want %q", tt.input, got.Inspect(), tt.expected)
		}
	}

	err, ok := testEval("`never closed").(*object.Error)
	if !ok || !strings.Contains(err.Message, "this string is never closed, add a ` at its end") {
		t.Errorf("unclosed backtick string: got %v", err)
	}
}
