// Package compiler turns a program into bytecode for the virtual machine.
package compiler

import (
	"cmp"
	"slices"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/strfmt"
	"github.com/migueltarga/TargaScript/token"
)

// Bytecode is a compiled program: its main function and the constants the
// instructions refer to.
type Bytecode struct {
	Main      *object.CompiledFunction
	Constants []object.Object
}

// Compiler compiles programs. Programs compiled with the same Compiler share
// their globals and constants, as the lines of a REPL do.
type Compiler struct {
	constants []object.Object
	globals   *GlobalTable

	// Globals whose declaration was already compiled, to detect constants
	// declared twice
	initialized map[string]bool

	fn    *functionScope
	scope *compilation

	// Captured locals of each function, kept across recompilations
	captured map[ast.Node]map[int]bool

	// hoisted are the functions declared before their block runs, so the
	// functions in it can call each other in any order
	hoisted map[*Symbol]bool

	builtinIndex map[string]int

	// Jumps out of the property chain being compiled, made by ?. when its
	// left side is null, to patch to the end of the chain
	chainJumps *[]int

	// Dir is the directory of the file being compiled, which load paths
	// are relative to. Empty means the working directory.
	Dir string

	// AllowLoad controls whether source can load files. Embedders disable it.
	AllowLoad bool

	// mainLocals are the top-level variables that are locals of the
	// program instead of globals; see LocalizeTopLevel
	mainLocals map[string]bool
	// uses records how the program uses its globals, for LocalizeTopLevel
	uses map[string]*globalUse

	// Errors found while compiling, which the program raises if it gets
	// to them
	problems []Problem
}

// Problem is an error the compiler found in the code, like a name that isn't
// defined. The compiled program raises it when it runs that code, but editors
// can show it before.
type Problem struct {
	Line    int
	Column  int
	Message string
}

// Problems returns the errors found by the last Compile, in source order.
func (c *Compiler) Problems() []Problem {
	seen := map[Problem]bool{}
	problems := []Problem{}
	// Functions compiled more than once report their problems again
	for _, p := range c.problems {
		if !seen[p] {
			seen[p] = true
			problems = append(problems, p)
		}
	}
	slices.SortStableFunc(problems, func(a, b Problem) int {
		if a.Line != b.Line {
			return cmp.Compare(a.Line, b.Line)
		}
		return cmp.Compare(a.Column, b.Column)
	})
	return problems
}

// compilation is the code of the function being compiled.
type compilation struct {
	instructions code.Instructions
	positions    []object.Position
	caches       int // property caches given out
	outer        *compilation
}

// cache gives out a property cache for an instruction of this function.
func (c *Compiler) cache() int {
	c.scope.caches++
	return c.scope.caches - 1
}

func New() *Compiler {
	c := &Compiler{
		globals:      NewGlobalTable(),
		initialized:  map[string]bool{},
		captured:     map[ast.Node]map[int]bool{},
		hoisted:      map[*Symbol]bool{},
		builtinIndex: map[string]int{},
		AllowLoad:    true,
	}
	for i, name := range evaluator.BuiltinNames() {
		c.builtinIndex[name] = i
	}
	return c
}

// Globals returns the globals of the compiled programs.
func (c *Compiler) Globals() *GlobalTable {
	return c.globals
}

// Compile compiles a program into its main function.
func (c *Compiler) Compile(program *ast.Program) (*Bytecode, error) {
	c.problems = nil

	// Names declared at the top level are globals from the start, so
	// functions can use globals defined after them
	for _, stmt := range program.Statements {
		for _, name := range declaredNames(stmt) {
			// The program's locals keep their name here too, for the
			// names errors suggest
			c.globals.define(name)
		}
		if load, ok := stmt.(*ast.LoadStatement); ok {
			if load.Alias != nil {
				c.globals.define(load.Alias.Value)
			} else if _, names, _ := c.loadedNames(load); names != nil {
				for _, n := range names {
					c.globals.define(n.name)
				}
			}
		}
	}

	main, err := c.compileFunction(program, "", nil, nil, nil, func() error {
		return c.compileStatements(program.Statements)
	})
	if err != nil {
		return nil, err
	}
	return &Bytecode{Main: main, Constants: c.constants}, nil
}

// declaredNames returns the names a statement declares.
func declaredNames(stmt ast.Statement) []string {
	switch stmt := stmt.(type) {
	case *ast.LetStatement:
		return bindingNames(stmt.Name, stmt.Pattern)
	case *ast.ConstStatement:
		return bindingNames(stmt.Name, stmt.Pattern)
	case *ast.FunctionStatement:
		return []string{stmt.Name.Value}
	}
	return nil
}

func bindingNames(name *ast.Identifier, pattern *ast.Pattern) []string {
	if pattern == nil {
		return []string{name.Value}
	}
	names := []string{}
	for _, n := range pattern.Names {
		names = append(names, n.Value)
	}
	if pattern.Rest != nil {
		names = append(names, pattern.Rest.Value)
	}
	return names
}

