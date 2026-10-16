package object

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"unsafe"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/strfmt"
)

type BuiltinFunction func(args ...Object) Object

type ObjectType string

const (
	NULL_OBJ  = "NULL"
	ERROR_OBJ = "ERROR"

	INTEGER_OBJ = "INTEGER"
	FLOAT_OBJ   = "FLOAT"
	BOOLEAN_OBJ = "BOOLEAN"
	STRING_OBJ  = "STRING"

	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"

	ARRAY_OBJ = "ARRAY"
	HASH_OBJ  = "HASH"
)

// HashKey identifies a key of an object: its kind and a number from its
// value. It fits in two words, so finding a key compares little.
type HashKey struct {
	Kind  KeyKind
	Value uint64
}

// KeyKind is the type of value a HashKey comes from.
type KeyKind byte

const (
	IntegerKey KeyKind = iota + 1
	BooleanKey
	StringKey
)

type Hashable interface {
	HashKey() HashKey
}

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return strconv.FormatInt(i.Value, 10) }
func (i *Integer) HashKey() HashKey {
	return HashKey{Kind: IntegerKey, Value: uint64(i.Value)}
}

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string {
	if math.IsInf(f.Value, 1) {
		return "Inf"
	}
	return strfmt.Sprintf("%g", f.Value)
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return strconv.FormatBool(b.Value) }
func (b *Boolean) HashKey() HashKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}

	return HashKey{Kind: BooleanKey, Value: value}
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type Error struct {
	Message string
	Line    int
	Column  int

	// Trace is the calls that led to the error, the innermost first, for
	// errors no try caught
	Trace []Call
}

// Call is a function in the trace of an error, with the line it was at.
// The top level of a program has no name.
type Call struct {
	Function string
	Line     int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	if e.Line > 0 && e.Column > 0 {
		return strfmt.Sprintf("ERROR at line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return "ERROR: " + e.Message
}

// inspectFunction shows a function by its name and parameters: fn add(a, b).
func inspectFunction(name string, parameters []*ast.Identifier, defaults []ast.Expression, rest bool) string {
	params := ast.ParameterStrings(parameters, defaults, rest)
	if name == "" {
		return "fn(" + strings.Join(params, ", ") + ")"
	}
	return "fn " + name + "(" + strings.Join(params, ", ") + ")"
}

type String struct {
	Value string

	// The hash of Value, computed the first time it's needed, or 0 before.
	// Strings are never changed in place, so it stays valid. A hash that
	// is 0 is computed every time, which keeps String small.
	hash uint64

	// buffer holds Value, and room after it, when the string was made by
	// Concat; see there
	buffer *[]byte
}

// Concat returns s followed by more. A string built by adding to it again
// and again, like html += "...", would copy everything each time, so the
// result keeps its text in a buffer with room to spare: adding to the
// newest string of a buffer writes after its end, and only the bytes
// added are copied. Existing strings never change, since each one only
// sees its own part of the buffer.
func Concat(s *String, more string) *String {
	if more == "" {
		return s
	}
	if s.buffer != nil && len(*s.buffer) == len(s.Value) && cap(*s.buffer)-len(*s.buffer) >= len(more) {
		buffer := append(*s.buffer, more...)
		*s.buffer = buffer
		return &String{Value: unsafe.String(&buffer[0], len(buffer)), buffer: s.buffer}
	}
	// Small strings are copied as usual; a buffer pays off for long ones
	if len(s.Value)+len(more) < 64 {
		return &String{Value: s.Value + more}
	}
	buffer := make([]byte, 0, 2*(len(s.Value)+len(more)))
	buffer = append(append(buffer, s.Value...), more...)
	return &String{Value: unsafe.String(&buffer[0], len(buffer)), buffer: &buffer}
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) HashKey() HashKey {
	if s.hash == 0 {
		// FNV-1a, inline to avoid allocating a hasher and a byte slice
		h := uint64(14695981039346656037)
		for i := 0; i < len(s.Value); i++ {
			h ^= uint64(s.Value[i])
			h *= 1099511628211
		}
		s.hash = h
	}
	return HashKey{Kind: StringKey, Value: s.hash}
}

type Builtin struct {
	Fn   BuiltinFunction
	Name string
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string {
	if b.Name != "" {
		return strfmt.Sprintf("builtin function: %s", b.Name)
	}
	return "builtin function"
}

type Array struct {
	Elements []Object
}

// NewArray returns an array of n elements, all nil, which for small arrays
// come in the same allocation as the array.
func NewArray(n int) *Array {
	switch n {
	case 1:
		b := &struct {
			a Array
			e [1]Object
		}{}
		b.a.Elements = b.e[:]
		return &b.a
	case 2:
		b := &struct {
			a Array
			e [2]Object
		}{}
		b.a.Elements = b.e[:]
		return &b.a
	case 3:
		b := &struct {
			a Array
			e [3]Object
		}{}
		b.a.Elements = b.e[:]
		return &b.a
	case 4:
		b := &struct {
			a Array
			e [4]Object
		}{}
		b.a.Elements = b.e[:]
		return &b.a
	}
	return &Array{Elements: make([]Object, n)}
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }
func (ao *Array) Inspect() string  { return ao.inspect(nil) }

// inspect formats the array inside the arrays and hashes in parents, the
// ones that contain it. An array inside itself shows as [...].
func (ao *Array) inspect(parents []Object) string {
	if contains(parents, ao) {
		return "[...]"
	}
	parents = append(parents, ao)

	var out bytes.Buffer

	elements := []string{}
	for _, e := range ao.Elements {
		elements = append(elements, inspectNested(e, parents))
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type HashPair struct {
	Key   Object
	Value Object
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string  { return h.inspect(nil) }

// inspect is like Array.inspect: a hash inside itself shows as {...}.
func (h *Hash) inspect(parents []Object) string {
	if contains(parents, h) {
		return "{...}"
	}
	parents = append(parents, h)

	var out bytes.Buffer

	pairs := []string{}
	for _, pair := range h.OrderedPairs() {
		pairs = append(pairs, strfmt.Sprintf("%s: %s",
			inspectNested(pair.Key, parents), inspectNested(pair.Value, parents)))
	}

	if typ := h.Struct(); typ != nil {
		out.WriteString(typ.Name + " ")
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// inspectNested formats a value inside an array or hash. Strings are quoted
// there, so ["1", 1] and {"a": "b"} print unambiguously.
func inspectNested(obj Object, parents []Object) string {
	switch obj := obj.(type) {
	case *String:
		return strconv.Quote(obj.Value)
	case *Array:
		return obj.inspect(parents)
	case *Hash:
		return obj.inspect(parents)
	}
	return obj.Inspect()
}

func contains(objects []Object, obj Object) bool {
	for _, o := range objects {
		if o == obj {
			return true
		}
	}
	return false
}
