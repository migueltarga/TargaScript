package vm

import "testing"

func TestPowerOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2 ** 10", "1024"},
		{"2 ** 0", "1"},
		{"(-3) ** 3", "-27"},
		{"2 ** 3 ** 2", "512"},
		{"-2 ** 2", "4"},
		{"2 * 3 ** 2", "18"},
		{"2 ** -1", "0.5"},
		{"2.5 ** 2", "6.25"},
		{"4 ** 0.5", "2"},
		{"2 ** 63", "ERROR at line 1, column 3: integer overflow"},
		{"2 ** 62", "4611686018427387904"},
		{"let x = 3\nx ** 2", "9"},
		{`"a" ** 2`, `ERROR at line 1, column 5: cannot use "**" with a string and an integer`},
	}

	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%q\n got: %s\nwant: %s", tt.input, got, tt.expected)
		}
	}
}