// compileFunction compiles a function body, compiling it again when a
// closure in it turns out to capture a local the body already used without a
// cell.
func (c *Compiler) compileFunction(node ast.Node, name string, params []*ast.Identifier, defaults []ast.Expression, patterns []*ast.Pattern, body func() error) (*object.CompiledFunction, error) {
	captured := c.captured[node]
	if captured == nil {
		captured = map[int]bool{}
		c.captured[node] = captured
	}

	constants := len(c.constants)
	initialized := map[string]bool{}
	for name := range c.initialized {
		initialized[name] = true
	}

	for attempt := 0; ; attempt++ {
		c.constants = c.constants[:constants]
		c.initialized = map[string]bool{}
		for name := range initialized {
			c.initialized[name] = true
		}
		c.fn = newFunctionScope(c.fn, captured)
		if lit, ok := node.(*ast.FunctionLiteral); ok {
			c.fn.implicitIt = lit.ImplicitIt
		}
		c.scope = &compilation{outer: c.scope}

		for _, p := range params {
			s := c.fn.defineLocal(p.Value, false)
			if s.Cell {
				// Parameters arrive as plain values; move captured ones into cells
				c.emit(token.Token{}, code.OpGetLocal, s.Index)
				c.emit(token.Token{}, code.OpNewCell, s.Index)
				c.emit(token.Token{}, code.OpPop)
			}
		}

		// Functions have their own this, set when they're called as a
		// method; arrow functions see the one around them
		hasThis := !isArrow(node)
		if hasThis {
			s := c.fn.defineLocal("this", true)
			if s.Cell {
				c.emit(token.Token{}, code.OpGetLocal, s.Index)
				c.emit(token.Token{}, code.OpNewCell, s.Index)
				c.emit(token.Token{}, code.OpPop)
			}
		}

		if err := c.compileDefaults(params, defaults); err != nil {
			return nil, err
		}
		for i, pattern := range patterns {
			if pattern != nil {
				s, _ := c.fn.lookupLocal(params[i].Value)
				c.emit(token.Token{}, code.OpGetLocal, s.Index)
				c.emitDestructure(pattern, false)
				c.emit(token.Token{}, code.OpPop)
			}
		}
		if err := body(); err != nil {
			return nil, err
		}
		c.emit(token.Token{}, code.OpReturnValue)

		fn := c.fn
		scope := c.scope
		c.fn = fn.outer
		c.scope = scope.outer

		if fn.recompile && attempt < 10 {
			continue
		}

		words, positions, registers, err := translate(scope.instructions, scope.positions, fn.numLocals, fn.cells, c.constants)
		if err != nil {
			return nil, strfmt.Errorf("internal error in %s: %w", functionName(name), err)
		}
		compiled := &object.CompiledFunction{
			Code:         words,
			Positions:    positions,
			Caches:       make([]object.PropertyCache, len(words)),
			NumRegisters: registers,
			NumLocals:    fn.numLocals,
			NumParams:    len(params),
			HasThis:      hasThis && fn.usesThis,
			SimpleCall:   len(params),
			Name:         name,
			LocalNames:   fn.localNames,
		}
		for _, s := range fn.free {
			compiled.FreeNames = append(compiled.FreeNames, s.Name)
		}
		if c.fn != nil {
			c.emitClosure(compiled, fn.free)
		}
		return compiled, nil
	}
}

// isArrow reports whether a function is an arrow function, x => ...
func isArrow(node ast.Node) bool {
	fn, ok := node.(*ast.FunctionLiteral)
	return ok && fn.Arrow
}

// compileDefaults gives parameters their default value when they are null,
// because their argument is missing or null.
func (c *Compiler) compileDefaults(params []*ast.Identifier, defaults []ast.Expression) error {
	for i, value := range defaults {
		if value == nil {
			continue
		}
		s, _ := c.fn.lookupLocal(params[i].Value)
		c.emit(token.Token{}, code.OpGetLocal, s.Index)
		skip := c.emit(token.Token{}, code.OpJumpNotNullKeep, 9999)
		if err := c.compileExpression(value); err != nil {
			return err
		}
		c.emit(token.Token{}, code.OpSetLocal, s.Index)
		c.changeOperands(skip, len(c.scope.instructions))
		c.emit(token.Token{}, code.OpPop)
	}
	return nil
}

// emitClosure pushes the cells a closure captures and creates it.
func (c *Compiler) emitClosure(fn *object.CompiledFunction, free []*Symbol) {
	for _, s := range free {
		switch s.Scope {
		case LocalScope:
			c.emit(token.Token{}, code.OpGetLocalCell, s.Index)
		case FreeScope:
			c.emit(token.Token{}, code.OpGetFreeCell, s.Index)
		}
	}
	c.emit(token.Token{}, code.OpClosure, c.addConstant(fn), len(free))
}

// compileStatements compiles statements so that they leave the value of the
// last one on the stack, or null when there are none.
func (c *Compiler) compileStatements(statements []ast.Statement) error {
	c.hoistFunctions(statements)
	compiled := 0
	for _, stmt := range statements {
		if stmt == nil {
			continue
		}
		if compiled > 0 {
			c.emit(token.Token{}, code.OpPop)
		}
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
		compiled++
	}
	if compiled == 0 {
		c.emit(token.Token{}, code.OpNull)
	}
	return nil
}

// hoistFunctions declares the functions of a block inside a function before
// its code, as globals already are, so one can call another declared after
// it. Their cells exist from the start, for the closures that capture them.
func (c *Compiler) hoistFunctions(statements []ast.Statement) {
	if c.atTopLevel() {
		return
	}
	for _, stmt := range statements {
		fn, ok := stmt.(*ast.FunctionStatement)
		if !ok {
			continue
		}
		if _, exists := c.fn.currentBlock()[fn.Name.Value]; exists {
			continue
		}
		s := c.define(fn.Name.Value, false)
		c.emitCellBeforeClosure(s)
		c.hoisted[s] = true
	}
}

// compileBlock compiles a block in its own scope.
func (c *Compiler) compileBlock(block *ast.BlockStatement) error {
	c.fn.pushBlock()
	defer c.fn.popBlock()
	return c.compileStatements(block.Statements)
}

func (c *Compiler) compileStatement(stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.ExpressionStatement:
		return c.compileExpression(stmt.Expression)

	case *ast.LetStatement:
		if stmt.Pattern != nil {
			return c.compileDestructuring(stmt.Token, stmt.Pattern, stmt.Value, false)
		}
		return c.compileLet(stmt.Token, stmt.Name.Value, stmt.Value, false)

	case *ast.ConstStatement:
		if stmt.Pattern != nil {
			return c.compileDestructuring(stmt.Token, stmt.Pattern, stmt.Value, true)
		}
		return c.compileLet(stmt.Token, stmt.Name.Value, stmt.Value, true)

	case *ast.FunctionStatement:
		return c.compileFunctionStatement(stmt)

	case *ast.ReturnStatement:
		if stmt.ReturnValue == nil {
			c.emit(token.Token{}, code.OpNull)
		} else if err := c.compileExpression(stmt.ReturnValue); err != nil {
			return err
		}
		if err := c.leaveTries(0); err != nil {
			return err
		}
		c.emit(token.Token{}, code.OpReturnValue)
		return nil

	case *ast.BreakStatement:
		return c.compileLoopControl(stmt.Token, "break", 0)

	case *ast.ContinueStatement:
		return c.compileLoopControl(stmt.Token, "continue", 1)

	case *ast.RepeatStatement:
		return c.compileRepeat(stmt)

	case *ast.ThrowStatement:
		if err := c.compileExpression(stmt.Value); err != nil {
			return err
		}
		c.emit(stmt.Token, code.OpThrow)
		return nil

	case *ast.BlockStatement:
		return c.compileBlock(stmt)

	case *ast.LoadStatement:
		return c.compileLoad(stmt)
	}
	return strfmt.Errorf("cannot compile %s", stmt.String())
}

