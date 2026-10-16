package object

import (
	"sort"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/code"
)

const (
	COMPILED_FUNCTION_OBJ = "COMPILED_FUNCTION"
	CELL_OBJ              = "CELL"
)

// Position maps the instruction at Offset to the line and column of the
// code it was compiled from.
type Position struct {
	Offset int
	Line   int
	Column int
}

// CompiledFunction is a function compiled to register code. It is a
// constant; running code works with closures of it.
type CompiledFunction struct {
	// Code is what the VM runs, and Positions, sorted by instruction, are
	// where in the source its instructions come from, for errors
	Code      []code.Word
	Positions []Position

	// Caches has, for each instruction that reads or writes properties,
	// where it found them last. A VM runs one instruction at a time, so
	// they need no lock.
	Caches []PropertyCache

	// NumRegisters is how many registers the function uses: its locals,
	// then the values it works on
	NumRegisters int
	NumLocals    int
	NumParams    int
	Name         string

	// Variadic is set when the last parameter collects the extra arguments
	// in an array
	Variadic bool

	// HasThis is set for functions that use this, which have a local, just
	// after the parameters, for it: the object a method was called on, or
	// null
	HasThis bool

	// Names of the locals and captured variables, for error messages
	LocalNames []string
	FreeNames  []string

	// Kept to print the function as its source
	Parameters []*ast.Identifier
	Defaults   []ast.Expression
	Rest       bool
	Body       *ast.BlockStatement
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string {
	return inspectFunction(cf.Name, cf.Parameters, cf.Defaults, cf.Rest)
}

// PositionAt returns the line and column of the instruction at offset, or
// zeros if unknown.
func (cf *CompiledFunction) PositionAt(offset int) (int, int) {
	positions := cf.Positions
	i := sort.Search(len(positions), func(i int) bool { return positions[i].Offset > offset })
	if i == 0 {
		return 0, 0
	}
	p := positions[i-1]
	return p.Line, p.Column
}

// ClosureRunner runs closures for code outside the virtual machine, such as
// builtins that take a function.
type ClosureRunner interface {
	CallClosure(closure *Closure, args []Object) Object

	// CallClosureAsync is like CallClosure, but the closure can pause at
	// an await, and then the result is a promise of what it returns.
	CallClosureAsync(closure *Closure, args []Object) Object
}

// Closure is a compiled function with the variables it captured. Scripts see
// it as a function.
type Closure struct {
	Fn     *CompiledFunction
	Free   []*Cell
	Runner ClosureRunner
}

func (c *Closure) Type() ObjectType { return FUNCTION_OBJ }
func (c *Closure) Inspect() string  { return c.Fn.Inspect() }

// Cell holds a variable captured by a closure, shared by the function that
// declared it and every closure that uses it.
type Cell struct {
	Value Object
}

func (c *Cell) Type() ObjectType { return CELL_OBJ }
func (c *Cell) Inspect() string  { return "cell" }
