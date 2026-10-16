package evaluator

import (
	"strings"

	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/strfmt"
)

func init() {
	builtins["escapeHTML"] = &object.Builtin{Name: "escapeHTML", Fn: escapeHTMLBuiltin}
	builtins["encodeURL"] = &object.Builtin{Name: "encodeURL", Fn: encodeURLBuiltin}
	builtins["decodeURL"] = &object.Builtin{Name: "decodeURL", Fn: decodeURLBuiltin}
}

// escapeHTMLBuiltin makes text safe to put in HTML, inside elements or
// quoted attributes. Values that aren't strings are shown as print shows
// them first.
func escapeHTMLBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "escapeHTML", len(args), "1")
	}
	text := objectToString(args[0])
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		case '"':
			out.WriteString("&#34;")
		case '\'':
			out.WriteString("&#39;")
		default:
			out.WriteByte(c)
		}
	}
	return &object.String{Value: out.String()}
}

// encodeURLBuiltin encodes text to put in a part of a URL, like a query
// value: spaces, &, = and the rest become %XX.
func encodeURLBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("encodeURL", args, 1)
	if err != nil {
		return err
	}
	const hex = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(values[0]); i++ {
		c := values[0][i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-_.~", c) >= 0 {
			out.WriteByte(c)
		} else {
			out.WriteByte('%')
			out.WriteByte(hex[c>>4])
			out.WriteByte(hex[c&15])
		}
	}
	return &object.String{Value: out.String()}
}

// decodeURLBuiltin is the opposite of encodeURL. A + is read as a space, as
// in form data.
func decodeURLBuiltin(args ...object.Object) object.Object {
	values, err := stringArgs("decodeURL", args, 1)
	if err != nil {
		return err
	}
	text := values[0]
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == '+':
			out.WriteByte(' ')
		case c == '%':
			if i+2 >= len(text) || unhex(text[i+1]) < 0 || unhex(text[i+2]) < 0 {
				return &object.Error{Message: strfmt.Sprintf("cannot decode %q: it has a %% not followed by two hex digits", text)}
			}
			out.WriteByte(byte(unhex(text[i+1])<<4 | unhex(text[i+2])))
			i += 2
		default:
			out.WriteByte(c)
		}
	}
	return &object.String{Value: out.String()}
}

// unhex is the value of a hex digit, or -1.
func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c-'a') + 10
	case 'A' <= c && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