// compileLet compiles let and const.
func (c *Compiler) compileLet(tok token.Token, name string, value ast.Expression, isConst bool) error {
	existing, inScope := c.lookupCurrentScope(name)

	// A function can call itself by name, so the name exists before the
	// function is compiled
	var target *Symbol
	if _, isFn := value.(*ast.FunctionLiteral); isFn && !inScope {
		target = c.define(name, isConst)
		c.emitCellBeforeClosure(target)
	}

	if err := c.compileExpression(value); err != nil {
		return err
	}

	if inScope && isConst {
		c.emitError(tok, "identifier already declared: %s", name)
		return nil
	}
	if inScope && existing.Const {
		c.emitError(tok, "cannot reassign constant: %s", name)
		return nil
	}

	if target == nil {
		target = c.define(name, isConst)
		c.emitDefine(target)
		return nil
	}
	c.emitAssignDefined(target)
	return nil
}

// compileDestructuring compiles `let [a, b] = value` and `let {a, b} =
// value`. Its value is the whole value, like a let's.
func (c *Compiler) compileDestructuring(tok token.Token, pattern *ast.Pattern, value ast.Expression, isConst bool) error {
	if err := c.compileExpression(value); err != nil {
		return err
	}

	for _, name := range append(append([]*ast.Identifier{}, pattern.Names...), restName(pattern)...) {
		existing, inScope := c.lookupCurrentScope(name.Value)
		if inScope && isConst {
			c.emitError(tok, "identifier already declared: %s", name.Value)
			return nil
		}
		if inScope && existing.Const {
			c.emitError(tok, "cannot reassign constant: %s", name.Value)
			return nil
		}
	}

	c.emitDestructure(pattern, isConst)
	return nil
}

// emitDestructure declares the names of a pattern with the parts of the
// value on top of the stack, keeping it.
func (c *Compiler) emitDestructure(pattern *ast.Pattern, isConst bool) {
	for i, name := range pattern.Names {
		c.emit(token.Token{}, code.OpDup)
		if pattern.Object {
			c.emit(pattern.Token, code.OpGetProperty, c.addConstant(&object.String{Value: pattern.Key(i)}), c.cache())
		} else {
			c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.Integer{Value: int64(i)}))
			c.emit(pattern.Token, code.OpIndex)
		}
		if value := pattern.Default(i); value != nil {
			// Like ??: the default replaces a null
			jump := c.emit(token.Token{}, code.OpJumpNotNullKeep, 9999)
			if err := c.compileExpression(value); err != nil {
				c.emitError(pattern.Token, "%s", err)
			}
			c.changeOperands(jump, len(c.scope.instructions))
		}
		c.emitDefine(c.define(name.Value, isConst))
		c.emit(token.Token{}, code.OpPop)
	}

	if pattern.Rest != nil {
		c.emit(token.Token{}, code.OpDup)
		if pattern.Object {
			names := []object.Object{}
			for i := range pattern.Names {
				names = append(names, &object.String{Value: pattern.Key(i)})
			}
			c.emit(pattern.Token, code.OpWithout, c.addConstant(&object.Array{Elements: names}))
		} else {
			c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.Integer{Value: int64(len(pattern.Names))}))
			c.emit(pattern.Token, code.OpCallMethod, c.addConstant(&object.String{Value: "slice"}), 1, c.cache())
		}
		c.emitDefine(c.define(pattern.Rest.Value, isConst))
		c.emit(token.Token{}, code.OpPop)
	}
}

func (c *Compiler) compileFunctionStatement(stmt *ast.FunctionStatement) error {
	name := stmt.Name.Value
	existing, inScope := c.lookupCurrentScope(name)

	var target *Symbol
	if !inScope {
		target = c.define(name, false)
		c.emitCellBeforeClosure(target)
	} else {
		target = existing
	}

	if _, err := c.compileFunction(stmt, name, stmt.Parameters, stmt.Defaults, stmt.Patterns, func() error {
		return c.compileStatements(stmt.Body.Statements)
	}); err != nil {
		return err
	}
	c.setFunctionSource(stmt.Parameters, stmt.Defaults, stmt.Rest, stmt.Body)

	if inScope && existing.Const {
		c.emitError(stmt.Token, "cannot reassign constant: %s", name)
		return nil
	}
	if inScope && !c.hoisted[target] {
		c.emitDefine(target)
		return nil
	}
	delete(c.hoisted, target)
	c.emitAssignDefined(target)
	return nil
}

// emitCellBeforeClosure creates the cell of a captured local that names a
// function before the function is created, so the function captures the
// cell it will be stored in and can call itself.
func (c *Compiler) emitCellBeforeClosure(s *Symbol) {
	if s.Scope == LocalScope && s.Cell {
		c.emit(token.Token{}, code.OpNull)
		c.emit(token.Token{}, code.OpNewCell, s.Index)
		c.emit(token.Token{}, code.OpPop)
	}
}

// emitAssignDefined stores the value on top of the stack in a variable
// prepared by emitCellBeforeClosure.
func (c *Compiler) emitAssignDefined(s *Symbol) {
	if s.Scope == LocalScope && s.Cell {
		c.emit(token.Token{}, code.OpSetLocal, s.Index)
		return
	}
	c.emitDefine(s)
}

// setFunctionSource records the source of the function just compiled, to
// print it.
func (c *Compiler) setFunctionSource(params []*ast.Identifier, defaults []ast.Expression, rest bool, body *ast.BlockStatement) {
	fn := c.constants[len(c.constants)-1].(*object.CompiledFunction)
	fn.Parameters = params
	fn.Defaults = defaults
	fn.Rest = rest
	fn.Variadic = rest
	if rest {
		fn.SimpleCall = -1
	}
	fn.Body = body
}

// lookupCurrentScope finds a variable declared in the innermost scope.
func (c *Compiler) lookupCurrentScope(name string) (*Symbol, bool) {
	if c.atTopLevel() && !c.mainLocals[name] {
		if !c.initialized[name] {
			return nil, false
		}
		s, ok := c.globals.symbols[name]
		return s, ok
	}
	s, ok := c.fn.currentBlock()[name]
	return s, ok
}

// atTopLevel reports whether declarations create globals: in the program
// itself, outside any block or function.
func (c *Compiler) atTopLevel() bool {
	return c.fn.outer == nil && len(c.fn.blocks) == 1
}

