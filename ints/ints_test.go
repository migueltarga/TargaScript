package ints

import (
	"errors"
	"strconv"
	"testing"
)

func check(t *testing.T, s string, base int) {
	want, err := strconv.ParseInt(s, base, 64)
	n, ok, tooLarge := Parse(s, base)
	if errors.Is(err, strconv.ErrRange) {
		if !ok || !tooLarge {
			t.Errorf("Parse(%q, %d) = %d %v %v, want too large", s, base, n, ok, tooLarge)
		}
		return
	}
	if (err == nil) != ok || tooLarge || ok && n != want {
		t.Errorf("Parse(%q, %d) = %d %v %v, want %d %v", s, base, n, ok, tooLarge, want, err)
	}
}

func TestParse(t *testing.T) {
	for _, s := range []string{"0", "-0", "+7", "42", "-9223372036854775808", "9223372036854775807",
		"9223372036854775808", "-9223372036854775809", "99999999999999999999999", "0x_ff", "0xFF", "0b101",
		"0o17", "017", "08", "1_000", "_1", "1_", "1__0", "0x", "", "-", "+", "12a", "0_7", "0x_", "0b2"} {
		for _, base := range []int{0, 10, 16} {
			check(t, s, base)
		}
	}
	// Too large where it overflows, though a letter of another base follows
	check(t, "70000000000000X", 28)
}

func FuzzParse(f *testing.F) {
	f.Add("0x_1f", 0)
	f.Add("-123", 10)
	f.Fuzz(func(t *testing.T, s string, base int) {
		if base != 0 && (base < 2 || base > 36) {
			return
		}
		check(t, s, base)
	})
}
