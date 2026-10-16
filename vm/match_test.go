package vm

import "testing"

func TestMatch(t *testing.T) {
	size := `fn size(n) {
    match n {
        0 => "none"
        1, 2 => "a few"
        3..9 => "several"
        _ => "many"
    }
}
`
	tests := []struct {
		input    string
		expected string
	}{
		{size + "[size(0), size(2), size(5), size(9), size(10), size(-1)]", `["none", "a few", "several", "several", "many", "many"]`},
		{`match "b" { "a" => 1, "b" => 2, _ => 3 }`, "2"},
		{`match "z" { "a" => 1 }`, "null"},
		{`match [1, 2] { [1, 2] => "pair", _ => "other" }`, "pair"},
		{`match 2.5 { 1..3 => "in" }`, "in"},
		{`match "x" { 1..3 => "in", _ => "not a number" }`, "not a number"},
		{"let x = 3\nmatch x { 1 => \"one\"\n x => \"same as x\" }", "same as x"},
		{"let calls = 0\nfn f() { calls++\n2 }\nmatch f() { 1 => 0, 2 => 0 }\ncalls", "1"},
		{"match 1 {\n  1 => {\n    let a = 10\n    a + 1\n  }\n  _ => 0\n}", "11"},
		{"match 1 { 1 => ({\"a\": 1}) }", `{"a": 1}`},
		{"fn grade(score) {\n  match true {\n    score >= 90 => \"A\"\n    score >= 80 => \"B\"\n    _ => \"C\"\n  }\n}\n[grade(95), grade(85), grade(10)]", `["A", "B", "C"]`},
		{"let out = []\nrepeat i in 1..4 { out.insert(match i % 2 { 0 => \"even\", _ => \"odd\" }) }\nout", `["odd", "even", "odd", "even"]`},
		{"match 1 { 1 => missing }", `ERROR at line 1, column 16: "missing" is not defined`},
		{"match 1 { 1 => 2", `ERROR: parser errors: line 1:1: this match is never closed, add a "}" after its last arm`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