// define declares a variable in the innermost scope.
func (c *Compiler) define(name string, isConst bool) *Symbol {
	if c.atTopLevel() && !c.mainLocals[name] {
		s := c.globals.define(name)
		s.Const = isConst
		c.initialized[name] = true
		return s
	}
	return c.fn.defineLocal(name, isConst)
}

// emitDefine stores the value on top of the stack in a new variable,
// keeping it on the stack.
func (c *Compiler) emitDefine(s *Symbol) {
	switch {
	case s.Scope == GlobalScope:
		c.noteGlobal(s.Name, useDefine)
		c.emit(token.Token{}, code.OpSetGlobal, s.Index)
	case s.Cell:
		c.emit(token.Token{}, code.OpNewCell, s.Index)
	default:
		c.emit(token.Token{}, code.OpSetLocal, s.Index)
	}
}

// knows reports whether a name is a variable, a builtin or a constant of
// the program.
func (c *Compiler) knows(name string) bool {
	if _, ok := c.resolve(name); ok {
		return true
	}
	_, ok := evaluator.Constant(name)
	return ok
}

// itIsVariable reports whether it is a variable of the program where an
// argument uses it, so the argument isn't a function of it. The it of an
// argument around this one doesn't count: the innermost argument has its
// own. A global counts once its declaration is compiled, as globals exist
// before.
func (c *Compiler) itIsVariable() bool {
	for fn := c.fn; fn != nil; fn = fn.outer {
		if fn.implicitIt {
			return false
		}
		for _, block := range fn.blocks {
			if _, ok := block["it"]; ok {
				return true
			}
		}
	}
	_, ok := c.globals.symbols["it"]
	return ok && c.initialized["it"]
}

// resolve finds the variable a name refers to.
func (c *Compiler) resolve(name string) (*Symbol, bool) {
	if s, ok := c.fn.resolve(name); ok {
		return s, true
	}
	if s, ok := c.globals.symbols[name]; ok {
		return s, true
	}
	if i, ok := c.builtinIndex[name]; ok {
		return &Symbol{Name: name, Scope: BuiltinScope, Index: i}, true
	}
	return nil, false
}

