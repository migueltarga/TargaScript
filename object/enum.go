package object

import "strings"

const (
	// ENUM_OBJ is the type of the enums a program declares
	ENUM_OBJ = "ENUM"
	// ENUM_VALUE_OBJ is the type of their values
	ENUM_VALUE_OBJ = "ENUM_VALUE"
)

// Enum is an enum a program declares: a name and its values, in order.
type Enum struct {
	Name   string
	Values []*EnumValue
}

// NewEnum returns the enum with a value for each of names.
func NewEnum(name string, names []string) *Enum {
	e := &Enum{Name: name}
	for _, n := range names {
		e.Values = append(e.Values, &EnumValue{Enum: e, Name: n})
	}
	return e
}

func (e *Enum) Type() ObjectType { return ENUM_OBJ }
func (e *Enum) Inspect() string  { return "enum " + e.Name }

// Value returns the value named name, or nil.
func (e *Enum) Value(name string) *EnumValue {
	for _, v := range e.Values {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// Names lists the names of its values, separated by commas.
func (e *Enum) Names() string {
	names := make([]string, len(e.Values))
	for i, v := range e.Values {
		names[i] = v.Name
	}
	return strings.Join(names, ", ")
}

// EnumValue is one value of an enum. Each is equal only to itself.
type EnumValue struct {
	Enum *Enum
	Name string
}

func (v *EnumValue) Type() ObjectType { return ENUM_VALUE_OBJ }
func (v *EnumValue) Inspect() string  { return v.Enum.Name + "." + v.Name }
