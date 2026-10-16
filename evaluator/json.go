package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
)

// JSON is parsed and written here instead of with encoding/json, which
// would add Go's reflection to the WebAssembly build and make it about 140
// KB larger.

func init() {
	builtins["parseJSON"] = &object.Builtin{Name: "parseJSON", Fn: parseJSON}
	builtins["toJSON"] = &object.Builtin{Name: "toJSON", Fn: toJSON}
}

// parseJSON converts JSON text to values: objects keep the order of their
// keys, and numbers without a fraction or exponent become integers.
func parseJSON(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for parseJSON. got=%d, want=1", len(args))}
	}
	text, ok := args[0].(*object.String)
	if !ok {
		return &object.Error{Message: fmt.Sprintf("argument to parseJSON must be a string, got %s", typeName(args[0]))}
	}

	p := &jsonParser{text: text.Value}
	value, err := p.value()
	if err == "" {
		p.skipSpace()
		if p.pos < len(p.text) {
			err = "unexpected text after the value"
		}
	}
	if err != "" {
		return &object.Error{Message: "invalid JSON" + jsonPosition(p.text, p.pos) + ": " + err}
	}
	return value
}

type jsonParser struct {
	text string
	pos  int
}

func (p *jsonParser) skipSpace() {
	for p.pos < len(p.text) && strings.IndexByte(" \t\r\n", p.text[p.pos]) >= 0 {
		p.pos++
	}
}

// value parses the value at the current position, or returns why it
// can't.
func (p *jsonParser) value() (object.Object, string) {
	p.skipSpace()
	if p.pos >= len(p.text) {
		return nil, "the text ended too early"
	}

	switch c := p.text[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		s, err := p.string()
		if err != "" {
			return nil, err
		}
		return &object.String{Value: s}, ""
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	case strings.HasPrefix(p.text[p.pos:], "true"):
		p.pos += 4
		return TRUE, ""
	case strings.HasPrefix(p.text[p.pos:], "false"):
		p.pos += 5
		return FALSE, ""
	case strings.HasPrefix(p.text[p.pos:], "null"):
		p.pos += 4
		return NULL, ""
	}
	return nil, p.unexpected()
}

func (p *jsonParser) unexpected() string {
	r, _ := utf8.DecodeRuneInString(p.text[p.pos:])
	return fmt.Sprintf("unexpected %q", r)
}

func (p *jsonParser) object() (object.Object, string) {
	p.pos++ // {
	hash := object.NewHash()
	p.skipSpace()
	if p.pos < len(p.text) && p.text[p.pos] == '}' {
		p.pos++
		return hash, ""
	}

	for {
		p.skipSpace()
		if p.pos >= len(p.text) {
			return nil, "the text ended too early"
		}
		if p.text[p.pos] != '"' {
			return nil, p.unexpected() + ", expected a key in quotes"
		}
		key, err := p.string()
		if err != "" {
			return nil, err
		}
		p.skipSpace()
		if p.pos >= len(p.text) {
			return nil, "the text ended too early"
		}
		if p.text[p.pos] != ':' {
			return nil, p.unexpected() + ", expected ':'"
		}
		p.pos++
		value, err := p.value()
		if err != "" {
			return nil, err
		}
		hash.Set(&object.String{Value: key}, value)

		p.skipSpace()
		if p.pos >= len(p.text) {
			return nil, "the text ended too early"
		}
		switch p.text[p.pos] {
		case ',':
			p.pos++
		case '}':
			p.pos++
			return hash, ""
		default:
			return nil, p.unexpected() + ", expected ',' or '}'"
		}
	}
}

func (p *jsonParser) array() (object.Object, string) {
	p.pos++ // [
	elements := []object.Object{}
	p.skipSpace()
	if p.pos < len(p.text) && p.text[p.pos] == ']' {
		p.pos++
		return &object.Array{Elements: elements}, ""
	}

	for {
		element, err := p.value()
		if err != "" {
			return nil, err
		}
		elements = append(elements, element)

		p.skipSpace()
		if p.pos >= len(p.text) {
			return nil, "the text ended too early"
		}
		switch p.text[p.pos] {
		case ',':
			p.pos++
		case ']':
			p.pos++
			return &object.Array{Elements: elements}, ""
		default:
			return nil, p.unexpected() + ", expected ',' or ']'"
		}
	}
}

func (p *jsonParser) string() (string, string) {
	p.pos++ // "
	var out strings.Builder
	for {
		if p.pos >= len(p.text) {
			return "", "the text ended too early"
		}
		c := p.text[p.pos]
		switch {
		case c == '"':
			p.pos++
			return out.String(), ""
		case c < 0x20:
			return "", "control character in string"
		case c != '\\':
			out.WriteByte(c)
			p.pos++
			continue
		}

		// An escape
		p.pos++
		if p.pos >= len(p.text) {
			return "", "the text ended too early"
		}
		escape := p.text[p.pos]
		p.pos++
		switch escape {
		case '"', '\\', '/':
			out.WriteByte(escape)
		case 'b':
			out.WriteByte('\b')
		case 'f':
			out.WriteByte('\f')
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'u':
			r, ok := p.hex4()
			if !ok {
				return "", "invalid \\u escape"
			}
			// Characters outside the BMP come as a pair of escapes
			if utf16.IsSurrogate(r) && strings.HasPrefix(p.text[p.pos:], "\\u") {
				p.pos += 2
				low, ok := p.hex4()
				if !ok {
					return "", "invalid \\u escape"
				}
				r = utf16.DecodeRune(r, low)
			}
			out.WriteRune(r)
		default:
			p.pos--
			return "", fmt.Sprintf("invalid escape \\%c", escape)
		}
	}
}