// knownNames lists the names visible at this point, to suggest one when a
// name is unknown.
func (c *Compiler) knownNames() []string {
	names := append([]string{}, c.globals.names...)
	for fn := c.fn; fn != nil; fn = fn.outer {
		for _, block := range fn.blocks {
			for name := range block {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names
}

func (c *Compiler) compileLoopControl(tok token.Token, keyword string, kind int) error {
	if c.fn.loops == 0 {
		c.emitError(tok, "%s outside of a loop", keyword)
		return nil
	}
	if err := c.leaveTries(c.fn.loops); err != nil {
		return err
	}
	c.emit(tok, code.OpBreak, kind)
	return nil
}

func (c *Compiler) compileRepeat(stmt *ast.RepeatStatement) error {
	if stmt.Collection == nil {
		c.emitError(stmt.Token, "invalid repeat statement: missing iterator or condition")
		return nil
	}

	if stmt.Iterator == nil {
		// repeat 3 { } runs three times, instead of looping forever on a
		// condition that is always true
		if count, ok := stmt.Collection.(*ast.IntegerLiteral); ok {
			counted := *stmt
			counted.Iterator = &ast.Identifier{Token: count.Token, Value: "\x00count"}
			return c.compileRepeat(&counted)
		}
		return c.compileWhile(stmt)
	}

	if r, ok := stmt.Collection.(*ast.InfixExpression); ok && r.Operator == ".." {
		if err := c.compileExpression(r.Left); err != nil {
			return err
		}
		if err := c.compileExpression(r.Right); err != nil {
			return err
		}
		c.emit(r.Token, code.OpRangeIter)
	} else {
		if err := c.compileExpression(stmt.Collection); err != nil {
			return err
		}
		c.emit(stmt.Token, code.OpIterStart)
	}

	pairs := 0
	if stmt.Key != nil {
		pairs = 1
	}
	enter := c.emit(token.Token{}, code.OpLoopEnter, 9999, 9999)
	next := c.emit(token.Token{}, code.OpIterNext, 9999, pairs)

	c.fn.loops++
	c.fn.pushBlock()
	iterator := c.fn.defineLocal(stmt.Iterator.Value, false)
	c.emitDefine(iterator)
	c.emit(token.Token{}, code.OpPop)
	if stmt.Key != nil {
		key := c.fn.defineLocal(stmt.Key.Value, false)
		c.emitDefine(key)
		c.emit(token.Token{}, code.OpPop)
	}
	if stmt.Pattern != nil {
		c.emit(token.Token{}, code.OpGetLocal, iterator.Index)
		c.emitDestructure(stmt.Pattern, false)
		c.emit(token.Token{}, code.OpPop)
	}
	err := c.compileStatements(stmt.Body.Statements)
	c.fn.popBlock()
	c.fn.loops--
	if err != nil {
		return err
	}
	c.emit(token.Token{}, code.OpPop)
	// The jump back counts a step, which errors point at the loop for
	c.emit(stmt.Token, code.OpJump, next)

	end := c.emit(token.Token{}, code.OpLoopExit)
	c.changeOperands(enter, end, next)
	c.changeOperands(next, end, pairs)
	c.emit(token.Token{}, code.OpPop) // the iterator
	c.emit(token.Token{}, code.OpNull)
	return nil
}

// compileTry compiles a try expression, whose value is the value of its
// body, or of its catch block when the body fails. A finally block runs
// after both, and again before an error it did not catch goes on.
func (c *Compiler) compileTry(stmt *ast.TryExpression) error {
	if stmt.Finally == nil {
		return c.compileTryCatch(stmt)
	}

	try := c.openTry(stmt.Finally)
	var err error
	if stmt.Catch != nil {
		err = c.compileTryCatch(stmt)
	} else {
		err = c.compileBlock(stmt.Body)
	}
	if err != nil {
		return err
	}
	c.closeTry()
	if err := c.compileFinally(stmt.Finally); err != nil {
		return err
	}
	jumpEnd := c.emit(token.Token{}, code.OpJump, 9999)

	// The VM jumps here with the error on the stack, to throw it again
	c.changeOperands(try, len(c.scope.instructions))
	if err := c.compileFinally(stmt.Finally); err != nil {
		return err
	}
	c.emit(token.Token{}, code.OpThrow)

	c.changeOperands(jumpEnd, len(c.scope.instructions))
	return nil
}

func (c *Compiler) compileTryCatch(stmt *ast.TryExpression) error {
	try := c.openTry(nil)
	if err := c.compileBlock(stmt.Body); err != nil {
		return err
	}
	c.closeTry()
	jumpEnd := c.emit(token.Token{}, code.OpJump, 9999)

	// The VM jumps here with the error on the stack
	c.changeOperands(try, len(c.scope.instructions))
	c.fn.pushBlock()
	if stmt.Name != nil {
		c.emitDefine(c.fn.defineLocal(stmt.Name.Value, false))
	}
	c.emit(token.Token{}, code.OpPop)
	err := c.compileStatements(stmt.Catch.Statements)
	c.fn.popBlock()
	if err != nil {
		return err
	}

	c.changeOperands(jumpEnd, len(c.scope.instructions))
	return nil
}

// openTry emits the start of a try block and returns the instruction whose
// operand is where its errors go.
func (c *Compiler) openTry(finally *ast.BlockStatement) int {
	c.fn.tries = append(c.fn.tries, openTry{finally: finally, loops: c.fn.loops})
	return c.emit(token.Token{}, code.OpTry, 9999)
}

func (c *Compiler) closeTry() {
	c.fn.tries = c.fn.tries[:len(c.fn.tries)-1]
	c.emit(token.Token{}, code.OpTryEnd)
}

// hasFinally reports whether one of the tries opened with at least loops
// loops open has a finally block.
func hasFinally(tries []openTry, loops int) bool {
	for i := len(tries) - 1; i >= 0 && tries[i].loops >= loops; i-- {
		if tries[i].finally != nil {
			return true
		}
	}
	return false
}

// compileFinally compiles a finally block, leaving the stack as it was.
func (c *Compiler) compileFinally(block *ast.BlockStatement) error {
	if err := c.compileBlock(block); err != nil {
		return err
	}
	c.emit(token.Token{}, code.OpPop)
	return nil
}

// leaveTries closes the try blocks that return, break or continue jump out
// of, innermost first, and runs their finally blocks. Those blocks are
// compiled as outside the tries they belong to, so that a return in one of
// them does not run it again.
func (c *Compiler) leaveTries(loops int) error {
	tries := c.fn.tries
	if !hasFinally(tries, loops) {
		return nil // the VM closes the tries itself
	}
	defer func() { c.fn.tries = tries }()
	for i := len(tries) - 1; i >= 0 && tries[i].loops >= loops; i-- {
		c.fn.tries = tries[:i]
		c.emit(token.Token{}, code.OpTryEnd)
		if tries[i].finally != nil {
			if err := c.compileFinally(tries[i].finally); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compiler) compileWhile(stmt *ast.RepeatStatement) error {
	enter := c.emit(token.Token{}, code.OpLoopEnter, 9999, 9999)
	start := len(c.scope.instructions)

	if err := c.compileExpression(stmt.Collection); err != nil {
		return err
	}
	exit := c.emit(token.Token{}, code.OpJumpNotTruthy, 9999)

	c.fn.loops++
	err := c.compileBlock(stmt.Body)
	c.fn.loops--
	if err != nil {
		return err
	}
	c.emit(token.Token{}, code.OpPop)
	c.emit(stmt.Token, code.OpJump, start)

	end := c.emit(token.Token{}, code.OpLoopExit)
	c.changeOperands(enter, end, start)
	c.changeOperands(exit, end)
	c.emit(token.Token{}, code.OpNull)
	return nil
}

func (c *Compiler) compileExpression(node ast.Expression) error {
	switch node.(type) {
	case *ast.DotExpression, *ast.MethodCallExpression, *ast.IndexExpression, *ast.CallExpression:
		if c.chainJumps == nil {
			// The outermost access of a chain like a?.b.c(): every ?. in it
			// jumps here when its left side is null
			jumps := []int{}
			c.chainJumps = &jumps
			err := c.compileAccess(node)
			c.chainJumps = nil
			for _, jump := range jumps {
				c.changeOperands(jump, len(c.scope.instructions))
			}
			return err
		}
		return c.compileAccess(node)
	}
	// Other expressions start chains of their own
	return c.compileOperand(node)
}

// compileOperand compiles an expression that isn't part of the chain around
// it, like an argument or an index.
func (c *Compiler) compileOperand(node ast.Expression) error {
	jumps := c.chainJumps
	c.chainJumps = nil
	defer func() { c.chainJumps = jumps }()
	switch node.(type) {
	case *ast.DotExpression, *ast.MethodCallExpression, *ast.IndexExpression, *ast.CallExpression:
		return c.compileExpression(node)
	}
	return c.compileOther(node)
}

// optionalJump makes an access null when its left side is null, for ?.
func (c *Compiler) optionalJump(optional bool) {
	if optional {
		*c.chainJumps = append(*c.chainJumps, c.emit(token.Token{}, code.OpJumpNullKeep, 9999))
	}
}

// compileAccess compiles property and index access and calls, which can be
// part of a chain.
func (c *Compiler) compileAccess(node ast.Expression) error {
	switch node := node.(type) {
	case *ast.IndexExpression:
		if err := c.compileExpression(node.Left); err != nil {
			return err
		}
		c.optionalJump(node.Optional)
		if err := c.compileOperand(node.Index); err != nil {
			return err
		}
		c.emit(node.Token, code.OpIndex)

	case *ast.DotExpression:
		if err := c.compileExpression(node.Left); err != nil {
			return err
		}
		c.optionalJump(node.Optional)
		c.emit(node.Token, code.OpGetProperty, c.addConstant(&object.String{Value: node.Property.Value}), c.cache())

	case *ast.MethodCallExpression:
		if err := c.compileExpression(node.Object); err != nil {
			return err
		}
		c.optionalJump(node.Optional)
		if hasSpread(node.Arguments) {
			if err := c.compileSpreadArgs(node.Token, node.Arguments); err != nil {
				return err
			}
			c.emit(node.Token, code.OpCallMethodArray, c.addConstant(&object.String{Value: node.Method.Value}))
			return nil
		}
		for _, arg := range node.Arguments {
			if err := c.compileOperand(arg); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpCallMethod, c.addConstant(&object.String{Value: node.Method.Value}), len(node.Arguments), c.cache())

	case *ast.CallExpression:
		if name, ok := node.Function.(*ast.Identifier); ok && node.Piped && !c.knows(name.Value) {
			// value |> map(f) is value.map(f)
			return c.compileExpression(&ast.MethodCallExpression{
				Token: node.Token, Object: node.Arguments[0], Method: name, Arguments: node.Arguments[1:],
			})
		}
		if err := c.compileExpression(node.Function); err != nil {
			return err
		}
		if hasSpread(node.Arguments) {
			if err := c.compileSpreadArgs(node.Token, node.Arguments); err != nil {
				return err
			}
			c.emit(node.Token, code.OpCallArray)
			return nil
		}
		for _, arg := range node.Arguments {
			if err := c.compileOperand(arg); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpCall, len(node.Arguments))
	}
	return nil
}

func (c *Compiler) compileOther(node ast.Expression) error {
	switch node := node.(type) {
	case *ast.IntegerLiteral:
		c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.Integer{Value: node.Value}))

	case *ast.FloatLiteral:
		c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.Float{Value: node.Value}))

	case *ast.StringLiteral:
		c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.String{Value: node.Value}))

	case *ast.Boolean:
		if node.Value {
			c.emit(token.Token{}, code.OpTrue)
		} else {
			c.emit(token.Token{}, code.OpFalse)
		}

	case *ast.NullLiteral:
		c.emit(token.Token{}, code.OpNull)

	case *ast.TemplateLiteral:
		for _, part := range node.Parts {
			if err := c.compileExpression(part); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpTemplate, len(node.Parts))

	case *ast.ArrayLiteral:
		if hasSpread(node.Elements) {
			return c.compileSpreadList(node.Token, node.Elements)
		}
		for _, el := range node.Elements {
			if err := c.compileExpression(el); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpArray, len(node.Elements))

	case *ast.SpreadExpression:
		c.emitError(node.Token, "... can only be used in arrays, objects and the arguments of calls")

	case *ast.HashLiteral:
		if hashHasSpread(node) {
			return c.compileSpreadHash(node)
		}
		for _, key := range node.Order {
			pair := node.Pairs[key]
			if parser.PatternOnly(pair) {
				c.emitError(node.Token, "{%s = ...} gives a default to a name in a pattern; in an object, write {%s: ...}", key, key)
				return nil
			}
			if ident, ok := pair.Key.(*ast.Identifier); ok {
				// Bare keys are names, like in `{name: "Ana"}`
				c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.String{Value: ident.Value}))
			} else if err := c.compileExpression(pair.Key); err != nil {
				return err
			}
			if err := c.compileExpression(pair.Value); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpHash, len(node.Order), c.cache())

	case *ast.PrefixExpression:
		if err := c.compileExpression(node.Right); err != nil {
			return err
		}
		switch node.Operator {
		case "!":
			c.emit(node.Token, code.OpBang)
		case "-":
			c.emit(node.Token, code.OpMinus)
		default:
			return strfmt.Errorf("unknown prefix operator %s", node.Operator)
		}

	case *ast.InfixExpression:
		return c.compileInfix(node)

	case *ast.IfExpression:
		return c.compileIf(node)

	case *ast.Identifier:
		c.compileGet(node.Token, node.Value)

	case *ast.AssignmentExpression:
		return c.compileAssignment(node)

	case *ast.PostfixExpression:
		return c.compilePostfix(node)

	case *ast.TryExpression:
		return c.compileTry(node)

	case *ast.MatchExpression:
		return c.compileMatch(node)

	case *ast.FormatExpression:
		// format("%spec", value), with the builtin even if a variable is
		// named format
		c.emit(token.Token{}, code.OpGetBuiltin, c.builtinIndex["format"])
		c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.String{Value: "%" + node.Spec}))
		if err := c.compileOperand(node.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpCall, 2)

	case *ast.ThrowExpression:
		if err := c.compileExpression(node.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpThrow)

	case *ast.AwaitExpression:
		if err := c.compileExpression(node.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpAwait)

	case *ast.FunctionLiteral:
		if node.ImplicitIt && c.itIsVariable() {
			// The argument is the value of the variable it
			return c.compileExpression(node.Body.Statements[0].(*ast.ExpressionStatement).Expression)
		}
		name := ""
		if node.Name != nil {
			name = node.Name.Value
		}
		if _, err := c.compileFunction(node, name, node.Parameters, node.Defaults, node.Patterns, func() error {
			return c.compileStatements(node.Body.Statements)
		}); err != nil {
			return err
		}
		c.setFunctionSource(node.Parameters, node.Defaults, node.Rest, node.Body)

	default:
		return strfmt.Errorf("cannot compile %s", node.String())
	}
	return nil
}

