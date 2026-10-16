package vm

import "testing"

func TestStructuralEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2] == [1, 2]", "true"},
		{"[1, 2] == [2, 1]", "false"},
		{"[1, [2, 3]] == [1, [2, 3]]", "true"},
		{"[1] == [1.0]", "true"},
		{`{"a": 1, "b": [2]} == {"b": [2], "a": 1}`, "true"},
		{`{"a": 1} == {"a": 2}`, "false"},
		{`{"a": 1} == {"a": 1, "b": 2}`, "false"},
		{"[] != []", "false"},
		{`[1, "x"] != [1, "y"]`, "true"},
		{"[null, true] == [null, true]", "true"},
		{"[1] == 1", "false"},
		{`"1" == 1`, "false"},
		{"fn f() { 1 }\nfn g() { 1 }\nf == g", "false"},
		{"fn f() { 1 }\nf == f", "true"},
		{"let a = []\na.insert(a)\na == a", "true"},
		{"let a = []\na.insert(a)\nlet b = []\nb.insert(b)\na == b", "false"},
		{"[[1, 2], [3]].contains([3])", "true"},
		{`[{"id": 1}, {"id": 2}].indexOf({"id": 2})`, "1"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
