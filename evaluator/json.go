package evaluator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/migueltarga/TargaScript/object"
)

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

	decoder := json.NewDecoder(strings.NewReader(text.Value))
	decoder.UseNumber()
	value, err := decodeJSON(decoder)
	offset := decoder.InputOffset()
	if err == nil {
		rest := text.Value[offset:]
		if trimmed := strings.TrimLeft(rest, " \t\r\n"); trimmed != "" {
			offset += int64(len(rest) - len(trimmed))
			err = errors.New("unexpected text after the value")
		}
	}
	if err != nil {
		return &object.Error{Message: "invalid JSON" + jsonPosition(text.Value, offset) + ": " + jsonMessage(err)}
	}
	return value
}

func decodeJSON(decoder *json.Decoder) (object.Object, error) {
	tok, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch tok := tok.(type) {
	case json.Delim:
		switch tok {
		case '[':
			elements := []object.Object{}
			for decoder.More() {
				element, err := decodeJSON(decoder)
				if err != nil {
					return nil, err
				}
				elements = append(elements, element)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, err
			}
			return &object.Array{Elements: elements}, nil
		case '{':
			hash := object.NewHash()
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				value, err := decodeJSON(decoder)
				if err != nil {
					return nil, err
				}
				hash.Set(&object.String{Value: key.(string)}, value)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, err
			}
			return hash, nil
		}
	case string:
		return &object.String{Value: tok}, nil
	case json.Number:
		if n, err := strconv.ParseInt(string(tok), 10, 64); err == nil {
			return &object.Integer{Value: n}, nil
		}
		f, err := strconv.ParseFloat(string(tok), 64)
		if err != nil {
			return nil, fmt.Errorf("number %s is out of range", tok)
		}
		return &object.Float{Value: f}, nil
	case bool:
		return nativeBoolToBooleanObject(tok), nil
	case nil:
		return NULL, nil
	}
	return nil, fmt.Errorf("unexpected %v", tok)
}

// jsonPosition describes where in text the parser stopped.
func jsonPosition(text string, offset int64) string {
	if offset > int64(len(text)) {
		offset = int64(len(text))
	}
	before := text[:offset]
	line := strings.Count(before, "\n") + 1
	column := len([]rune(before[strings.LastIndex(before, "\n")+1:])) + 1
	return fmt.Sprintf(" at line %d, column %d", line, column)
}

func jsonMessage(err error) string {
	if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
		return "the text ended too early"
	}
	return err.Error()
}

// toJSON converts a value to JSON text, indented by the number of spaces or
// the string given as second argument.
func toJSON(args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return &object.Error{Message: fmt.Sprintf("wrong number of arguments for toJSON. got=%d, want=1 or 2", len(args))}
	}

	var out bytes.Buffer
	if err := encodeJSON(&out, args[0]); err != nil {
		return err
	}
	if len(args) == 1 {
		return &object.String{Value: out.String()}
	}

	indent := ""
	switch arg := args[1].(type) {
	case *object.Integer:
		indent = strings.Repeat(" ", int(min(max(arg.Value, 0), 10)))
	case *object.String:
		indent = arg.Value
	default:
		return &object.Error{Message: fmt.Sprintf("indent for toJSON must be an integer or a string, got %s", typeName(args[1]))}
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, out.Bytes(), "", indent); err != nil {
		return &object.Error{Message: err.Error()}
	}
	return &object.String{Value: indented.String()}
}

func encodeJSON(out *bytes.Buffer, obj object.Object) *object.Error {
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
		out.WriteByte('[')
		for i, element := range obj.Elements {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := encodeJSON(out, element); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case *object.Hash:
		out.WriteByte('{')
		for i, pair := range obj.OrderedPairs() {
			if i > 0 {
				out.WriteByte(',')
			}
			// JSON keys are strings, so other keys use their text
			writeJSONString(out, objectToString(pair.Key))
			out.WriteByte(':')
			if err := encodeJSON(out, pair.Value); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return &object.Error{Message: fmt.Sprintf("cannot convert %s to JSON", typeName(obj))}
	}
	return nil
}

func writeJSONString(out *bytes.Buffer, s string) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.Encode(s)
	out.Truncate(out.Len() - 1) // Encode adds a newline
}