var infixOpcodes = map[string]code.Opcode{
	"+":  code.OpAdd,
	"-":  code.OpSub,
	"*":  code.OpMul,
	"/":  code.OpDiv,
	"%":  code.OpMod,
	"**": code.OpPow,
	"^":  code.OpXor,
	"in": code.OpIn,
	"==": code.OpEqual,
	"!=": code.OpNotEqual,
	">":  code.OpGreaterThan,
	">=": code.OpGreaterEqual,
	"<":  code.OpLessThan,
	"<=": code.OpLessEqual,
	"..": code.OpRange,
}

func (c *Compiler) compileInfix(node *ast.InfixExpression) error {
	if node.Operator == "&&" || node.Operator == "||" || node.Operator == "??" {
		if err := c.compileExpression(node.Left); err != nil {
			return err
		}
		jump := code.OpJumpNotTruthyKeep
		switch node.Operator {
		case "||":
			jump = code.OpJumpTruthyKeep
		case "??":
			jump = code.OpJumpNotNullKeep
		}
		pos := c.emit(token.Token{}, jump, 9999)
		if err := c.compileExpression(node.Right); err != nil {
			return err
		}
		c.changeOperands(pos, len(c.scope.instructions))
		return nil
	}

	if r, ok := node.Right.(*ast.InfixExpression); ok && node.Operator == "in" && r.Operator == ".." {
		// x in a..b compares with the ends instead of building the range
		for _, operand := range []ast.Expression{node.Left, r.Left, r.Right} {
			if err := c.compileExpression(operand); err != nil {
				return err
			}
		}
		c.emit(node.Token, code.OpInRange)
		return nil
	}

	op, ok := infixOpcodes[node.Operator]
	if !ok {
		return strfmt.Errorf("unknown operator %s", node.Operator)
	}
	if err := c.compileExpression(node.Left); err != nil {
		return err
	}
	if err := c.compileExpression(node.Right); err != nil {
		return err
	}
	c.emit(node.Token, op)
	return nil
}

