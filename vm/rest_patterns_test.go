package vm

import "testing"

func TestRestInPatterns(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let [first, ...rest] = [1, 2, 3]\n[first, rest]", "[1, [2, 3]]"},
		{"let [a, b, ...rest] = [1]\n[a, b, rest]", "[1, null, []]"},
		{"let [...all] = [1, 2]\nall", "[1, 2]"},
		{`let [h, ...t] = "héllo"` + "\n[h, t]", `["h", "éllo"]`},
		{`let {name, ...others} = {"name": "Ana", "age": 30, "city": "Rio"}` + "\n[name, others]", `["Ana", {"age": 30, "city": "Rio"}]`},
		{`let {...copy} = {"a": 1}` + "\ncopy", `{"a": 1}`},
		{"fn f([head, ...tail]) { tail.length }\nf([1, 2, 3])", "2"},
		{"let out = []\nrepeat [k, ...vs] in [[1, 2, 3], [4]] { out.insert(vs) }\nout", "[[2, 3], []]"},
		{"let out = []\nrepeat {id, ...data} in [{\"id\": 1, \"x\": 2}] { out.insert(data) }\nout", `[{"x": 2}]`},
		{`let {a, ...r} = 5`, "ERROR at line 1, column 5: cannot read property \"a\" of an integer"},
		{"let [...rest, last] = [1]", "ERROR: parser errors: line 1:13: ...rest must be the last name in [...rest]"},
		{"fn f([a, ...b]) { 1 }\nf", "fn f([a, ...b]) {\n{ 1 }\n}"},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
