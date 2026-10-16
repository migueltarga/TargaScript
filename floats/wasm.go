//go:build tinygo && wasm

package floats

import "unsafe"

// numberText writes f.toExponential(), or with digits of 0 or more
// f.toExponential(digits), or BigInt(f) for floats from 2^53, at buf, and
// returns its length.
//
//go:wasmimport targascript numberText
func hostNumberText(f float64, digits int32, buf *byte, size uint32) uint32

// numberParse returns Number(text).
//
//go:wasmimport targascript numberParse
func hostNumberParse(text *byte, length uint32) float64

func text(f float64, digits int) string {
	var buf [800]byte
	n := hostNumberText(f, int32(digits), &buf[0], uint32(len(buf)))
	return string(buf[:n])
}

// Format formats f like strconv.FormatFloat(f, fmt, prec, 64), for the
// formats 'e', 'E', 'f', 'g' and 'G', with digits from JavaScript.
func Format(f float64, fmt byte, prec int) string {
	return format(f, fmt, prec, func(f float64) string { return text(f, -1) }, text)
}

// Parse reads s like strconv.ParseFloat(s, 64), and reports whether it is
// a number within range. JavaScript reads decimal numbers.
func Parse(s string) (float64, bool) {
	return parse(s, func(s string) float64 {
		if s == "" {
			return 0
		}
		return hostNumberParse(unsafe.StringData(s), uint32(len(s)))
	})
}