func (c *Compiler) compileIf(node *ast.IfExpression) error {
	if err := c.compileExpression(node.Condition); err != nil {
		return err
	}
	jumpElse := c.emit(token.Token{}, code.OpJumpNotTruthy, 9999)

	if err := c.compileBlock(node.Consequence); err != nil {
		return err
	}
	jumpEnd := c.emit(token.Token{}, code.OpJump, 9999)

	c.changeOperands(jumpElse, len(c.scope.instructions))
	if node.Alternative == nil {
		c.emit(token.Token{}, code.OpNull)
	} else if err := c.compileBlock(node.Alternative); err != nil {
		return err
	}
	c.changeOperands(jumpEnd, len(c.scope.instructions))
	return nil
}

// compileGet pushes the value of a variable.
func (c *Compiler) compileGet(tok token.Token, name string) {
	if name == "this" {
		for fn := c.fn; fn != nil; fn = fn.outer {
			if _, own := fn.lookupLocal("this"); own {
				fn.usesThis = true
				break
			}
		}
	}
	s, ok := c.resolve(name)
	if !ok && name == "this" {
		// Outside functions, this is null
		c.emit(tok, code.OpNull)
		return
	}
	if value, isConstant := evaluator.Constant(name); !ok && isConstant {
		c.emit(tok, code.OpConstant, c.addConstant(value))
		return
	}
	if !ok {
		c.emitNotDefined(tok, name)
		return
	}
	switch s.Scope {
	case GlobalScope:
		c.noteGlobal(s.Name, useRead)
		c.emit(tok, code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(tok, code.OpGetLocal, s.Index)
	case FreeScope:
		c.emit(tok, code.OpGetFree, s.Index)
	case BuiltinScope:
		c.emit(tok, code.OpGetBuiltin, s.Index)
	}
}

// compileSet stores the value on top of the stack in an existing variable,
// keeping it.
func (c *Compiler) compileSet(tok token.Token, name string) {
	s, ok := c.resolve(name)
	if !ok || s.Scope == BuiltinScope {
		c.emitNotDefined(tok, name)
		return
	}
	if s.Const {
		c.emitError(tok, "cannot reassign constant: %s", name)
		return
	}
	switch s.Scope {
	case GlobalScope:
		c.noteGlobal(s.Name, useAssign)
		c.emit(tok, code.OpSetGlobal, s.Index)
	case LocalScope:
		c.emit(tok, code.OpSetLocal, s.Index)
	case FreeScope:
		c.emit(tok, code.OpSetFree, s.Index)
	}
}

func (c *Compiler) emitNotDefined(tok token.Token, name string) {
	err := evaluator.NotDefined(tok.Line, tok.Column, name, c.knownNames())
	c.emitError(tok, "%s", err.Message)
}

func (c *Compiler) compileAssignment(node *ast.AssignmentExpression) error {
	if node.Operator != "" {
		return c.compileCompoundAssignment(node)
	}
	switch target := node.Left.(type) {
	case *ast.Identifier:
		if err := c.compileExpression(node.Value); err != nil {
			return err
		}
		c.compileSet(node.Token, target.Value)

	case *ast.IndexExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		if err := c.compileExpression(target.Index); err != nil {
			return err
		}
		if err := c.compileExpression(node.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpSetIndex)

	case *ast.DotExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		if err := c.compileExpression(node.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpSetProperty, c.addConstant(&object.String{Value: target.Property.Value}), c.cache())

	default:
		c.emitError(node.Token, "invalid assignment target: %s", node.Left.String())
	}
	return nil
}

func (c *Compiler) compilePostfix(node *ast.PostfixExpression) error {
	decrement := 0
	if node.Operator == "--" {
		decrement = 1
	}

	switch target := node.Left.(type) {
	case *ast.Identifier:
		c.compileGet(target.Token, target.Value)
		c.emit(node.Token, code.OpIncrement, decrement)
		c.compileSet(node.Token, target.Value)

	case *ast.IndexExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		if err := c.compileExpression(target.Index); err != nil {
			return err
		}
		c.emit(token.Token{}, code.OpDup2)
		c.emit(target.Token, code.OpIndex)
		c.emit(node.Token, code.OpIncrement, decrement)
		c.emit(node.Token, code.OpSetIndex)

	case *ast.DotExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		name := c.addConstant(&object.String{Value: target.Property.Value})
		c.emit(token.Token{}, code.OpDup)
		c.emit(target.Token, code.OpGetProperty, name, c.cache())
		c.emit(node.Token, code.OpIncrement, decrement)
		c.emit(node.Token, code.OpSetProperty, name, c.cache())

	default:
		c.emitError(node.Token, "invalid assignment target: %s", node.Left.String())
	}
	return nil
}

// emitError compiles a runtime error, raised when the code runs.
func (c *Compiler) emitError(tok token.Token, format string, a ...interface{}) {
	message := strfmt.Sprintf(format, a...)
	c.problems = append(c.problems, Problem{Line: tok.Line, Column: tok.Column, Message: message})
	c.emit(tok, code.OpError, c.addConstant(&object.String{Value: message}))
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

// emit appends an instruction and returns its offset. When tok has a
// position, errors raised by the instruction are reported there.
func (c *Compiler) emit(tok token.Token, op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := len(c.scope.instructions)
	if tok.Line > 0 {
		c.scope.positions = append(c.scope.positions, object.Position{Offset: pos, Line: tok.Line, Column: tok.Column})
	} else if n := len(c.scope.positions); n > 0 && c.scope.positions[n-1].Line != 0 {
		// Instructions without a position don't take the previous one
		c.scope.positions = append(c.scope.positions, object.Position{Offset: pos})
	}
	c.scope.instructions = append(c.scope.instructions, ins...)
	return pos
}

// changeOperands rewrites the operands of the instruction at pos, to point a
// jump at a target known only later.
func (c *Compiler) changeOperands(pos int, operands ...int) {
	op := code.Opcode(c.scope.instructions[pos])
	copy(c.scope.instructions[pos:], code.Make(op, operands...))
}

func hasSpread(expressions []ast.Expression) bool {
	for _, e := range expressions {
		if _, ok := e.(*ast.SpreadExpression); ok {
			return true
		}
	}
	return false
}

func hashHasSpread(hash *ast.HashLiteral) bool {
	for _, name := range hash.Order {
		if hash.Pairs[name].Value == nil {
			return true
		}
	}
	return false
}

// compileSpreadList builds an array element by element, adding all the
// elements of each spread.
func (c *Compiler) compileSpreadList(tok token.Token, elements []ast.Expression) error {
	c.emit(token.Token{}, code.OpArray, 0)
	for _, el := range elements {
		if spread, ok := el.(*ast.SpreadExpression); ok {
			if err := c.compileOperand(spread.Value); err != nil {
				return err
			}
			c.emit(spread.Token, code.OpAppend, 1)
			continue
		}
		if err := c.compileOperand(el); err != nil {
			return err
		}
		c.emit(tok, code.OpAppend, 0)
	}
	return nil
}

// compileSpreadArgs builds the arguments of a call with a spread as an
// array.
func (c *Compiler) compileSpreadArgs(tok token.Token, args []ast.Expression) error {
	return c.compileSpreadList(tok, args)
}

// compileSpreadHash builds an object pair by pair, copying the properties of
// each spread.
func (c *Compiler) compileSpreadHash(node *ast.HashLiteral) error {
	c.emit(token.Token{}, code.OpHash, 0, c.cache())
	for _, name := range node.Order {
		pair := node.Pairs[name]
		if pair.Value == nil {
			spread := pair.Key.(*ast.SpreadExpression)
			if err := c.compileOperand(spread.Value); err != nil {
				return err
			}
			c.emit(spread.Token, code.OpHashMerge)
			continue
		}
		if ident, ok := pair.Key.(*ast.Identifier); ok {
			c.emit(token.Token{}, code.OpConstant, c.addConstant(&object.String{Value: ident.Value}))
		} else if err := c.compileOperand(pair.Key); err != nil {
			return err
		}
		if err := c.compileOperand(pair.Value); err != nil {
			return err
		}
		c.emit(node.Token, code.OpHashSet)
	}
	return nil
}

func restName(pattern *ast.Pattern) []*ast.Identifier {
	if pattern.Rest == nil {
		return nil
	}
	return []*ast.Identifier{pattern.Rest}
}

// compileMatch compiles a match expression. The subject stays on the stack
// while the arms test it, and the chosen arm replaces it with its value.
func (c *Compiler) compileMatch(node *ast.MatchExpression) error {
	if err := c.compileOperand(node.Subject); err != nil {
		return err
	}

	ends := []int{}
	for _, arm := range node.Arms {
		toBody := []int{}
		nextArm := -1
		for _, pattern := range arm.Patterns {
			c.emit(token.Token{}, code.OpDup)
			if r, ok := pattern.(*ast.InfixExpression); ok && r.Operator == ".." {
				// A range matches the values between its ends
				if err := c.compileOperand(r.Left); err != nil {
					return err
				}
				if err := c.compileOperand(r.Right); err != nil {
					return err
				}
				c.emit(r.Token, code.OpInRange)
			} else {
				if err := c.compileOperand(pattern); err != nil {
					return err
				}
				c.emit(node.Token, code.OpEqual)
			}
			miss := c.emit(token.Token{}, code.OpJumpNotTruthy, 9999)
			toBody = append(toBody, c.emit(token.Token{}, code.OpJump, 9999))
			c.changeOperands(miss, len(c.scope.instructions))
		}
		if arm.Patterns != nil {
			nextArm = c.emit(token.Token{}, code.OpJump, 9999)
		}

		for _, jump := range toBody {
			c.changeOperands(jump, len(c.scope.instructions))
		}
		guardMiss := -1
		if arm.Guard != nil {
			if err := c.compileOperand(arm.Guard); err != nil {
				return err
			}
			guardMiss = c.emit(token.Token{}, code.OpJumpNotTruthy, 9999)
		}
		c.emit(token.Token{}, code.OpPop) // the subject
		var err error
		if block, ok := arm.Body.(*ast.BlockStatement); ok {
			err = c.compileBlock(block)
		} else {
			err = c.compileOperand(arm.Body.(ast.Expression))
		}
		if err != nil {
			return err
		}
		ends = append(ends, c.emit(token.Token{}, code.OpJump, 9999))

		if nextArm >= 0 {
			c.changeOperands(nextArm, len(c.scope.instructions))
		}
		if guardMiss >= 0 {
			c.changeOperands(guardMiss, len(c.scope.instructions))
		}
	}

	// No arm matched
	c.emit(token.Token{}, code.OpPop)
	c.emit(token.Token{}, code.OpNull)
	for _, jump := range ends {
		c.changeOperands(jump, len(c.scope.instructions))
	}
	return nil
}

// compileCompoundAssignment compiles x op= value, which reads the target
// once. x ??= value only assigns when x is null.
func (c *Compiler) compileCompoundAssignment(node *ast.AssignmentExpression) error {
	// below is how many values the target leaves under its current value:
	// none for a variable, the object for obj.name, and two for c[i]
	below := 0
	var set func()
	switch target := node.Left.(type) {
	case *ast.Identifier:
		c.compileGet(target.Token, target.Value)
		set = func() { c.compileSet(node.Token, target.Value) }
	case *ast.IndexExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		if err := c.compileExpression(target.Index); err != nil {
			return err
		}
		c.emit(token.Token{}, code.OpDup2)
		c.emit(target.Token, code.OpIndex)
		below = 2
		set = func() { c.emit(node.Token, code.OpSetIndex) }
	case *ast.DotExpression:
		if err := c.compileExpression(target.Left); err != nil {
			return err
		}
		name := c.addConstant(&object.String{Value: target.Property.Value})
		c.emit(token.Token{}, code.OpDup)
		c.emit(target.Token, code.OpGetProperty, name, c.cache())
		below = 1
		set = func() { c.emit(node.Token, code.OpSetProperty, name, c.cache()) }
	default:
		c.emitError(node.Token, "invalid assignment target: %s", node.Left.String())
		return nil
	}

	if node.Operator == "??" {
		keep := c.emit(token.Token{}, code.OpJumpNotNullKeep, 9999)
		if err := c.compileOperand(node.Value); err != nil {
			return err
		}
		set()
		end := c.emit(token.Token{}, code.OpJump, 9999)
		// Not null: keep the current value, without what was below it
		c.changeOperands(keep, len(c.scope.instructions))
		if below > 0 {
			c.emit(token.Token{}, code.OpSlide, below)
		}
		c.changeOperands(end, len(c.scope.instructions))
		return nil
	}

	op, ok := infixOpcodes[node.Operator]
	if !ok {
		return strfmt.Errorf("unknown operator %s=", node.Operator)
	}
	if err := c.compileOperand(node.Value); err != nil {
		return err
	}
	c.emit(node.Token, op)
	set()
	return nil
}
