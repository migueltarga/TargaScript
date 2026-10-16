package vm

import "testing"

func TestThousandsSeparators(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"${1234567:,}"`, "1,234,567"},
		{`"${1234.5:,}"`, "1,234.5"},
		{`"${-9876543.219:,.2f}"`, "-9,876,543.22"},
		{`"${42:,}"`, "42"},
		{`"${1000:,d}"`, "1,000"},
		{`format("%,d", 1234567)`, "1,234,567"},
		{`format("%12,d|", 1234567)`, "   1,234,567|"},
		{`format("%-12,.1f|", 98765.43)`, "98,765.4    |"},
		{`format("%,d", -1000)`, "-1,000"},
		{`format("%+,d", 1000)`, "+1,000"},
		// Without the comma, as before
		{`format("%05d|%5s|%-5s|", 42, "ab", "ab")`, "00042|   ab|ab   |"},
		{`format("%*d|%.*f", 5, 42, 2, 3.14159)`, "   42|3.14"},
		{`format("100%%")`, "100%"},
		{`format("%+06d|%-4d|", -42, 7)`, "-00042|7   |"},
		{`format("%200000000d", 1)`, "ERROR at line 1, column 7: printf width 200000000 is too large"},
		{`format("%.*f", 200000000, 1.5)`, "ERROR at line 1, column 7: printf precision 200000000 is too large"},
		{`format("%1000000d", 1).length`, "1000000"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %q, want %q", tt.input, got, tt.expected)
		}
	}
}
