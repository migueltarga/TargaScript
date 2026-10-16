// Package floats turns floats into text and text into floats, as strconv
// does. The WebAssembly build has JavaScript find the digits and read the
// numbers, so the float code and tables of strconv stay out of it; this
// package lays the digits out as strconv would.
package floats

import (
	"math"
	"strconv"
	"strings"
)

// digits is a decimal number 0.d[0]d[1]... × 10^dp, with no zeros at the
// end of d. Zero has no digits.
type digits struct {
	d  []byte
	dp int
}

// parseExponent reads digits written like JavaScript's toExponential does:
// "1.2345e+21", "5e-324", "0e+0"; or the digits of an integer, as
// BigInt(f).toString() writes them.
func parseExponent(s string) digits {
	var n digits
	if !strings.Contains(s, "e") {
		s += "e+" + strconv.Itoa(len(s)-1)
		s = s[:1] + "." + s[1:]
	}
	i := 0
	for ; i < len(s) && s[i] != 'e'; i++ {
		if s[i] != '.' {
			n.d = append(n.d, s[i])
		}
	}
	exp, _ := strconv.Atoi(s[i+1:])
	n.dp = exp + 1
	for len(n.d) > 0 && n.d[len(n.d)-1] == '0' {
		n.d = n.d[:len(n.d)-1]
	}
	if len(n.d) == 0 {
		n.dp = 0
	}
	return n
}

// round rounds to nd digits, to the nearest and halfway to even, as the
// digits are all the number has.
func (n *digits) round(nd int) {
	if nd < 0 || nd >= len(n.d) {
		return
	}
	up := n.d[nd] > '5' || n.d[nd] == '5' && (nd+1 < len(n.d) || nd > 0 && (n.d[nd-1]-'0')%2 == 1)
	if !up {
		n.d = n.d[:nd]
		for len(n.d) > 0 && n.d[len(n.d)-1] == '0' {
			n.d = n.d[:len(n.d)-1]
		}
		if len(n.d) == 0 {
			n.dp = 0
		}
		return
	}
	for i := nd - 1; i >= 0; i-- {
		if n.d[i] < '9' {
			n.d[i]++
			n.d = n.d[:i+1]
			return
		}
	}
	// All nines: 999 rounds to 1000
	n.d = append(n.d[:0], '1')
	n.dp++
}

// format lays out f like strconv.FormatFloat(f, fmt, prec, 64), from its
// shortest digits, or with prec of at least 0 from its exact digits: the
// ones exact returns, rounded to that many after the first, or all of them
// for floats from 2^53, which are integers.
func format(f float64, fmt byte, prec int, shortest func(float64) string, exact func(float64, int) string) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "+Inf"
	case math.IsInf(f, -1):
		return "-Inf"
	}
	neg := math.Signbit(f)
	f = math.Abs(f)
	var n digits
	if prec < 0 {
		n = parseExponent(shortest(f))
		switch fmt {
		case 'e', 'E':
			prec = max(len(n.d)-1, 0)
		case 'f':
			prec = max(len(n.d)-n.dp, 0)
		case 'g', 'G':
			prec = len(n.d)
		}
	} else {
		// 100 digits after the first are enough to round exactly to the
		// digits a format shows: no float is closer than that to halfway
		// without being halfway
		n = parseExponent(exact(f, 100))
		switch fmt {
		case 'e', 'E':
			n.round(prec + 1)
		case 'f':
			n.round(n.dp + prec)
		case 'g', 'G':
			if prec == 0 {
				prec = 1
			}
			n.round(prec)
		}
		if fmt == 'g' || fmt == 'G' {
			return layoutG(neg, n, prec, fmt, false)
		}
	}
	switch fmt {
	case 'e', 'E':
		return layoutE(neg, n, prec, fmt)
	case 'f':
		return layoutF(neg, n, prec)
	}
	return layoutG(neg, n, prec, fmt, true)
}

func layoutG(neg bool, n digits, prec int, fmt byte, shortest bool) string {
	eprec := prec
	if eprec > len(n.d) && len(n.d) >= n.dp {
		eprec = len(n.d)
	}
	// Shortest digits decide with a precision of 6
	if shortest {
		eprec = 6
	}
	exp := n.dp - 1
	if exp < -4 || exp >= eprec {
		if prec > len(n.d) {
			prec = len(n.d)
		}
		return layoutE(neg, n, prec-1, fmt+'e'-'g')
	}
	if prec > n.dp {
		prec = len(n.d)
	}
	return layoutF(neg, n, max(prec-n.dp, 0))
}

func layoutE(neg bool, n digits, prec int, fmt byte) string {
	var b []byte
	if neg {
		b = append(b, '-')
	}
	first := byte('0')
	if len(n.d) > 0 {
		first = n.d[0]
	}
	b = append(b, first)
	if prec > 0 {
		b = append(b, '.')
		i := 1
		if m := min(len(n.d), prec+1); i < m {
			b = append(b, n.d[i:m]...)
			i = m
		}
		for ; i <= prec; i++ {
			b = append(b, '0')
		}
	}
	b = append(b, fmt)
	exp := n.dp - 1
	if len(n.d) == 0 {
		exp = 0
	}
	sign := byte('+')
	if exp < 0 {
		sign, exp = '-', -exp
	}
	b = append(b, sign)
	if exp < 10 {
		b = append(b, '0')
	}
	return string(strconv.AppendInt(b, int64(exp), 10))
}

func layoutF(neg bool, n digits, prec int) string {
	var b []byte
	if neg {
		b = append(b, '-')
	}
	if n.dp > 0 {
		m := min(len(n.d), n.dp)
		b = append(b, n.d[:m]...)
		for ; m < n.dp; m++ {
			b = append(b, '0')
		}
	} else {
		b = append(b, '0')
	}
	if prec > 0 {
		b = append(b, '.')
		for i := 1; i <= prec; i++ {
			c := byte('0')
			if j := n.dp + i - 1; 0 <= j && j < len(n.d) {
				c = n.d[j]
			}
			b = append(b, c)
		}
	}
	return string(b)
}
