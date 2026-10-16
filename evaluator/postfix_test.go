package evaluator

import (
	"testing"

	"github.com/migueltarga/TargaScript/object"
)

func TestPostfixOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let x = 1\nx++\nx", "2"},
		{"let x = 1\nx--\nx", "0"},
		{"let x = 1\nx++", "2"},
		{"let f = 1.5\nf++\nf", "2.5"},
		{"let h = {\"count\": 0}\nh.count++\nh.count++\nh.count", "2"},
		{"let a = [5, 6]\na[1]--\na", "[5, 5]"},
		{"let h = {\"k\": 1}\nh[\"k\"]++\nh", `{"k": 2}`},
		{"let n = 0\nrepeat i in 3 { n++ }\nn", "3"},
		{"let n = 10\nlet i = 0\nrepeat i < 3 { i++\nn-- }\nn", "7"},
		{"let x = 1\nx++ + 10", "12"},
		{"let total = 0\nfn bump() { total++ }\nbump()\nbump()\ntotal", "2"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestPostfixErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"const c = 1\nc++", "cannot reassign constant: c"},
		{"missing++", `"missing" is not defined`},
		{"let s = \"a\"\ns++", "operator ++ not supported for STRING"},
		{"let b = true\nb--", "operator -- not supported for BOOLEAN"},
		{"5++", "invalid assignment target: *ast.IntegerLiteral"},
	}

	for _, tt := range tests {
		errObj, ok := testEval(tt.input).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error, got %s", tt.input, testEval(tt.input).Inspect())
			continue
		}
		if errObj.Message != tt.expected {
			t.Errorf("%q: got=%q, want=%q", tt.input, errObj.Message, tt.expected)
		}
	}
}
