// Package strfmt formats text like fmt's Sprintf, for the verbs, flags and
// types TargaScript uses. It doesn't use reflection, which keeps fmt and
// most of reflect out of the WebAssembly build.
//
// It knows strings, booleans, integers, floats and errors. Values with a
// String method must be passed as its result: a case for them in a type
// switch would keep every String method in the program, reflect's among
// them, in the WebAssembly build. The verbs are %v %s %q %d %b %o %x %X %c %t %e %E %f %F
// %g %G and %%, with the flags + - # 0 and space, a width and a precision,
// either of which can be *.
package strfmt

import (
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/floats"
)

// Sprintf formats args by format.
func Sprintf(format string, args ...any) string {
	var b strings.Builder
	write(&b, format, args)
	return b.String()
}

// Errorf returns an error with the text Sprintf gives.
func Errorf(format string, args ...any) error {
	return errors.New(Sprintf(format, args...))
}

// Fprintf writes the text Sprintf gives to w.
func Fprintf(w io.Writer, format string, args ...any) (int, error) {
	return io.WriteString(w, Sprintf(format, args...))
}

// spec is one verb with its flags.
type spec struct {
	plus, minus, sharp, space, zero bool
	width, precision                int // -1 when not given
}

func write(b *strings.Builder, format string, args []any) {
	next := 0
	take := func() (any, bool) {
		if next >= len(args) {
			return nil, false
		}
		next++
		return args[next-1], true
	}
	takeInt := func() int {
		arg, ok := take()
		if !ok {
			return -1
		}
		n, ok := toInt(arg)
		if !ok {
			return -1
		}
		return int(n)
	}

	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		i++
		s := spec{width: -1, precision: -1}
	flags:
		for ; i < len(format); i++ {
			switch format[i] {
			case '+':
				s.plus = true
			case '-':
				s.minus = true
			case '#':
				s.sharp = true
			case ' ':
				s.space = true
			case '0':
				s.zero = true
			default:
				break flags
			}
		}
		if i < len(format) && format[i] == '*' {
			s.width = takeInt()
			if s.width < 0 {
				s.minus, s.width = true, -s.width
			}
			i++
		} else {
			i, s.width = number(format, i)
		}
		if i < len(format) && format[i] == '.' {
			i++
			if i < len(format) && format[i] == '*' {
				s.precision = takeInt()
				i++
			} else {
				i, s.precision = number(format, i)
				if s.precision < 0 {
					s.precision = 0
				}
			}
		}
		if i >= len(format) {
			b.WriteString("%!(NOVERB)")
			break
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size - 1
		if verb == '%' {
			b.WriteByte('%')
			continue
		}
		arg, ok := take()
		if !ok {
			b.WriteString("%!" + string(verb) + "(MISSING)")
			continue
		}
		pad(b, s, verbText(s, verb, arg))
	}
	if next < len(args) {
		b.WriteString("%!(EXTRA)")
	}
}

// number reads the digits at i, returning where they end and their value,
// or -1 without any.
func number(format string, i int) (int, int) {
	start := i
	n := 0
	for i < len(format) && format[i] >= '0' && format[i] <= '9' {
		n = n*10 + int(format[i]-'0')
		i++
	}
	if i == start {
		return i, -1
	}
	return i, n
}

// pad writes text in the width of s.
func pad(b *strings.Builder, s spec, t text) {
	n := s.width - utf8.RuneCountInString(t.prefix+t.body)
	switch {
	case n <= 0:
		b.WriteString(t.prefix)
		b.WriteString(t.body)
	case s.minus:
		b.WriteString(t.prefix)
		b.WriteString(t.body)
		b.WriteString(strings.Repeat(" ", n))
	case s.zero && t.zeroable:
		// Zeros go after the sign and the base
		b.WriteString(t.prefix)
		b.WriteString(strings.Repeat("0", n))
		b.WriteString(t.body)
	default:
		b.WriteString(strings.Repeat(" ", n))
		b.WriteString(t.prefix)
		b.WriteString(t.body)
	}
}

// text is a formatted value: its sign and base, which zero padding goes
// after, and the rest.
type text struct {
	prefix, body string
	zeroable     bool
}

func verbText(s spec, verb rune, arg any) text {
	if verb == 'v' {
		// %+v and %#v are for Go values; for these, only %#v of a string
		// differs, quoting it
		s.plus = false
		if str, ok := arg.(string); ok && s.sharp {
			return text{body: strconv.Quote(truncate(str, s.precision))}
		}
		s.sharp = false
		switch arg.(type) {
		case string, error:
			verb = 's'
		case bool:
			verb = 't'
		case float64, float32:
			verb = 'g'
		case nil:
			return text{body: "<nil>"}
		default:
			if _, ok := toInt(arg); ok {
				verb = 'd'
			} else {
				verb = 's'
			}
		}
	}
	switch verb {
	case 's', 'q':
		str, ok := toString(arg)
		if !ok {
			if n, isInt := toInt(arg); isInt && verb == 'q' {
				if s.plus {
					return text{body: strconv.QuoteRuneToASCII(toRune(n))}
				}
				return text{body: strconv.QuoteRune(toRune(n))}
			}
			return bad(verb, arg)
		}
		str = truncate(str, s.precision)
		if verb == 'q' {
			switch {
			case s.sharp && strconv.CanBackquote(str):
				return text{body: "`" + str + "`"}
			case s.plus:
				return text{body: strconv.QuoteToASCII(str)}
			}
			return text{body: strconv.Quote(str)}
		}
		return text{body: str, zeroable: true}
	case 't':
		if v, ok := arg.(bool); ok {
			return text{body: strconv.FormatBool(v)}
		}
	case 'c':
		if n, ok := toInt(arg); ok {
			return text{body: string(toRune(n))}
		}
	case 'd', 'b', 'o', 'x', 'X':
		if n, ok := toInt(arg); ok {
			return integer(s, verb, n)
		}
		if str, ok := toString(arg); ok && (verb == 'x' || verb == 'X') {
			// The precision counts bytes here
			if s.precision >= 0 && s.precision < len(str) {
				str = str[:s.precision]
			}
			return hexString(s, verb, str)
		}
	case 'e', 'E', 'f', 'F', 'g', 'G':
		switch v := arg.(type) {
		case float64:
			return float(s, verb, v)
		case float32:
			return float(s, verb, float64(v))
		}
	}
	return bad(verb, arg)
}

