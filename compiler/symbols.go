package compiler

import "github.com/migueltarga/TargaScript/ast"

// SymbolScope says where a variable lives.
type SymbolScope int

const (
	GlobalScope  SymbolScope = iota // in the globals, by index
	LocalScope                      // in a slot of the running function
	FreeScope                       // captured from an enclosing function
	BuiltinScope                    // a builtin function
)

// Symbol is a variable known to the compiler.
type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
	Const bool

	// Cell is set for locals captured by a closure, which are stored in a
	// cell so every function using them shares the variable.
	Cell bool

	// Declaration number of a local within its function, used to remember
	// which locals are captured when the function is compiled again
	decl int
	fn   *functionScope
}

// GlobalTable holds the globals, shared by every program compiled with it so
// a REPL keeps its variables between lines.
type GlobalTable struct {
	symbols map[string]*Symbol
	names   []string
}

func NewGlobalTable() *GlobalTable {
	return &GlobalTable{symbols: map[string]*Symbol{}}
}

// define returns the global for name, creating it if needed.
func (g *GlobalTable) define(name string) *Symbol {
	if s, ok := g.symbols[name]; ok {
		return s
	}
	s := &Symbol{Name: name, Scope: GlobalScope, Index: len(g.names)}
	g.symbols[name] = s
	g.names = append(g.names, name)
	return s
}

// Define declares a global and returns its index, for values the host
// provides before running code.
func (g *GlobalTable) Define(name string) int {
	return g.define(name).Index
}

// Names returns the names of the globals, by index.
func (g *GlobalTable) Names() []string {
	return g.names
}

// functionScope tracks the variables of one function being compiled: its
// nested blocks, its slots and the variables it captures.
type functionScope struct {
	outer  *functionScope
	blocks []map[string]*Symbol

	numLocals  int
	localNames []string
	decls      int

	free []*Symbol // symbols of the enclosing function, in capture order

	// captured lists the declarations of this function that a closure
	// captures, and recompile is set when one is found after its code was
	// compiled without a cell
	captured  map[int]bool
	recompile bool

	loops int // loops open in this function, for break and continue
	tries []openTry
}

// openTry is a try block whose handler is open at the code being compiled,
// for return, break and continue to close it and run its finally block.
type openTry struct {
	finally *ast.BlockStatement
	loops   int
}

func newFunctionScope(outer *functionScope, captured map[int]bool) *functionScope {
	return &functionScope{outer: outer, blocks: []map[string]*Symbol{{}}, captured: captured}
}

func (f *functionScope) pushBlock() { f.blocks = append(f.blocks, map[string]*Symbol{}) }
func (f *functionScope) popBlock()  { f.blocks = f.blocks[:len(f.blocks)-1] }

// currentBlock returns the innermost block.
func (f *functionScope) currentBlock() map[string]*Symbol {
	return f.blocks[len(f.blocks)-1]
}

// defineLocal declares name in the innermost block with a new slot.
func (f *functionScope) defineLocal(name string, isConst bool) *Symbol {
	s := &Symbol{Name: name, Scope: LocalScope, Index: f.numLocals, Const: isConst, decl: f.decls, fn: f}
	s.Cell = f.captured[s.decl]
	f.decls++
	f.numLocals++
	f.localNames = append(f.localNames, name)
	f.currentBlock()[name] = s
	return s
}

// lookupLocal finds name in this function's blocks.
func (f *functionScope) lookupLocal(name string) (*Symbol, bool) {
	for i := len(f.blocks) - 1; i >= 0; i-- {
		if s, ok := f.blocks[i][name]; ok {
			return s, true
		}
	}
	return nil, false
}

// resolve finds name in this function or an enclosing one, capturing it
// when it belongs to an enclosing function.
func (f *functionScope) resolve(name string) (*Symbol, bool) {
	if s, ok := f.lookupLocal(name); ok {
		return s, true
	}
	for _, s := range f.free {
		if s.Name == name {
			return &Symbol{Name: name, Scope: FreeScope, Index: indexOf(f.free, s), Const: s.Const}, true
		}
	}
	if f.outer == nil {
		return nil, false
	}

	outer, ok := f.outer.resolve(name)
	if !ok {
		return nil, false
	}
	if outer.Scope == LocalScope {
		outer.fn.capture(outer)
	}
	f.free = append(f.free, outer)
	return &Symbol{Name: name, Scope: FreeScope, Index: len(f.free) - 1, Const: outer.Const}, true
}

// capture marks a local as captured by a closure.
func (f *functionScope) capture(s *Symbol) {
	if s.Cell {
		return
	}
	s.Cell = true
	f.captured[s.decl] = true
	f.recompile = true
}

func indexOf(symbols []*Symbol, s *Symbol) int {
	for i, x := range symbols {
		if x == s {
			return i
		}
	}
	return -1
}
