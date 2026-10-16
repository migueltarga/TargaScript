// Package vm runs bytecode produced by the compiler.
package vm

import (
	"io"

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

// frame is a function call in progress. Its locals live on the stack from
// base, and loops remembers the stack height of its open loops.
type frame struct {
	cl       *object.Closure
	ip       int
	base     int
	loops    []loop
	handlers []handler
}

type loop struct {
	sp         int
	breakTo    int
	continueTo int
	handlers   int // try blocks open when the loop started
}

// handler is an open try block: where its catch block starts, and the stack
// height and loops to go back to when an error is caught.
type handler struct {
	sp    int
	catch int
	loops int
}

// VM runs compiled programs. A VM keeps its globals between runs, for a
// REPL, as long as the programs come from the same compiler.
type VM struct {
	constants []object.Object
	globals   []object.Object
	names     *compiler.GlobalTable
	builtins  []*object.Builtin

	stack []object.Object
	sp    int // the next free slot; the top is stack[sp-1]

	frames []*frame

	modules *modules

	// pendingError stops a resumed run right away, when the promise it
	// awaited was rejected
	pendingError *object.Error

	// Output is where print writes while this VM runs, including when a
	// paused run resumes later. When nil, print uses evaluator.SetOutput.
	Output io.Writer
}

func New(globals *compiler.GlobalTable) *VM {
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

// Run runs a compiled program and returns the value of its last statement,
// or the error that stopped it.
func (vm *VM) Run(bytecode *compiler.Bytecode) object.Object {
	vm.constants = bytecode.Constants
	for len(vm.globals) < len(vm.names.Names()) {
		vm.globals = append(vm.globals, nil)
	}

	main := &object.Closure{Fn: bytecode.Main, Runner: vm}
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

var (
	// Closures running for code outside a VM, each nested in Go's stack
	nestedRuns int

	// MaxNestedRuns limits nestedRuns where the native stack is small, as
	// in WebAssembly. Zero means no limit.
	MaxNestedRuns int
)

// CallClosure runs a closure for code outside the VM, such as a builtin that
// takes a function, and returns its result.
func (vm *VM) CallClosure(cl *object.Closure, args []object.Object) object.Object {
	if MaxNestedRuns > 0 && nestedRuns >= MaxNestedRuns {
		return evaluator.NewError(0, 0, "maximum call depth exceeded")
	}
	nestedRuns++
	defer func() { nestedRuns-- }()

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
			fb = &fiber{promise: object.NewPromise()}
		}
		vm.suspend(base, fb, s)
		return fb.promise
	}
	if err, ok := result.(*object.Error); ok {
		f := vm.frames[len(vm.frames)-1]
		if err.Line == 0 {
			err.Line, err.Column = f.cl.Fn.PositionAt(f.ip)
		}
		bottom := vm.frames[base]
		vm.sp = bottom.base - 1
		vm.frames = vm.frames[:base]
		return err
	}
	return result
}

func (vm *VM) loop(base int) object.Object {
	if err := vm.pendingError; err != nil {
		vm.pendingError = nil
		return err
	}

	f := vm.frames[len(vm.frames)-1]
	ins := f.cl.Fn.Instructions

	for {
		ip := f.ip
		op := code.Opcode(ins[ip])
		f.ip++

		switch op {
		case code.OpConstant:
			vm.push(vm.constants[code.ReadUint16(ins[ip+1:])])
			f.ip += 2

		case code.OpTrue:
			vm.push(TRUE)
		case code.OpFalse:
			vm.push(FALSE)
		case code.OpNull:
			vm.push(NULL)

		case code.OpArray:
			n := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			elements := make([]object.Object, n)
			copy(elements, vm.stack[vm.sp-n:vm.sp])
			vm.sp -= n
			vm.push(&object.Array{Elements: elements})

		case code.OpHash:
			n := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			hash := object.NewHash()
			start := vm.sp - 2*n
			for i := start; i < vm.sp; i += 2 {
				key, ok := vm.stack[i].(object.Hashable)
				if !ok {
					return vm.errorAt(f, ip, "unusable as hash key: %s", vm.stack[i].Type())
				}
				hash.Set(key, vm.stack[i+1])
			}
			vm.sp = start
			vm.push(hash)

		case code.OpTemplate:
			n := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			text := ""
			for i := vm.sp - n; i < vm.sp; i++ {
				text += evaluator.ToDisplayString(vm.stack[i])
			}
			vm.sp -= n
			vm.push(&object.String{Value: text})

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpXor,
			code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterEqual,
			code.OpLessThan, code.OpLessEqual, code.OpRange:
			right := vm.stack[vm.sp-1]
			left := vm.stack[vm.sp-2]
			vm.sp -= 2
			result := vm.binary(f, ip, op, left, right)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpMinus, code.OpBang:
			operator := "-"
			if op == code.OpBang {
				operator = "!"
			}
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.Prefix(operator, vm.pop(), line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpIncrement:
			operator := "++"
			if ins[ip+1] == 1 {
				operator = "--"
			}
			f.ip++
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.Increment(vm.pop(), operator, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpPop:
			vm.sp--
		case code.OpDup:
			vm.push(vm.stack[vm.sp-1])
		case code.OpDup2:
			vm.push(vm.stack[vm.sp-2])
			vm.push(vm.stack[vm.sp-2])

		case code.OpJump:
			f.ip = int(code.ReadUint16(ins[ip+1:]))

		case code.OpJumpNotTruthy:
			target := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			if !evaluator.IsTruthy(vm.pop()) {
				f.ip = target
			}

		case code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep:
			target := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			truthy := evaluator.IsTruthy(vm.stack[vm.sp-1])
			if truthy == (op == code.OpJumpTruthyKeep) {
				f.ip = target
			} else {
				vm.sp--
			}

		case code.OpJumpNotNullKeep:
			target := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			if vm.stack[vm.sp-1] != NULL {
				f.ip = target
			} else {
				vm.sp--
			}

		case code.OpJumpNullKeep:
			target := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			if vm.stack[vm.sp-1] == NULL {
				f.ip = target
			}

		case code.OpLoopEnter:
			f.loops = append(f.loops, loop{
				sp:         vm.sp,
				breakTo:    int(code.ReadUint16(ins[ip+1:])),
				continueTo: int(code.ReadUint16(ins[ip+3:])),
				handlers:   len(f.handlers),
			})
			f.ip += 4

		case code.OpLoopExit:
			f.loops = f.loops[:len(f.loops)-1]

		case code.OpBreak:
			l := f.loops[len(f.loops)-1]
			vm.sp = l.sp
			f.handlers = f.handlers[:l.handlers]
			if ins[ip+1] == 1 {
				f.ip = l.continueTo
			} else {
				f.ip = l.breakTo
			}

		case code.OpIterStart:
			it, err := newIterator(vm.pop())
			if err != nil {
				return vm.errorAt(f, ip, "%s", err.Error())
			}
			vm.push(it)

		case code.OpRangeIter:
			end := vm.pop()
			start := vm.pop()
			s, okStart := start.(*object.Integer)
			e, okEnd := end.(*object.Integer)
			if !okStart || !okEnd {
				return vm.errorAt(f, ip, "range bounds must be integers, got %s..%s", start.Type(), end.Type())
			}
			vm.push(&rangeIterator{next: s.Value, end: e.Value})

		case code.OpIterNext:
			target := int(code.ReadUint16(ins[ip+1:]))
			pairs := ins[ip+3] == 1
			f.ip += 3
			it := vm.stack[vm.sp-1].(iterator)
			key, value, ok := it.Next()
			switch {
			case !ok:
				f.ip = target
			case pairs:
				vm.push(key)
				vm.push(value)
			case it.KeysFirst():
				vm.push(key)
			default:
				vm.push(value)
			}

		case code.OpGetGlobal:
			i := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			value := vm.globals[i]
			if value == nil {
				name := vm.names.Names()[i]
				// A builtin is still usable before a global with its name is set
				if builtin, ok := evaluator.Builtin(name); ok {
					value = builtin
				} else {
					return vm.notDefined(f, ip, name)
				}
			}
			vm.push(value)

		case code.OpSetGlobal:
			vm.globals[code.ReadUint16(ins[ip+1:])] = vm.stack[vm.sp-1]
			f.ip += 2

		case code.OpGetLocal:
			i := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			value := vm.stack[f.base+i]
			if cell, ok := value.(*object.Cell); ok {
				value = cell.Value
			}
			if value == nil {
				return vm.notDefined(f, ip, f.cl.Fn.LocalNames[i])
			}
			vm.push(value)

		case code.OpSetLocal:
			i := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			if cell, ok := vm.stack[f.base+i].(*object.Cell); ok {
				cell.Value = vm.stack[vm.sp-1]
			} else {
				vm.stack[f.base+i] = vm.stack[vm.sp-1]
			}

		case code.OpNewCell:
			i := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			vm.stack[f.base+i] = &object.Cell{Value: vm.stack[vm.sp-1]}

		case code.OpGetLocalCell:
			i := int(code.ReadUint16(ins[ip+1:]))
			f.ip += 2
			cell, ok := vm.stack[f.base+i].(*object.Cell)
			if !ok {
				// Captured before its declaration ran, as by a function
				// defined before the variable
				cell = &object.Cell{}
				vm.stack[f.base+i] = cell
			}
			vm.push(cell)

		case code.OpGetFree:
			i := int(ins[ip+1])
			f.ip++
			value := f.cl.Free[i].Value
			if value == nil {
				return vm.notDefined(f, ip, f.cl.Fn.FreeNames[i])
			}
			vm.push(value)

		case code.OpSetFree:
			f.cl.Free[ins[ip+1]].Value = vm.stack[vm.sp-1]
			f.ip++

		case code.OpGetFreeCell:
			vm.push(f.cl.Free[ins[ip+1]])
			f.ip++

		case code.OpGetBuiltin:
			vm.push(vm.builtins[code.ReadUint16(ins[ip+1:])])
			f.ip += 2

		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.Index(left, index, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpSetIndex:
			value := vm.pop()
			index := vm.pop()
			container := vm.pop()
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.SetIndex(container, index, value, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpGetProperty:
			name := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			f.ip += 2
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.GetProperty(vm.pop(), name, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpSetProperty:
			name := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			f.ip += 2
			value := vm.pop()
			obj := vm.pop()
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.SetProperty(obj, name, value, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpCall:
			argc := int(ins[ip+1])
			f.ip++
			switched, err := vm.call(f, ip, argc)
			if err != nil {
				return err
			}
			if switched {
				f = vm.frames[len(vm.frames)-1]
				ins = f.cl.Fn.Instructions
			}

		case code.OpCallArray:
			args := vm.pop().(*object.Array)
			for _, arg := range args.Elements {
				vm.push(arg)
			}
			switched, err := vm.call(f, ip, len(args.Elements))
			if err != nil {
				return err
			}
			if switched {
				f = vm.frames[len(vm.frames)-1]
				ins = f.cl.Fn.Instructions
			}

		case code.OpCallMethodArray:
			name := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			f.ip += 2
			args := vm.pop().(*object.Array)
			obj := vm.pop()
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.CallMethod(obj, name, append([]object.Object{}, args.Elements...), line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpAppend:
			value := vm.pop()
			array := vm.stack[vm.sp-1].(*object.Array)
			f.ip++
			if ins[ip+1] == 0 {
				array.Elements = append(array.Elements, value)
				continue
			}
			switch value := value.(type) {
			case *object.Array:
				array.Elements = append(array.Elements, value.Elements...)
			case *object.String:
				for _, char := range value.Value {
					array.Elements = append(array.Elements, &object.String{Value: string(char)})
				}
			default:
				return vm.errorAt(f, ip, "cannot spread %s into a list, only arrays and strings", evaluator.TypeName(value))
			}

		case code.OpHashSet:
			value := vm.pop()
			key := vm.pop()
			hash := vm.stack[vm.sp-1].(*object.Hash)
			hashable, ok := key.(object.Hashable)
			if !ok {
				return vm.errorAt(f, ip, "unusable as hash key: %s", key.Type())
			}
			hash.Set(hashable, value)

		case code.OpHashMerge:
			source := vm.pop()
			hash := vm.stack[vm.sp-1].(*object.Hash)
			from, ok := source.(*object.Hash)
			if !ok {
				return vm.errorAt(f, ip, "cannot spread %s into an object, only objects", evaluator.TypeName(source))
			}
			for _, pair := range from.OrderedPairs() {
				hash.Set(pair.Key.(object.Hashable), pair.Value)
			}

		case code.OpCallMethod:
			name := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			argc := int(ins[ip+3])
			f.ip += 3
			args := make([]object.Object, argc)
			copy(args, vm.stack[vm.sp-argc:vm.sp])
			vm.sp -= argc
			obj := vm.pop()
			line, column := f.cl.Fn.PositionAt(ip)
			result := evaluator.CallMethod(obj, name, args, line, column)
			if err, ok := result.(*object.Error); ok {
				return err
			}
			vm.push(result)

		case code.OpReturnValue:
			value := vm.pop()
			vm.frames = vm.frames[:len(vm.frames)-1]
			vm.sp = f.base - 1
			vm.push(value)
			if len(vm.frames) == base {
				vm.sp--
				return value
			}
			f = vm.frames[len(vm.frames)-1]
			ins = f.cl.Fn.Instructions

		case code.OpClosure:
			fn := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.CompiledFunction)
			n := int(ins[ip+3])
			f.ip += 3
			free := make([]*object.Cell, n)
			for i := 0; i < n; i++ {
				free[i] = vm.stack[vm.sp-n+i].(*object.Cell)
			}
			vm.sp -= n
			vm.push(&object.Closure{Fn: fn, Free: free, Runner: vm})

		case code.OpPrint:
			n := int(ins[ip+1])
			f.ip++
			values := make([]object.Object, n)
			copy(values, vm.stack[vm.sp-n:vm.sp])
			vm.sp -= n
			evaluator.Print(values)
			vm.push(NULL)

		case code.OpLoad:
			path := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			shown := vm.constants[code.ReadUint16(ins[ip+3:])].(*object.String).Value
			f.ip += 4
			line, column := f.cl.Fn.PositionAt(ip)
			module := vm.load(path, shown, line, column)
			if err, ok := module.(*object.Error); ok {
				return err
			}
			vm.push(module)

		case code.OpAwait:
			value := vm.pop()
			awaitable, ok := value.(object.Awaitable)
			if !ok || !awaitable.IsAwaitable() {
				vm.push(value)
				continue
			}
			line, column := f.cl.Fn.PositionAt(ip)
			if base != 0 {
				// Go code called this function, and its stack can't be saved
				return evaluator.NewError(line, column,
					"await can't pause here: this function was called by a builtin such as map or sort")
			}
			return &suspension{awaited: awaitable, line: line, column: column}

		case code.OpTry:
			f.handlers = append(f.handlers, handler{
				sp:    vm.sp,
				catch: int(code.ReadUint16(ins[ip+1:])),
				loops: len(f.loops),
			})
			f.ip += 2

		case code.OpTryEnd:
			f.handlers = f.handlers[:len(f.handlers)-1]

		case code.OpThrow:
			return vm.thrown(f, ip, vm.pop())

		case code.OpError:
			message := vm.constants[code.ReadUint16(ins[ip+1:])].(*object.String).Value
			f.ip += 2
			return vm.errorAt(f, ip, "%s", message)

		default:
			return vm.errorAt(f, ip, "unknown opcode %d", op)
		}
	}
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
	vm.ensureStack(base + fn.NumLocals + 256)
	for i := base + fn.NumParams; i < base+fn.NumLocals; i++ {
		vm.stack[i] = nil
	}
	vm.sp = base + fn.NumLocals

	vm.frames = append(vm.frames, &frame{cl: cl, base: base})
	return nil
}

func (vm *VM) binary(f *frame, ip int, op code.Opcode, left, right object.Object) object.Object {
	// Fast paths for small integers, which can't overflow
	if l, ok := left.(*object.Integer); ok {
		if r, ok := right.(*object.Integer); ok && smallInt(l.Value) && smallInt(r.Value) {
			switch op {
			case code.OpAdd:
				return &object.Integer{Value: l.Value + r.Value}
			case code.OpSub:
				return &object.Integer{Value: l.Value - r.Value}
			case code.OpLessThan:
				return nativeBool(l.Value < r.Value)
			case code.OpLessEqual:
				return nativeBool(l.Value <= r.Value)
			case code.OpGreaterThan:
				return nativeBool(l.Value > r.Value)
			case code.OpGreaterEqual:
				return nativeBool(l.Value >= r.Value)
			case code.OpEqual:
				return nativeBool(l.Value == r.Value)
			case code.OpNotEqual:
				return nativeBool(l.Value != r.Value)
			}
		}
	}

	line, column := f.cl.Fn.PositionAt(ip)
	return evaluator.Infix(operators[op], left, right, line, column)
}

var operators = map[code.Opcode]string{
	code.OpAdd: "+", code.OpSub: "-", code.OpMul: "*", code.OpDiv: "/",
	code.OpMod: "%", code.OpXor: "^", code.OpEqual: "==", code.OpNotEqual: "!=",
	code.OpGreaterThan: ">", code.OpGreaterEqual: ">=", code.OpLessThan: "<",
	code.OpLessEqual: "<=", code.OpRange: "..",
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

func (vm *VM) pop() object.Object {
	vm.sp--
	return vm.stack[vm.sp]
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
