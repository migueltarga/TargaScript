//go:build !(tinygo && wasm)

package floats

import "strconv"

// Format formats f like strconv.FormatFloat(f, fmt, prec, 64), for the
// formats 'e', 'E', 'f', 'g' and 'G'.
func Format(f float64, fmt byte, prec int) string {
	return strconv.FormatFloat(f, fmt, prec, 64)
}

// Parse reads s like strconv.ParseFloat(s, 64), and reports whether it is
// a number within range.
func Parse(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}
