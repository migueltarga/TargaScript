package vm

import "testing"

// Indexes give the same values in the VM's own paths and the general one
func TestIndexes(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let a = [1, 2, 3]\na[0] + a[2]", "4"},
		{"[1, 2, 3][-1]", "3"},
		{"[1, 2, 3][3]", "null"},
		{"let a = [1, 2]\na[1] = 5\na", "[1, 5]"},
		{"let a = [1, 2]\na[-1] = 7\na", "[1, 7]"},
		{`let h = {"x": 1}` + "\nh[\"x\"] + (h[\"y\"] ?? 10)", "11"},
		{`let h = {"x": 1}` + "\nh[\"x\"] = 2\nh[\"y\"] = 3\nh", `{"x": 2, "y": 3}`},
		{`let h = {"x": 1, "y": 2}` + "\nlet s = 0\nrepeat k in h { s = s * 10 + h[k] }\ns", "12"},
		{"let h = {}\nh[1] = \"one\"\nh[1]", "one"},
		{`"héllo"[1]`, "é"},
		{"let counts = {}\nrepeat w in [\"a\", \"b\", \"a\"] { counts[w] = (counts[w] ?? 0) + 1 }\ncounts", `{"a": 2, "b": 1}`},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
	if !isError(testEval("let a = [1]\na[5] = 1")) {
		t.Error("setting past the end should fail")
	}
	if !isError(testEval("null[0]")) {
		t.Error("indexing null should fail")
	}
}
