package floats

import (
	"math"
	"strings"
)

// parse reads s like strconv.ParseFloat(s, 64), reporting whether it is a
// number within range. decimal reads a decimal number in JavaScript's
// syntax, which the ones strconv accepts are, without underscores.
func parse(s string, decimal func(string) float64) (float64, bool) {
	if f, n, ok := special(s); ok {
		return f, n == len(s)
	}
	mantissa, exp, neg, trunc, hex, n, ok := readFloat(s)
	if !ok || n != len(s) {
		return 0, false
	}
	if hex {
		return atofHex(mantissa, exp, neg, trunc)
	}
	f := decimal(strings.ReplaceAll(s, "_", ""))
	return f, !math.IsInf(f, 0)
}

func lower(c byte) byte { return c | ('x' - 'X') }

// special reads infinity or NaN, as strconv does.
func special(s string) (f float64, n int, ok bool) {
	if len(s) == 0 {
		return 0, 0, false
	}
	sign, nsign := 1, 0
	switch s[0] {
	case '+', '-':
		if s[0] == '-' {
			sign = -1
		}
		nsign = 1
		s = s[1:]
		fallthrough
	case 'i', 'I':
		n := prefixIgnoringCase(s, "infinity")
		if 3 < n && n < 8 {
			n = 3
		}
		if n == 3 || n == 8 {
			return math.Inf(sign), nsign + n, true
		}
	case 'n', 'N':
		if prefixIgnoringCase(s, "nan") == 3 {
			return math.NaN(), 3, true
		}
	}
	return 0, 0, false
}

func prefixIgnoringCase(s, prefix string) int {
	n := min(len(prefix), len(s))
	for i := 0; i < n; i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != prefix[i] {
			return i
		}
	}
	return n
}

// readFloat reads the syntax of a decimal or hexadecimal number, as strconv
// does, with the mantissa and exponent of a hexadecimal one.
func readFloat(s string) (mantissa uint64, exp int, neg, trunc, hex bool, i int, ok bool) {
	underscores := false
	if i >= len(s) {
		return
	}
	switch {
	case s[i] == '+':
		i++
	case s[i] == '-':
		neg = true
		i++
	}
	base := uint64(10)
	maxMantDigits := 19
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base, maxMantDigits, expChar, hex = 16, 16, 'p', true
		i += 2
	}
	sawdot, sawdigits := false, false
	nd, ndMant, dp := 0, 0, 0
loop:
	for ; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '_':
			underscores = true
			continue
		case c == '.':
			if sawdot {
				break loop
			}
			sawdot = true
			dp = nd
			continue
		case '0' <= c && c <= '9':
			sawdigits = true
			if c == '0' && nd == 0 {
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa = mantissa*base + uint64(c-'0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			continue
		case base == 16 && 'a' <= lower(c) && lower(c) <= 'f':
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa = mantissa*16 + uint64(lower(c)-'a'+10)
				ndMant++
			} else {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}
	if base == 16 {
		dp *= 4
		ndMant *= 4
	}
	if i < len(s) && lower(s[i]) == expChar {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		dp += e * esign
	} else if base == 16 {
		// A hexadecimal number needs its p exponent
		return
	}
	if mantissa != 0 {
		exp = dp - ndMant
	}
	if underscores && !underscoreOK(s[:i]) {
		return
	}
	ok = true
	return
}

// underscoreOK reports whether the underscores in s only separate digits,
// as strconv requires.
func underscoreOK(s string) bool {
	saw := byte('^')
	i := 0
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0'
		hex = lower(s[1]) == 'x'
	}
	for ; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
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

// atofHex makes the float of a hexadecimal mantissa and binary exponent,
// rounded to even, as strconv does, reporting whether it is within range.
func atofHex(mantissa uint64, exp int, neg, trunc bool) (float64, bool) {
	const mantbits, expbits, bias = 52, 11, -1023
	maxExp := 1<<expbits + bias - 2
	minExp := bias + 1
	exp += mantbits
	for mantissa != 0 && mantissa>>(mantbits+2) == 0 {
		mantissa <<= 1
		exp--
	}
	if trunc {
		mantissa |= 1
	}
	for mantissa>>(1+mantbits+2) != 0 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	for mantissa > 1 && exp < minExp-2 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	round := mantissa & 3
	mantissa >>= 2
	round |= mantissa & 1
	exp += 2
	if round == 3 {
		mantissa++
		if mantissa == 1<<(1+mantbits) {
			mantissa >>= 1
			exp++
		}
	}
	if mantissa>>mantbits == 0 {
		exp = bias
	}
	ok := true
	if exp > maxExp {
		mantissa = 1 << mantbits
		exp = maxExp + 1
		ok = false
	}
	bits := mantissa & (1<<mantbits - 1)
	bits |= uint64((exp-bias)&(1<<expbits-1)) << mantbits
	if neg {
		bits |= 1 << mantbits << expbits
	}
	return math.Float64frombits(bits), ok
}
