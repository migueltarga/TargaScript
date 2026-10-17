package strfmt

import "unicode/utf8"

// Quote puts s in double quotes, as strconv.Quote does, without its tables
// of the characters that print.
func Quote(s string) string {
	return quote(s, '"', false)
}

// quote puts s in quote marks, as strconv.Quote does, escaping quote marks,
// backslashes and characters that don't print. With ascii it escapes the
// rest of non-ASCII too. Characters that don't print are the control and
// format ones and the spaces other than ' '. Unlike strconv it prints the
// code points Unicode hasn't assigned, which would take tables of kilobytes
// in the WebAssembly build to tell apart.
func quote(s string, mark byte, ascii bool) string {
	out := []byte{mark}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			out = append(out, `\x`...)
			out = appendHex(out, uint32(s[i]), 2)
			i++
			continue
		}
		out = appendEscaped(out, r, mark, ascii)
		i += size
	}
	return string(append(out, mark))
}

func appendEscaped(out []byte, r rune, mark byte, ascii bool) []byte {
	if r == rune(mark) || r == '\\' {
		return append(out, '\\', byte(r))
	}
	if printable(r) && (!ascii || r < utf8.RuneSelf) {
		return utf8.AppendRune(out, r)
	}
	switch r {
	case '\a':
		return append(out, `\a`...)
	case '\b':
		return append(out, `\b`...)
	case '\f':
		return append(out, `\f`...)
	case '\n':
		return append(out, `\n`...)
	case '\r':
		return append(out, `\r`...)
	case '\t':
		return append(out, `\t`...)
	case '\v':
		return append(out, `\v`...)
	}
	switch {
	case r < ' ' || r == 0x7f:
		return appendHex(append(out, `\x`...), uint32(r), 2)
	case r < 0x10000:
		return appendHex(append(out, `\u`...), uint32(r), 4)
	}
	return appendHex(append(out, `\U`...), uint32(r), 8)
}

func printable(r rune) bool {
	switch {
	case r < ' ', r >= 0x7f && r <= 0xa0, r == 0xad, r >= 0x600 && r <= 0x605,
		r == 0x61c, r == 0x6dd, r == 0x70f, r == 0x1680, r == 0x180e,
		r >= 0x2000 && r <= 0x200f, r >= 0x2028 && r <= 0x202f,
		r >= 0x205f && r <= 0x206f, r == 0x3000, r >= 0xd800 && r < 0xe000,
		r == 0xfeff, r >= 0xfff9 && r <= 0xfffb, r > utf8.MaxRune:
		return false
	}
	return true
}

func appendHex(out []byte, n uint32, digits int) []byte {
	for shift := (digits - 1) * 4; shift >= 0; shift -= 4 {
		out = append(out, "0123456789abcdef"[n>>uint(shift)&0xf])
	}
	return out
}

// canBackquote reports whether s can be put in backquotes as it is, as
// strconv.CanBackquote does.
func canBackquote(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || r == '`' || r == 0xfeff || r < ' ' && r != '\t' || r == 0x7f {
			return false
		}
		i += size
	}
	return true
}
