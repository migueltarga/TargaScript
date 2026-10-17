package evaluator

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/migueltarga/TargaScript/object"
)

func init() {
	builtins["bytes"] = &object.Builtin{Name: "bytes", Fn: bytesBuiltin}
}

// bytesBuiltin makes binary data: bytes(n) is n zero bytes, bytes(text) the
// UTF-8 of a string, bytes(list) the bytes of a list of numbers from 0 to
// 255, and bytes(data) a copy of other bytes.
func bytesBuiltin(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongArgs(0, 0, "bytes", len(args), "1")
	}
	switch arg := args[0].(type) {
	case *object.Integer:
		if arg.Value < 0 || arg.Value > maxStringLength {
			return errorText(0, 0, "bytes(n) takes a size from 0 to 100000000")
		}
		return &object.Bytes{Value: make([]byte, arg.Value)}
	case *object.String:
		return &object.Bytes{Value: []byte(arg.Value)}
	case *object.Bytes:
		return &object.Bytes{Value: append([]byte(nil), arg.Value...)}
	case *object.Array:
		data := make([]byte, len(arg.Elements))
		for i, e := range arg.Elements {
			n, ok := byteValue(e)
			if !ok {
				return errorText(0, 0, "bytes(list) takes numbers from 0 to 255, got "+e.Inspect()+" at "+strconv.Itoa(i))
			}
			data[i] = n
		}
		return &object.Bytes{Value: data}
	}
	return wrongType(0, 0, "argument to `bytes`", "a size, a string, a list of numbers or bytes", args[0])
}

func byteValue(obj object.Object) (byte, bool) {
	n, ok := obj.(*object.Integer)
	if !ok || n.Value < 0 || n.Value > 255 {
		return 0, false
	}
	return byte(n.Value), true
}

// bytesIndex is data[index], or null outside the data.
func bytesIndex(data *object.Bytes, index object.Object, line, column int) object.Object {
	i, ok := index.(*object.Integer)
	if !ok {
		return wrongType(line, column, "bytes index", "an integer", index)
	}
	at := fromEnd(i.Value, int64(len(data.Value)))
	if at < 0 || at >= int64(len(data.Value)) {
		return NULL
	}
	return &object.Integer{Value: int64(data.Value[at])}
}

// setBytesIndex stores a number from 0 to 255 in data[index].
func setBytesIndex(data *object.Bytes, index, val object.Object, line, column int) object.Object {
	i, ok := index.(*object.Integer)
	if !ok {
		return wrongType(line, column, "bytes index", "an integer", index)
	}
	at := fromEnd(i.Value, int64(len(data.Value)))
	if at < 0 || at >= int64(len(data.Value)) {
		return newError(line, column, "index out of bounds: %d", i.Value)
	}
	n, ok := byteValue(val)
	if !ok {
		return errorText(line, column, "a byte is a number from 0 to 255, got "+val.Inspect())
	}
	data.Value[at] = n
	return val
}

// bytesMethods are the methods of binary data.
var bytesMethods = []string{"slice", "text", "hex", "toArray", "read", "readBE", "write", "writeBE"}

// bytesMethod calls the methods of binary data.
func bytesMethod(data *object.Bytes, method string, args []object.Object, line, column int) object.Object {
	// The arguments each takes: slice one or two, read two, write three
	low, high := 0, 0
	switch method {
	case "slice":
		low, high = 1, 2
	case "read", "readBE":
		low, high = 2, 2
	case "write", "writeBE":
		low, high = 3, 3
	case "text", "hex", "toArray":
	default:
		return newError(line, column, "bytes have no method %q%s", method, suggestion(method, bytesMethods))
	}
	if len(args) < low || len(args) > high {
		want := strconv.Itoa(low)
		if high > low {
			want += " or " + strconv.Itoa(high)
		}
		return wrongArgs(line, column, "bytes."+method+"()", len(args), want)
	}
	b := data.Value
	switch method {
	case "slice":
		start, end, err := sliceBounds(args, len(b), line, column)
		if err != nil {
			return err
		}
		return &object.Bytes{Value: append([]byte(nil), b[start:end]...)}
	case "text":
		return &object.String{Value: strings.ToValidUTF8(string(b), string(utf8.RuneError))}
	case "hex":
		return &object.String{Value: object.Hex(b, false)}
	case "toArray":
		elements := make([]object.Object, len(b))
		for i, c := range b {
			elements[i] = &object.Integer{Value: int64(c)}
		}
		return &object.Array{Elements: elements}
	}

	// read, readBE, write and writeBE: a number of size bytes at a position
	at, okAt := args[0].(*object.Integer)
	size, okSize := args[1].(*object.Integer)
	if !okAt || !okSize || size.Value != 1 && size.Value != 2 && size.Value != 4 && size.Value != 8 {
		return errorText(line, column, "bytes."+method+"() takes a position and a size of 1, 2, 4 or 8 bytes")
	}
	if at.Value < 0 || at.Value+size.Value > int64(len(b)) {
		return errorText(line, column, "bytes."+method+"() at "+strconv.Itoa(int(at.Value))+" with "+strconv.Itoa(int(size.Value))+" bytes is outside the "+strconv.Itoa(len(b))+" bytes")
	}
	field := b[at.Value : at.Value+size.Value]
	bigEndian := strings.HasSuffix(method, "BE")
	if method[0] == 'r' {
		var n uint64
		for i := range field {
			if bigEndian {
				n = n<<8 | uint64(field[i])
			} else {
				n |= uint64(field[i]) << (8 * i)
			}
		}
		return &object.Integer{Value: int64(n)}
	}
	v, ok := args[2].(*object.Integer)
	bits := 8 * size.Value
	if !ok || bits < 64 && (v.Value >= 1<<bits || v.Value < -(1<<(bits-1))) {
		return errorText(line, column, args[2].Inspect()+" doesn't fit in "+strconv.Itoa(int(size.Value))+" byte(s)")
	}
	for i := range field {
		shift := 8 * i
		if bigEndian {
			shift = 8 * (len(field) - 1 - i)
		}
		field[i] = byte(uint64(v.Value) >> shift)
	}
	return data
}

// sliceBounds returns the start and end that slice(start, end) takes, as
// arrays do: negative ones count from the end and both are kept inside.
func sliceBounds(args []object.Object, length, line, column int) (int, int, *object.Error) {
	bounds := []int64{0, int64(length)}
	for i, arg := range args {
		n, ok := arg.(*object.Integer)
		if !ok {
			return 0, 0, wrongType(line, column, "arguments to slice()", "an integer", arg)
		}
		bounds[i] = clampSliceIndex(n.Value, int64(length))
	}
	return int(min(bounds[0], bounds[1])), int(bounds[1]), nil
}
