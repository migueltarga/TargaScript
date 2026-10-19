package object

// PatternKind is what a part of a match pattern does with the value it
// gets.
type PatternKind byte

const (
	PatternAny    PatternKind = iota // _ matches anything
	PatternBind                      // a name takes the value
	PatternValue                     // the value must equal one given with the pattern
	PatternRange                     // the value must be between two given with the pattern
	PatternArray                     // an array whose elements match Parts, in order
	PatternObject                    // an object whose property Keys[i] matches Parts[i]
)

// Pattern is the pattern of a match arm that takes the subject apart, like
// `[first, ...rest]` or `{name, age: 0..17}`, compiled as a constant for
// the VM to match against a value. The values it compares with (its
// literals and expressions) come on the stack with it, counted by Values,
// and the names it binds get their values in order, counted by Binds.
type Pattern struct {
	Kind PatternKind

	// Index is which binding takes the value, which of the values to
	// equal, or which value starts the range, with its end after it
	Index int

	Parts []*Pattern
	Keys  *Array // the property (a String) each part of an object pattern matches

	// Rest is nil, or an Any or Bind that takes the elements or properties
	// the parts don't, from `...rest`
	Rest *Pattern

	// On the whole pattern: how many values come with it, and how many
	// names it binds
	Values, Binds int
}

func (p *Pattern) Type() ObjectType { return PATTERN_OBJ }
func (p *Pattern) Inspect() string  { return "pattern" }
