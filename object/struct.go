package object

import "strconv"

// STRUCT_OBJ is the type of the structs a program declares.
const STRUCT_OBJ = "STRUCT"

// StructType is a struct a program declares: its name, its fields, the
// function that makes its values, called with the struct as this and a
// value for each field, and its methods. Its values are objects whose shape
// names the struct.
type StructType struct {
	Name        string
	Fields      []string
	Constructor *Closure
	Methods     map[string]*Closure

	// shape is the shape of its values, with their fields
	shape *Shape
}

func (s *StructType) Type() ObjectType { return STRUCT_OBJ }
func (s *StructType) Inspect() string  { return "struct " + s.Name }

// New returns a value of the struct with values for its fields, in order.
func (s *StructType) New(values []Object) *Hash {
	if s.shape == nil {
		shape := &Shape{typ: s}
		for _, name := range s.Fields {
			key := &String{Value: name}
			next := shape.with(key.HashKey(), key)
			if next == nil {
				// No more shapes can be made
				h := &Hash{dict: &dictionary{typ: s}}
				for i, value := range values {
					h.Set(&String{Value: s.Fields[i]}, value)
				}
				return h
			}
			shape = next
		}
		s.shape = shape
	}
	h := shapedHash(len(values))
	h.shape = s.shape
	h.values = append(h.values, values...)
	return h
}

// Call makes a value of the struct with args for its fields, in order.
func (s *StructType) Call(args []Object) Object {
	if len(args) > len(s.Fields) {
		return &Error{Message: s.tooMany(len(args))}
	}
	return s.Constructor.Runner.CallMethod(s.Constructor, s, args)
}

func (s *StructType) tooMany(got int) string {
	return "wrong number of arguments for " + s.Name + ": got " + strconv.Itoa(got) + ", want up to " + strconv.Itoa(len(s.Fields))
}

// StructSpec is what a struct statement declares, as a constant of the
// code that makes the struct.
type StructSpec struct {
	Name    string
	Fields  []string
	Methods []string
}

func (s *StructSpec) Type() ObjectType { return "STRUCT_SPEC" }
func (s *StructSpec) Inspect() string  { return "struct " + s.Name }
