package vm

import "testing"

// A sort that fails leaves the array as it was
func TestFailedSortKeepsTheArray(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let xs = [3, 1, 2]\ntry { xs.sort((a, b) => a.nope) } catch e { }\nxs", "[3, 1, 2]"},
		{"let xs = [3, 1, 2]\nlet n = 0\ntry { xs.sort((a, b) => { n += 1\nif n > 2 { throw \"stop\" }\na - b }) } catch e { }\nxs", "[3, 1, 2]"},
		{"let xs = [3, 1, \"a\"]\ntry { xs.sort() } catch e { }\nxs", `[3, 1, "a"]`},
		{"[{k: 1, i: 0}, {k: 0, i: 1}, {k: 1, i: 2}].sort((a, b) => a.k - b.k).map(it.i)", "[1, 0, 2]"},
	}
	for _, tt := range tests {
		if got := testEval(tt.input).Inspect(); got != tt.expected {
			t.Errorf("%s: got %s, want %s", tt.input, got, tt.expected)
		}
	}
}
