// Package ints reads integers from text as strconv.ParseInt does. strconv's
// errors quote the text, which keeps strconv.Quote and its tables in the
// WebAssembly build, so this reports failures with booleans instead.
package ints

// Parse reads s as strconv.ParseInt(s, base, 64) does: an optional sign, then
// digits in base, from 2 to 36. With base 0 the base comes from a prefix,
// 0b, 0o, 0x or a 0 alone for octal, and underscores may separate digits.
// ok reports whether s is a number, and tooLarge whether it is one out of
// the range of int64.
func Parse(s string, base int) (n int64, ok, tooLarge bool) {
	text := s
	negative := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		negative = s[0] == '-'
		s = s[1:]
	}
	underscores := base == 0
	if base == 0 {
		base = 10
		if len(s) > 1 && s[0] == '0' {
			switch s[1] | 0x20 {
			case 'b':
				base, s = 2, s[2:]
			case 'o':
				base, s = 8, s[2:]
			case 'x':
				base, s = 16, s[2:]
			default:
				base, s = 8, s[1:]
			}
		}
	}
	if s == "" {
		return 0, false, false
	}
	var u uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' && underscores {
			continue
		}
		d := digit(c)
		if d >= base {
			return 0, false, false
		}
		// Like strconv, a number too large is reported where it overflows,
		// whatever follows
		next := u*uint64(base) + uint64(d)
		if u > (1<<64-1)/uint64(base) || next < u*uint64(base) {
			return 0, true, true
		}
		u = next
	}
	if underscores && !underscoreOK(text) {
		return 0, false, false
	}
	limit := uint64(1 << 63)
	if !negative && u >= limit || negative && u > limit {
		return 0, true, true
	}
	if negative {
		return -int64(u), true, false
	}
	return int64(u), true, false
}

// digit is the value of a digit in bases up to 36, or 36 for other bytes.
func digit(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c|0x20 && c|0x20 <= 'z':
		return int(c|0x20-'a') + 10
	}
	return 36
}

// underscoreOK reports whether the underscores in s are where Go allows
// them in number literals: only between digits, or between a base prefix
// and a digit. It is strconv's rule.
func underscoreOK(s string) bool {
	// saw is the last character seen: '^' for the start, '0' for a digit
	// or a base prefix, '_' for an underscore, and '!' for anything else
	saw := '^'
	i := 0
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	hex := false
	if len(s) >= 2 && s[0] == '0' && (s[1]|0x20 == 'b' || s[1]|0x20 == 'o' || s[1]|0x20 == 'x') {
		i = 2
		saw = '0'
		hex = s[1]|0x20 == 'x'
	}
	for ; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= s[i]|0x20 && s[i]|0x20 <= 'f' {
			saw = '0'
			continue
		}
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		if saw == '_' {
			return false
		}
		saw = '!'
	}
	return saw != '_'
}
