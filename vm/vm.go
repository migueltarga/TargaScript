// Package vm runs bytecode produced by the compiler.
package vm

import (
	"io"
	"sync"

	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

const initialStackSize = 1024

var (
	NULL  = evaluator.NULL
	TRUE  = evaluator.TRUE
	FALSE = evaluator.FALSE
)

// frame is a function call in progress. Its registers live on the stack
// from base: its locals, then the values it works on.
type frame struct {
	cl       *object.Closure
	ip       int
	base     int
	handlers []handler
	// driver is set for a call of the function of map, filter or reduce
	// that the VM runs itself
	driver *driver
}

// handler is an open try block: where its catch block starts, and the
// register the error goes in.
type handler struct {
	catch int
	reg   int
}

// VM runs compiled programs. A VM keeps its globals between runs, for a
// REPL, as long as the programs come from the same compiler.
type VM struct {
	constants []object.Object
	globals   []object.Object
	names     *compiler.GlobalTable
	builtins  []*object.Builtin
	host      evaluator.Host

	stack []object.Object
	sp    int // the next free slot; the top is stack[sp-1]

	ints ints

	frames []*frame

	modules *modules

	// pendingError stops a resumed run right away, when the promise it
	// awaited was rejected
	pendingError *object.Error

	// receiver is the object of the method call about to start, for its
	// function's this
	receiver object.Object

	// asyncBases are the bases of the runs that can pause at an await
	// besides the outermost one: closures called with CallClosureAsync
	// and resumed fibers, which nothing waits for
	asyncBases []int

	// Output is where print writes while this VM runs, including when a
	// paused run resumes later. When nil, print uses evaluator.SetOutput.
	Output io.Writer

	// MaxSteps, when above zero, stops a run with an error after that many
	// loop iterations and calls, to bound programs that might not end
	MaxSteps   int
	steps      int
	nestedRuns int
}

func New(globals *compiler.GlobalTable) *VM {
	smallIntsOnce.Do(makeSmallInts)
	vm := &VM{
		names:   globals,
		stack:   make([]object.Object, initialStackSize),
		modules: newModules(),
	}
	for _, name := range evaluator.BuiltinNames() {
		builtin, _ := evaluator.Builtin(name)
		vm.builtins = append(vm.builtins, builtin)
	}
	return vm
}

// SetHost gives this VM its own asynchronous builtins. Loaded modules inherit it.
func (vm *VM) SetHost(h evaluator.Host) {
	vm.host = h
	for i, name := range evaluator.BuiltinNames() {
		vm.builtins[i], _ = evaluator.BuiltinWithHost(name, h)
	}
}

// Host returns the host used by this VM's asynchronous builtins.
func (vm *VM) Host() evaluator.Host { return vm.host }

// RestrictBuiltins replaces host capabilities that an embedding application
// did not grant, even if another package configured the evaluator globally.
func (vm *VM) RestrictBuiltins(names ...string) {
	indexes := map[string]int{}
	for i, name := range evaluator.BuiltinNames() {
		indexes[name] = i
	}
	for _, name := range names {
		if i, ok := indexes[name]; ok {
			name := name
			vm.builtins[i] = &object.Builtin{Name: name, Fn: func(...object.Object) object.Object {
				return evaluator.NewError(0, 0, "%s is not available in embedded scripts", name)
			}}
		}
	}
}

// Run runs a compiled program and returns the value of its last statement,
// or the error that stopped it.
func (vm *VM) Run(bytecode *compiler.Bytecode) object.Object {
	vm.constants = bytecode.Constants
	for len(vm.globals) < len(vm.names.Names()) {
		vm.globals = append(vm.globals, nil)
	}

	main := &object.Closure{Fn: bytecode.Main, Runner: vm}
	vm.steps = 0
	vm.sp = 0
	vm.frames = vm.frames[:0]
	vm.push(main)
	if err := vm.callClosure(main, 0); err != nil {
		return err
	}
	return vm.run(0)
}

// SetGlobal defines a global with a value, for values the host provides,
// like the js object in the browser.
func (vm *VM) SetGlobal(name string, value object.Object) {
	i := vm.names.Define(name)
	for len(vm.globals) <= i {
		vm.globals = append(vm.globals, nil)
	}
	vm.globals[i] = value
}

// MaxNestedRuns limits nestedRuns where the native stack is small, as
// in WebAssembly. Zero means no limit.
var MaxNestedRuns int

// CallClosure runs a closure for code outside the VM, such as a builtin that
// takes a function, and returns its result.
func (vm *VM) CallClosure(cl *object.Closure, args []object.Object) object.Object {
	if MaxNestedRuns > 0 && vm.nestedRuns >= MaxNestedRuns {
		return evaluator.NewError(0, 0, "maximum call depth exceeded")
	}
	if vm.MaxSteps > 0 {
		vm.steps++
		if vm.steps > vm.MaxSteps {
			return evaluator.NewError(0, 0, "the program took more than %d steps", vm.MaxSteps)
		}
	}
	vm.nestedRuns++
	defer func() { vm.nestedRuns-- }()

	base := len(vm.frames)
	vm.push(cl)
	for _, arg := range args {
		vm.push(arg)
	}
	if err := vm.callClosure(cl, len(args)); err != nil {
		vm.sp -= len(args) + 1
		return err
	}
	return vm.run(base)
}

// CallClosureAsync is CallClosure for a closure that can pause at an await,
// like the function of map: then it returns the promise of its result.
func (vm *VM) CallClosureAsync(cl *object.Closure, args []object.Object) object.Object {
	vm.asyncBases = append(vm.asyncBases, len(vm.frames))
	defer func() { vm.asyncBases = vm.asyncBases[:len(vm.asyncBases)-1] }()
	return vm.CallClosure(cl, args)
}

// trace adds the calls of the run from base to the trace of err. The
// innermost frame is where the error happened, unless a run inside it,
// like the function given to map, already traced it; the others are at the
// call they're waiting for.
func (vm *VM) trace(base int, err *object.Error) {
	for i := len(vm.frames) - 1; i >= base; i-- {
		f := vm.frames[i]
		line := err.Line
		if i < len(vm.frames)-1 || len(err.Trace) > 0 {
			line, _ = f.cl.Fn.PositionAt(f.ip - 1)
		}
		err.Trace = append(err.Trace, object.Call{Function: f.cl.Fn.Name, Line: line})
	}
}

// step counts a loop iteration or a call against MaxSteps.
func (vm *VM) step(f *frame, ip int) *object.Error {
	vm.steps++
	if vm.steps > vm.MaxSteps {
		return vm.positioned(f, ip, evaluator.NewError(0, 0, "the program took more than %d steps", vm.MaxSteps))
	}
	return nil
}

// ownMethod returns the function an object holds under name, when it runs
// on this VM. An object's own functions come before the methods every
// object has, like keys.
func (vm *VM) ownMethod(obj object.Object, name *object.String, cache *object.PropertyCache) *object.Closure {
	hash, ok := obj.(*object.Hash)
	if !ok {
		return nil
	}
	value, ok := hash.Lookup(name.HashKey(), cache)
	if !ok {
		return nil
	}
	if cl, ok := value.(*object.Closure); ok && cl.Runner == vm {
		return cl
	}
	return nil
}

// waitForPausedCall waits, like waitFor, when the call that just ran on
// another VM, like a loaded file's function, paused and left its promise on
// the stack.
func (vm *VM) waitForPausedCall(f *frame, ip, base int) object.Object {
	promise, ok := vm.stack[vm.sp-1].(*object.Promise)
	if !ok || !promise.Paused {
		return nil
	}
	vm.sp--
	return vm.waitFor(f, ip, base, promise)
}

// waitFor pauses the run at a call that gave the promise of a function
// that paused, as an await would, and returns the suspension, or pushes the
// promise's result when it's already settled. It returns an error when the
// run can't pause.
func (vm *VM) waitFor(f *frame, ip, base int, promise *object.Promise) object.Object {
	if promise.Settled() {
		value, rejected := promise.Result()
		if rejected {
			return vm.positioned(f, ip, evaluator.NewError(0, 0, "%s", rejectionMessage(value)))
		}
		vm.push(value)
		return nil
	}
	line, column := f.cl.Fn.PositionAt(ip)
	if !vm.canPause(base) {
		return evaluator.NewError(line, column,
			"await can't pause here: this function was called by a builtin such as filter or sort")
	}
	return &suspension{awaited: promise, line: line, column: column}
}

// canPause reports whether the run from base can pause at an await.
func (vm *VM) canPause(base int) bool {
	if base == 0 {
		return true
	}
	for i := len(vm.asyncBases) - 1; i >= 0; i-- {
		if vm.asyncBases[i] == base {
			return true
		}
	}
	return false
}

// run executes instructions until the frame at index base returns, and
// returns its value. On an error it unwinds back to base.
func (vm *VM) run(base int) object.Object {
	return vm.runFiber(base, nil)
}

// runFiber is run for a fiber being resumed, or for a new run when fb is nil.
// When the run pauses at an await it returns the promise of its fiber.
func (vm *VM) runFiber(base int, fb *fiber) object.Object {
	if vm.Output != nil {
		previous := evaluator.Output()
		evaluator.SetOutput(vm.Output)
		defer evaluator.SetOutput(previous)
	}

	result := vm.loop(base)
	for {
		err, ok := result.(*object.Error)
		if !ok || !vm.catch(base, err) {
			break
		}
		result = vm.loop(base)
	}

	if s, ok := result.(*suspension); ok {
		if fb == nil {
			fb = &fiber{promise: &object.Promise{Paused: true}}
		}
		vm.suspend(base, fb, s)
		return fb.promise
	}
	if err, ok := result.(*object.Error); ok {
		f := vm.frames[len(vm.frames)-1]
		if err.Line == 0 {
			err.Line, err.Column = f.cl.Fn.PositionAt(f.ip)
		}
		vm.trace(base, err)
		bottom := vm.frames[base]
		vm.sp = bottom.base - 1
		vm.frames = vm.frames[:base]
		return err
	}
	return result
}

// loop runs the code of the frames from base.
func (vm *VM) loop(base int) object.Object {
	return vm.loopWords(base)
}

// call calls the function below the argc arguments on the stack. For a
// closure of this VM it pushes a frame and reports switched, so the loop
// continues in it; other functions run right away and leave their result.
func (vm *VM) call(f *frame, ip int, argc int) (switched bool, err *object.Error) {
	callee := vm.stack[vm.sp-1-argc]
	if cl, ok := callee.(*object.Closure); ok && cl.Runner == vm {
		if err := vm.callClosure(cl, argc); err != nil {
			err.Line, err.Column = f.cl.Fn.PositionAt(ip)
			return false, err
		}
		return true, nil
	}

	args := make([]object.Object, argc)
	copy(args, vm.stack[vm.sp-argc:vm.sp])
	vm.sp -= argc + 1
	result := evaluator.CallFunction(callee, args)
	if err, ok := result.(*object.Error); ok {
		if err.Line == 0 {
			err.Line, err.Column = f.cl.Fn.PositionAt(ip)
		}
		return false, err
	}
	vm.push(result)
	return false, nil
}

// newFrame returns a frame for a call, reusing one from a finished call
// when possible.
func (vm *VM) pushFrame(cl *object.Closure, base int) *frame {
	// The frames of finished calls stay past the end of vm.frames, to
	// reuse, unless a paused run took them
	n := len(vm.frames)
	var f *frame
	if n < cap(vm.frames) {
		vm.frames = vm.frames[:n+1]
		f = vm.frames[n]
	} else {
		vm.frames = append(vm.frames, nil)
	}
	if f == nil {
		f = &frame{}
		vm.frames[n] = f
	}
	f.cl, f.ip, f.base, f.driver = cl, 0, base, nil
	f.handlers = f.handlers[:0]
	return f
}

// callClosure starts a call to the closure below the argc arguments on the
// stack.
func (vm *VM) callClosure(cl *object.Closure, argc int) *object.Error {
	// The main program has a frame too, which isn't a call
	if len(vm.frames) > evaluator.MaxCallDepth {
		return evaluator.NewError(0, 0, "maximum call depth exceeded (%d)", evaluator.MaxCallDepth)
	}

	fn := cl.Fn
	if fn.Variadic {
		// The extra arguments go in an array, as the last parameter
		fixed := fn.NumParams - 1
		rest := []object.Object{}
		if argc > fixed {
			rest = append(rest, vm.stack[vm.sp-(argc-fixed):vm.sp]...)
			vm.sp -= argc - fixed
			argc = fixed
		}
		for ; argc < fixed; argc++ {
			vm.push(NULL)
		}
		vm.push(&object.Array{Elements: rest})
		argc++
	}
	// Missing arguments are null and extra ones are dropped
	for ; argc < fn.NumParams; argc++ {
		vm.push(NULL)
	}
	vm.sp -= argc - fn.NumParams

	base := vm.sp - fn.NumParams
	vm.ensureStack(base + fn.NumRegisters + 256)
	for i := base + fn.NumParams; i < base+fn.NumLocals; i++ {
		vm.stack[i] = nil
	}
	if fn.HasThis {
		this := vm.receiver
		if this == nil {
			this = NULL
		}
		vm.stack[base+fn.NumParams] = this
	}
	vm.receiver = nil
	// The frame takes all its registers
	vm.sp = base + fn.NumRegisters

	vm.pushFrame(cl, base)
	return nil
}

func (vm *VM) binary(f *frame, ip int, op code.Opcode, left, right object.Object) object.Object {
	if l, ok := left.(*object.Integer); ok {
		if r, ok := right.(*object.Integer); ok {
			if result := vm.intBinary(op, l.Value, r.Value); result != nil {
				return result
			}
		}
	}

	result := evaluator.Infix(operator(op), left, right, 0, 0)
	if err, ok := result.(*object.Error); ok {
		return vm.positioned(f, ip, err)
	}
	return result
}

// addSub does + and - on integers small enough not to overflow, the most
// common arithmetic; it is small enough for Go to inline in the loop. It
// returns false for the cases it leaves to intBinary.
func addSub(op code.Opcode, x, y int64) (int64, bool) {
	if !smallInt(x) || !smallInt(y) {
		return 0, false
	}
	switch op {
	case code.OpAdd:
		return x + y, true
	case code.OpSub:
		return x - y, true
	}
	return 0, false
}

// compare gives the result of a comparison as a bool, for the jumps that
// do one; ip is where an error is reported.
func (vm *VM) compare(f *frame, ip int, op code.Opcode, left, right object.Object) (bool, *object.Error) {
	if l, ok := left.(*object.Integer); ok {
		if r, ok := right.(*object.Integer); ok {
			return compareInts(op, l.Value, r.Value), nil
		}
	}
	result := vm.binary(f, ip, op, left, right)
	if err, ok := result.(*object.Error); ok {
		return false, err
	}
	return evaluator.IsTruthy(result), nil
}

// compareInts compares two integers with a comparison operator.
func compareInts(op code.Opcode, a, b int64) bool {
	switch op {
	case code.OpLessThan:
		return a < b
	case code.OpLessEqual:
		return a <= b
	case code.OpGreaterThan:
		return a > b
	case code.OpGreaterEqual:
		return a >= b
	case code.OpEqual:
		return a == b
	}
	return a != b
}

// intBinary is the fast path of binary for integers small enough not to
// overflow. It returns nil for the cases it leaves to the slow path.
func (vm *VM) intBinary(op code.Opcode, a, b int64) object.Object {
	if !smallInt(a) || !smallInt(b) {
		return nil
	}
	switch op {
	case code.OpAdd:
		return vm.ints.get(a + b)
	case code.OpSub:
		return vm.ints.get(a - b)
	case code.OpMul:
		if a > -(1<<30) && a < 1<<30 && b > -(1<<30) && b < 1<<30 {
			return vm.ints.get(a * b)
		}
	case code.OpDiv:
		if b != 0 {
			return vm.ints.get(a / b)
		}
	case code.OpMod:
		if b != 0 {
			return vm.ints.get(a % b)
		}
	case code.OpLessThan:
		return nativeBool(a < b)
	case code.OpLessEqual:
		return nativeBool(a <= b)
	case code.OpGreaterThan:
		return nativeBool(a > b)
	case code.OpGreaterEqual:
		return nativeBool(a >= b)
	case code.OpEqual:
		return nativeBool(a == b)
	case code.OpNotEqual:
		return nativeBool(a != b)
	}
	return nil
}

// operator is the text of a binary operator.
func operator(op code.Opcode) string {
	switch op {
	case code.OpAdd:
		return "+"
	case code.OpSub:
		return "-"
	case code.OpMul:
		return "*"
	case code.OpDiv:
		return "/"
	case code.OpMod:
		return "%"
	case code.OpPow:
		return "**"
	case code.OpXor:
		return "^"
	case code.OpEqual:
		return "=="
	case code.OpNotEqual:
		return "!="
	case code.OpGreaterThan:
		return ">"
	case code.OpGreaterEqual:
		return ">="
	case code.OpLessThan:
		return "<"
	case code.OpLessEqual:
		return "<="
	case code.OpRange:
		return ".."
	case code.OpIn:
		return "in"
	}
	return ""
}

// Integers from smallIntMin to smallIntMax are shared, so loops and
// counters don't allocate a new one for every value. Integers are never
// changed in place, so sharing them is safe.
const (
	smallIntMin = -128
	smallIntMax = 1024
)

// smallInts are made when the first VM is, in one allocation. Made while
// the program starts, they would be data in the WebAssembly build.
var (
	smallInts     [smallIntMax - smallIntMin + 1]*object.Integer
	smallIntsOnce sync.Once
)

func makeSmallInts() {
	values := make([]object.Integer, len(smallInts))
	for i := range values {
		values[i].Value = int64(i + smallIntMin)
		smallInts[i] = &values[i]
	}
}

// intBlock is how many integers ints allocates at a time.
const intBlock = 128

// ints makes integers in blocks, as one allocation of many costs far less
// than one for each. A block is freed once none of its integers is used.
// Each VM has its own, as VMs can run on different goroutines.
type ints struct {
	free []object.Integer
}

func (a *ints) get(n int64) *object.Integer {
	if n >= smallIntMin && n <= smallIntMax {
		return smallInts[n-smallIntMin]
	}
	if len(a.free) == 0 {
		a.free = make([]object.Integer, intBlock)
	}
	i := &a.free[0]
	a.free = a.free[1:]
	i.Value = n
	return i
}

// positioned gives an error raised by the instruction at ip its position,
// when it has none. Operations get no position up front, so the common case
// of no error doesn't pay for looking it up.
func (vm *VM) positioned(f *frame, ip int, err *object.Error) *object.Error {
	if err.Line == 0 {
		err.Line, err.Column = f.cl.Fn.PositionAt(ip)
	}
	return err
}

func smallInt(n int64) bool {
	return n > -(1<<61) && n < 1<<61
}

func nativeBool(b bool) object.Object {
	if b {
		return TRUE
	}
	return FALSE
}

func (vm *VM) errorAt(f *frame, ip int, format string, a ...interface{}) *object.Error {
	line, column := f.cl.Fn.PositionAt(ip)
	return evaluator.NewError(line, column, format, a...)
}

func (vm *VM) notDefined(f *frame, ip int, name string) *object.Error {
	line, column := f.cl.Fn.PositionAt(ip)
	return evaluator.NotDefined(line, column, name, vm.names.Names())
}

func (vm *VM) push(obj object.Object) {
	if vm.sp >= len(vm.stack) {
		vm.ensureStack(vm.sp + 1)
	}
	vm.stack[vm.sp] = obj
	vm.sp++
}

// ensureStack grows the stack to hold at least n values.
func (vm *VM) ensureStack(n int) {
	if n <= len(vm.stack) {
		return
	}
	size := len(vm.stack) * 2
	for size < n {
		size *= 2
	}
	stack := make([]object.Object, size)
	copy(stack, vm.stack[:vm.sp])
	vm.stack = stack
}