func (p *jsonParser) hex4() (rune, bool) {
	if p.pos+4 > len(p.text) {
		return 0, false
	}
	n, err := strconv.ParseUint(p.text[p.pos:p.pos+4], 16, 32)
	if err != nil {
		return 0, false
	}
	p.pos += 4
	return rune(n), true
}

func (p *jsonParser) number() (object.Object, string) {
	start := p.pos
	isFloat := false
	if p.text[p.pos] == '-' {
		p.pos++
	}
	digits := func() int {
		n := 0
		for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
			p.pos++
			n++
		}
		return n
	}
	// The integer part has no leading zeros
	if p.pos < len(p.text) && p.text[p.pos] == '0' {
		p.pos++
	} else if digits() == 0 {
		return nil, "invalid number"
	}
	if p.pos < len(p.text) && p.text[p.pos] == '.' {
		isFloat = true
		p.pos++
		if digits() == 0 {
			return nil, "invalid number"
		}
	}
	if p.pos < len(p.text) && (p.text[p.pos] == 'e' || p.text[p.pos] == 'E') {
		isFloat = true
		p.pos++
		if p.pos < len(p.text) && (p.text[p.pos] == '+' || p.text[p.pos] == '-') {
			p.pos++
		}
		if digits() == 0 {
			return nil, "invalid number"
		}
	}

	literal := p.text[start:p.pos]
	if !isFloat {
		if n, err := strconv.ParseInt(literal, 10, 64); err == nil {
			return &object.Integer{Value: n}, ""
		}
	}
	f, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		return nil, fmt.Sprintf("number %s is out of range", literal)
	}
	return &object.Float{Value: f}, ""
}

// jsonPosition describes a position in text.
func jsonPosition(text string, offset int) string {
	offset = min(offset, len(text))
	before := text[:offset]
	line := strings.Count(before, "\n") + 1
	column := utf8.RuneCountInString(before[strings.LastIndex(before, "\n")+1:]) + 1
	return fmt.Sprintf(" at line %d, column %d", line, column)
}

// toJSON converts a value to JSON text, indented by the number of spaces or
// the string given as second argument.
func toJSON(args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for toJSON. got=%d, want=1 or 2", len(args))}
	}

	indent := ""
	if len(args) == 2 {
		switch arg := args[1].(type) {
		case *object.Integer:
			indent = strings.Repeat(" ", int(min(max(arg.Value, 0), 10)))
		case *object.String:
			indent = arg.Value
		default:
			return &object.Error{Message: fmt.Sprintf("indent for toJSON must be an integer or a string, got %s", typeName(args[1]))}
		}
	}

	var out strings.Builder
	if err := encodeJSON(&out, args[0], indent, 0); err != nil {
		return err
	}
	return &object.String{Value: out.String()}
}

func encodeJSON(out *strings.Builder, obj object.Object, indent string, depth int) *object.Error {
	newline := func(depth int) {
		if indent != "" {
			out.WriteByte('\n')
			out.WriteString(strings.Repeat(indent, depth))
		}
	}

	switch obj := obj.(type) {
	case *object.Null:
		out.WriteString("null")
	case *object.Boolean:
		out.WriteString(strconv.FormatBool(obj.Value))
	case *object.Integer:
		out.WriteString(strconv.FormatInt(obj.Value, 10))
	case *object.Float:
		if math.IsInf(obj.Value, 0) || math.IsNaN(obj.Value) {
			return &object.Error{Message: fmt.Sprintf("cannot convert %s to JSON", obj.Inspect())}
		}
		out.WriteString(strconv.FormatFloat(obj.Value, 'g', -1, 64))
	case *object.String:
		writeJSONString(out, obj.Value)
	case *object.Array:
		if len(obj.Elements) == 0 {
			out.WriteString("[]")
			break
		}
		out.WriteByte('[')
		for i, element := range obj.Elements {
			if i > 0 {
				out.WriteByte(',')
			}
			newline(depth + 1)
			if err := encodeJSON(out, element, indent, depth+1); err != nil {
				return err
			}
		}
		newline(depth)
		out.WriteByte(']')
	case *object.Hash:
		if len(obj.Pairs) == 0 {
			out.WriteString("{}")
			break
		}
		out.WriteByte('{')
		for i, pair := range obj.OrderedPairs() {
			if i > 0 {
				out.WriteByte(',')
			}
			newline(depth + 1)
			// JSON keys are strings, so other keys use their text
			writeJSONString(out, objectToString(pair.Key))
			out.WriteByte(':')
			if indent != "" {
				out.WriteByte(' ')
			}
			if err := encodeJSON(out, pair.Value, indent, depth+1); err != nil {
				return err
			}
		}
		newline(depth)
		out.WriteByte('}')
	default:
		return &object.Error{Message: fmt.Sprintf("cannot convert %s to JSON", typeName(obj))}
	}
	return nil
}

func writeJSONString(out *strings.Builder, s string) {
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(out, `\u%04x`, r)
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
}
