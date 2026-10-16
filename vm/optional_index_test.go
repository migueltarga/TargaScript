package vm

import "testing"

func TestOptionalIndex(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let g = [[1, 2], [3, 4]]; g[1]?.[0]", "3"},
		{"let g = [[1, 2]]; g[5]?.[0]", "null"},
		{"let m = null; m?.[0]", "null"},
		{"let m = null; m?.[0].x.y", "null"},
		{`let o = {list: [{n: 1}]}; o.list?.[0]?.n`, "1"},
		{`let o = {}; o.list?.[0]?.n ?? "none"`, `"none"`},
		{`let s = "abc"; s?.[-1]`, `"c"`},
	}
	for _, tt := range tests {
		got := testEval(tt.input)
		inspected := got.Inspect()
		if got.Type() == "STRING" {
			inspected = `"` + inspected + `"`
		}
		if inspected != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, inspected, tt.expected)
		}
	}
}