// bad is what fmt writes for a value the verb doesn't take.
func bad(verb rune, arg any) text {
	value, _ := toString(arg)
	if n, ok := toInt(arg); ok {
		value = strconv.FormatInt(n, 10)
	}
	return text{body: "%!" + string(verb) + "(" + value + ")"}
}

func integer(s spec, verb rune, n int64) text {
	base := 10
	switch verb {
	case 'b':
		base = 2
	case 'o':
		base = 8
	case 'x', 'X':
		base = 16
	}
	negative := n < 0
	var digits string
	if negative {
		// The magnitude, without overflowing at the smallest int64
		digits = strconv.FormatUint(uint64(-(n+1))+1, base)
	} else {
		digits = strconv.FormatInt(n, base)
	}
	if verb == 'X' {
		digits = strings.ToUpper(digits)
	}
	if s.precision >= 0 {
		if s.precision == 0 && n == 0 {
			digits = ""
		} else if len(digits) < s.precision {
			digits = strings.Repeat("0", s.precision-len(digits)) + digits
		}
	}
	if digits == "" {
		// No digits, no sign
		return text{}
	}
	prefix := ""
	switch {
	case negative:
		prefix = "-"
	case s.plus:
		prefix = "+"
	case s.space:
		prefix = " "
	}
	if s.sharp {
		switch verb {
		case 'b':
			prefix += "0b"
		case 'o':
			if !strings.HasPrefix(digits, "0") {
				prefix += "0"
			}
		case 'x':
			prefix += "0x"
		case 'X':
			prefix += "0X"
		}
	}
	return text{prefix: prefix, body: digits, zeroable: s.precision < 0}
}

func hexString(s spec, verb rune, str string) text {
	const lower, upper = "0123456789abcdef", "0123456789ABCDEF"
	digits := lower
	if verb == 'X' {
		digits = upper
	}
	var b strings.Builder
	for i := 0; i < len(str); i++ {
		if i > 0 && s.space {
			b.WriteByte(' ')
		}
		if s.sharp && (i == 0 || s.space) {
			b.WriteString("0" + string(verb))
		}
		b.WriteByte(digits[str[i]>>4])
		b.WriteByte(digits[str[i]&15])
	}
	return text{body: b.String(), zeroable: true}
}

func float(s spec, verb rune, v float64) text {
	precision := s.precision
	if precision < 0 && verb != 'g' && verb != 'G' {
		precision = 6
	}
	format := byte(verb)
	if verb == 'F' {
		format = 'f'
	}
	body := floats.Format(v, format, precision)
	if s.sharp && !math.IsInf(v, 0) && !math.IsNaN(v) {
		body = alternate(body, verb, precision)
	}
	prefix := ""
	negative := body[0] == '-'
	if body[0] == '-' || body[0] == '+' {
		body = body[1:]
	}
	switch {
	case negative:
		prefix = "-"
	case s.plus:
		prefix = "+"
	case s.space:
		prefix = " "
	case body == "Inf":
		// fmt always signs infinities
		prefix = "+"
	}
	return text{prefix: prefix, body: body, zeroable: body != "Inf" && body != "NaN"}
}

// truncate keeps the first precision characters of str, all with -1.
func truncate(str string, precision int) string {
	if precision >= 0 && utf8.RuneCountInString(str) > precision {
		return string([]rune(str)[:precision])
	}
	return str
}

// toRune is n as a character, or the replacement character outside them.
func toRune(n int64) rune {
	if n < 0 || n > utf8.MaxRune {
		return utf8.RuneError
	}
	return rune(n)
}

// alternate is the # form of a formatted float, as fmt writes it: always
// with a decimal point, and for %g with its trailing zeros.
func alternate(num string, verb rune, precision int) string {
	digits := 0
	if verb == 'g' || verb == 'G' {
		digits = precision
		if digits == -1 {
			digits = 6
		}
	}
	sign := ""
	if num[0] == '-' || num[0] == '+' {
		sign, num = num[:1], num[1:]
	}
	tail := ""
	if i := strings.IndexAny(num, "eE"); i >= 0 {
		num, tail = num[:i], num[i:]
	}
	hasPoint, nonzero := false, false
	for i := 0; i < len(num); i++ {
		switch num[i] {
		case '.':
			hasPoint = true
		default:
			if num[i] != '0' {
				nonzero = true
			}
			if nonzero {
				digits--
			}
		}
	}
	if !hasPoint {
		if num == "0" {
			digits--
		}
		num += "."
	}
	if digits > 0 {
		num += strings.Repeat("0", digits)
	}
	return sign + num + tail
}

func toString(arg any) (string, bool) {
	switch v := arg.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case error:
		return v.Error(), true
	}
	return "", false
}

func toInt(arg any) (int64, bool) {
	switch v := arg.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	}
	return 0, false
}
