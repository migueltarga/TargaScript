package vm

import "testing"

func TestSpread(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let a = [1, 2]\nlet b = [3]\n[...a, ...b, 4]", "[1, 2, 3, 4]"},
		{"let a = [1, 2]\nlet copy = [...a]\ncopy.insert(3)\n[a, copy]", "[[1, 2], [1, 2, 3]]"},
		{`[..."abc"]`, `["a", "b", "c"]`},
		{"[0, ...[], 1]", "[0, 1]"},
		{"fn add(a, b, c) { a + b + c }\nlet args = [1, 2, 3]\nadd(...args)", "6"},
		{"fn add(a, b, c) { a + b + c }\nadd(1, ...[2, 3])", "6"},
		{"max(...[3, 9])", "9"},
		{`"a-b".split("-").join(...["+"])`, "a+b"},
		{`let defaults = {"host": "localhost", "port": 80}` + "\n{...defaults, \"port\": 8080}", `{"host": "localhost", "port": 8080}`},
		{`let a = {"x": 1}` + "\nlet b = {\"y\": 2}\n{...a, ...b, \"z\": 3}", `{"x": 1, "y": 2, "z": 3}`},
		{`let a = {"x": 1}` + "\nlet b = {...a}\nb.x = 9\n[a.x, b.x]", "[1, 9]"},
		{"[...5]", "ERROR at line 1, column 2: cannot spread an integer into a list, only arrays and strings"},
		{`{...[1]}`, "ERROR at line 1, column 2: cannot spread an array into an object, only objects"},
		{"let x = ...[1]", "ERROR at line 1, column 9: ... can only be used in arrays, objects and the arguments of calls"},
		{"fn f(...) { }", "ERROR: parser errors: line 1:6: expected a parameter name, found \"...\""},
		{"let a = null\n[...(a ?? [])]", "[]"},
		{"let parts = [[1], [2, 3]]\n[...parts[1], ...parts[0]]", "[2, 3, 1]"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
