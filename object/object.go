package object

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
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

type HashKey struct {
	Type  ObjectType
	Value uint64
}

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
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string {
	if math.IsInf(f.Value, 1) {
		return "Inf"
	}
	return fmt.Sprintf("%g", f.Value)
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) HashKey() HashKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}

	return HashKey{Type: b.Type(), Value: value}
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type Error struct {
	Message string
	Line    int
	Column  int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("ERROR at line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return "ERROR: " + e.Message
}

// inspectFunction prints a function as its source.
func inspectFunction(name string, parameters []*ast.Identifier, defaults []ast.Expression, rest bool, body *ast.BlockStatement) string {
	var out bytes.Buffer

	params := ast.ParameterStrings(parameters, defaults, rest)

	out.WriteString("fn")
	if name != "" {
		out.WriteString(" " + name)
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(body.String())
	out.WriteString("\n}")

	return out.String()
}

type String struct {
	Value string

	// The hash of Value, computed the first time it's needed. Strings are
	// never changed in place, so it stays valid.
	hash   uint64
	hashed bool
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) HashKey() HashKey {
	if !s.hashed {
		// FNV-1a, inline to avoid allocating a hasher and a byte slice
		h := uint64(14695981039346656037)
		for i := 0; i < len(s.Value); i++ {
			h ^= uint64(s.Value[i])
			h *= 1099511628211
		}
		s.hash, s.hashed = h, true
	}
	return HashKey{Type: STRING_OBJ, Value: s.hash}
}

type Builtin struct {
	Fn   BuiltinFunction
	Name string
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string {
	if b.Name != "" {
		return fmt.Sprintf("builtin function: %s", b.Name)
	}
	return "builtin function"
}

type Array struct {
	Elements []Object
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }
func (ao *Array) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range ao.Elements {
		elements = append(elements, inspectNested(e))
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

type Hash struct {
	Pairs map[HashKey]HashPair
	Order []HashKey // keys of Pairs in insertion order
}

func NewHash() *Hash {
	return &Hash{Pairs: make(map[HashKey]HashPair)}
}

// Set adds or updates a pair, keeping the original position of existing keys.
func (h *Hash) Set(key Hashable, value Object) {
	hashKey := key.HashKey()
	if _, exists := h.Pairs[hashKey]; !exists {
		h.Order = append(h.Order, hashKey)
	}
	h.Pairs[hashKey] = HashPair{Key: key.(Object), Value: value}
}

// OrderedPairs returns the pairs in insertion order.
func (h *Hash) OrderedPairs() []HashPair {
	pairs := make([]HashPair, 0, len(h.Order))
	for _, hashKey := range h.Order {
		pairs = append(pairs, h.Pairs[hashKey])
	}
	return pairs
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, pair := range h.OrderedPairs() {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			inspectNested(pair.Key), inspectNested(pair.Value)))
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// inspectNested formats a value inside an array or hash. Strings are quoted
// there, so ["1", 1] and {"a": "b"} print unambiguously.
func inspectNested(obj Object) string {
	if str, ok := obj.(*String); ok {
		return strconv.Quote(str.Value)
	}
	return obj.Inspect()
}
