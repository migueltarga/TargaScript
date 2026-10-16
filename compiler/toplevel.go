package compiler

import "github.com/migueltarga/TargaScript/ast"

// The ways a program uses a global
const (
	useDefine = iota + 1
	useRead
	useAssign
)

// globalUse is how a program uses a global: what it does with it first in
// its own code, and whether a function uses it.
type globalUse struct {
	first      int
	inFunction bool
}

func (c *Compiler) noteGlobal(name string, use int) {
	if c.uses == nil {
		return
	}
	u := c.uses[name]
	if u == nil {
		u = &globalUse{}
		c.uses[name] = u
	}
	if c.fn.outer != nil {
		u.inFunction = true
	} else if u.first == 0 {
		u.first = use
	}
}

// LocalizeTopLevel makes the top-level variables of the next program
// compiled locals of the program instead of globals, which take an
// instruction to read and another to write, when only the program's own
// code uses them, from their declaration on. It compiles the program once
// to find them. Programs whose globals are used after they run, like the
// lines of a REPL or an embedder's scripts, can't use it.
func (c *Compiler) LocalizeTopLevel(program *ast.Program) {
	probe := New()
	probe.Dir, probe.AllowLoad = c.Dir, c.AllowLoad
	probe.uses = map[string]*globalUse{}
	if _, err := probe.Compile(program); err != nil {
		return
	}

	declared := map[string]int{}
	for _, stmt := range program.Statements {
		switch stmt.(type) {
		case *ast.LetStatement, *ast.ConstStatement:
			for _, name := range declaredNames(stmt) {
				declared[name]++
			}
		default:
			// Functions and loaded names stay globals
			for _, name := range declaredNames(stmt) {
				declared[name] = -1
			}
			if load, ok := stmt.(*ast.LoadStatement); ok {
				if load.Alias != nil {
					declared[load.Alias.Value] = -1
				} else if _, names, _ := probe.loadedNames(load); names != nil {
					for _, n := range names {
						declared[n.name] = -1
					}
				}
			}
		}
	}
	c.mainLocals = map[string]bool{}
	for name, n := range declared {
		u := probe.uses[name]
		if _, exists := c.globals.symbols[name]; n > 0 && !exists && u != nil && u.first == useDefine && !u.inFunction {
			c.mainLocals[name] = true
		}
	}
}
