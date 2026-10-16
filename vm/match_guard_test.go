package vm

import "testing"

func TestMatchGuards(t *testing.T) {
	grade := "fn grade(s) { match s {\n100 => \"perfect\"\n_ if s >= 90 => \"A\"\n80..89 => \"B\"\n_ => \"C\"\n} }\n"
	tests := []struct{ input, expected string }{
		{grade + "grade(100)", `"perfect"`},
		{grade + "grade(95)", `"A"`},
		{grade + "grade(85)", `"B"`},
		{grade + "grade(10)", `"C"`},
		// A guard that fails moves on to the next arm, even when the pattern matched
		{`let x = 2; match 1 { 1 if x > 5 => "big", 1 => "small", _ => "other" }`, `"small"`},
		{`match "GET" { "GET", "HEAD" if false => 1, _ => 2 }`, "2"},
		{"match 3 { _ if false => 1 }", "null"},
		// The guard runs only when the pattern matches
		{"let calls = 0; fn check() { calls++; true }; match 1 { 2 if check() => 0, _ => calls }", "0"},
		{"match 1 { 1 if true => { let a = 5; a } }", "5"},
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
