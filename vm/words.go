package vm

import (
	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// operators maps the binary operations of register code to the operators
// of stack code, which binary and the fast paths for integers take.
var wordOperators = [...]code.Opcode{
	code.RAdd: code.OpAdd, code.RSub: code.OpSub, code.RMul: code.OpMul, code.RDiv: code.OpDiv,
	code.RMod: code.OpMod, code.RPow: code.OpPow, code.RXor: code.OpXor, code.REqual: code.OpEqual,
	code.RNotEqual: code.OpNotEqual, code.RGreater: code.OpGreaterThan, code.RGreaterEqual: code.OpGreaterEqual,
	code.RLess: code.OpLessThan, code.RLessEqual: code.OpLessEqual, code.RRange: code.OpRange, code.RIn: code.OpIn,
	code.RAddK: code.OpAdd, code.RSubK: code.OpSub, code.RMulK: code.OpMul, code.RDivK: code.OpDiv, code.RModK: code.OpMod,
	code.RAddI: code.OpAdd, code.RSubI: code.OpSub,
	code.RJumpLess: code.OpLessThan, code.RJumpLessEqual: code.OpLessEqual, code.RJumpGreater: code.OpGreaterThan,
	code.RJumpGreaterEqual: code.OpGreaterEqual, code.RJumpEqual: code.OpEqual, code.RJumpNotEqual: code.OpNotEqual,
	code.RJumpLessI: code.OpLessThan, code.RJumpLessEqualI: code.OpLessEqual, code.RJumpGreaterI: code.OpGreaterThan,
	code.RJumpGreaterEqualI: code.OpGreaterEqual, code.RJumpEqualI: code.OpEqual, code.RJumpNotEqualI: code.OpNotEqual,
	code.RJumpEqualK: code.OpEqual, code.RJumpNotEqualK: code.OpNotEqual,
}

// loopWords runs register code until the frame at index base returns, like
// loopStack runs stack code.
func (vm *VM) loopWords(base int) object.Object {
	if err := vm.pendingError; err != nil {
		vm.pendingError = nil
		return err
	}

	// pc and r, the frame's registers, are kept in local variables, which
	// Go keeps in machine registers. f.ip is set from pc before anything
	// that reads it, like a call or an error, and r is taken again after
	// anything that can grow the stack.
	f := vm.frames[len(vm.frames)-1]
	words := f.cl.Fn.Code
	r := vm.stack[f.base:]
	pc := f.ip

	for {
		w := words[pc]
		pc++

		switch op := w.Op(); op {
		case code.RMove:
			value := r[w.B()]
			if value == nil {
				f.ip = pc
				return vm.notDefinedRegister(f, pc-1, w.B())
			}
			r[w.A()] = value

		case code.RLoadK:
			r[w.A()] = vm.constants[w.B()]
		case code.RLoadI:
			r[w.A()] = vm.ints.get(int64(w.SB()))
		case code.RLoadNull:
			r[w.A()] = NULL
		case code.RLoadTrue:
			r[w.A()] = TRUE
		case code.RLoadFalse:
			r[w.A()] = FALSE

		case code.RAdd, code.RSub, code.RMul, code.RDiv, code.RMod, code.RPow, code.RXor,
			code.REqual, code.RNotEqual, code.RGreater, code.RGreaterEqual, code.RLess, code.RLessEqual,
			code.RRange, code.RIn:
			left, right := r[w.B()], r[w.C()]
			operator := wordOperators[op]
			if l, ok := left.(*object.Integer); ok {
				if rv, ok := right.(*object.Integer); ok {
					if n, ok := addSub(operator, l.Value, rv.Value); ok {
						r[w.A()] = vm.ints.get(n)
						continue
					}
					if result := vm.intBinary(operator, l.Value, rv.Value); result != nil {
						r[w.A()] = result
						continue
					}
				}
			}
			result, err := vm.binaryWords(f, pc-1, operator, w.B(), left, w.C(), right)
			if err != nil {
				f.ip = pc
				return err
			}
			r[w.A()] = result

		case code.RAddK, code.RSubK, code.RMulK, code.RDivK, code.RModK:
			left, right := r[w.B()], vm.constants[w.C()]
			operator := wordOperators[op]
			if l, ok := left.(*object.Integer); ok {
				if rv, ok := right.(*object.Integer); ok {
					if n, ok := addSub(operator, l.Value, rv.Value); ok {
						r[w.A()] = vm.ints.get(n)
						continue
					}
					if result := vm.intBinary(operator, l.Value, rv.Value); result != nil {
						r[w.A()] = result
						continue
					}
				}
			}
			result, err := vm.binaryWords(f, pc-1, operator, w.B(), left, -1, right)
			if err != nil {
				f.ip = pc
				return err
			}
			r[w.A()] = result

		case code.RAddI, code.RSubI:
			left := r[w.B()]
			if l, ok := left.(*object.Integer); ok {
				n := int64(w.SC())
				if op == code.RSubI {
					n = -n
				}
				if smallInt(l.Value) {
					r[w.A()] = vm.ints.get(l.Value + n)
					continue
				}
			}
			result, err := vm.binaryWords(f, pc-1, wordOperators[op], w.B(), left, -1, &object.Integer{Value: int64(w.SC())})
			if err != nil {
				f.ip = pc
				return err
			}
			r[w.A()] = result

		case code.RJumpLess, code.RJumpLessEqual, code.RJumpGreater, code.RJumpGreaterEqual,
			code.RJumpEqual, code.RJumpNotEqual:
			left, right := r[w.A()], r[w.B()]
			if l, ok := left.(*object.Integer); ok {
				if rv, ok := right.(*object.Integer); ok {
					if !compareInts(wordOperators[op], l.Value, rv.Value) {
						pc = w.C()
					}
					continue
				}
			}
			truthy, err := vm.compareWords(f, pc-1, wordOperators[op], w.A(), left, w.B(), right)
			if err != nil {
				f.ip = pc
				return err
			}
			if !truthy {
				pc = w.C()
			}

		case code.RJumpLessI, code.RJumpLessEqualI, code.RJumpGreaterI, code.RJumpGreaterEqualI,
			code.RJumpEqualI, code.RJumpNotEqualI:
			left := r[w.A()]
			if l, ok := left.(*object.Integer); ok {
				if !compareInts(wordOperators[op], l.Value, int64(w.SB())) {
					pc = w.C()
				}
				continue
			}
			truthy, err := vm.compareWords(f, pc-1, wordOperators[op], w.A(), left, -1, &object.Integer{Value: int64(w.SB())})
			if err != nil {
				f.ip = pc
				return err
			}
			if !truthy {
				pc = w.C()
			}

		case code.RJumpEqualK, code.RJumpNotEqualK:
			truthy, err := vm.compareWords(f, pc-1, wordOperators[op], w.A(), r[w.A()], -1, vm.constants[w.B()])
			if err != nil {
				f.ip = pc
				return err
			}
			if !truthy {
				pc = w.C()
			}

		case code.RJump:
			if w.A() < pc && vm.MaxSteps > 0 {
				// The jump back has the position of its loop
				if err := vm.step(f, pc-1); err != nil {
					f.ip = pc
					return err
				}
			}
			pc = w.A()

		case code.RJumpFalsy, code.RJumpTruthy:
			value := r[w.A()]
			if value == nil {
				f.ip = pc
				return vm.notDefinedRegister(f, pc-1, w.A())
			}
			if evaluator.IsTruthy(value) == (op == code.RJumpTruthy) {
				pc = w.B()
			}

		case code.RJumpNull, code.RJumpNotNull:
			if (r[w.A()] == NULL) == (op == code.RJumpNull) {
				pc = w.B()
			}

		case code.RGetGlobal:
			value := vm.globals[w.B()]
			if value == nil {
				name := vm.names.Names()[w.B()]
				// A builtin is still usable before a global with its name is set
				builtin, ok := vm.builtin(name)
				if !ok {
					f.ip = pc
					return vm.notDefined(f, pc-1, name)
				}
				value = builtin
			}
			r[w.A()] = value

		case code.RSetGlobal:
			vm.globals[w.B()] = r[w.A()]

		case code.RGetBuiltin:
			r[w.A()] = vm.builtins[w.B()]

		case code.RGetCell:
			value := r[w.B()]
			if cell, ok := value.(*object.Cell); ok {
				value = cell.Value
			}
			if value == nil {
				f.ip = pc
				return vm.notDefinedRegister(f, pc-1, w.B())
			}
			r[w.A()] = value

		case code.RSetCell:
			if cell, ok := r[w.A()].(*object.Cell); ok {
				cell.Value = r[w.B()]
			} else {
				r[w.A()] = r[w.B()]
			}

		case code.RNewCell:
			r[w.A()] = &object.Cell{Value: r[w.B()]}

		case code.RLocalCell:
			cell, ok := r[w.B()].(*object.Cell)
			if !ok {
				// Captured before its declaration ran, as by a function
				// defined before the variable
				cell = &object.Cell{}
				r[w.B()] = cell
			}
			r[w.A()] = cell

		case code.RGetFree:
			value := f.cl.Free[w.B()].Value
			if value == nil {
				f.ip = pc
				return vm.notDefined(f, pc-1, f.cl.Fn.FreeNames[w.B()])
			}
			r[w.A()] = value

		case code.RSetFree:
			f.cl.Free[w.B()].Value = r[w.A()]

		case code.RFreeCell:
			r[w.A()] = f.cl.Free[w.B()]

		case code.RCallGlobal:
			value := vm.globals[w.C()]
			if value == nil {
				name := vm.names.Names()[w.C()]
				builtin, ok := vm.builtin(name)
				if !ok {
					f.ip = pc
					return vm.notDefined(f, pc-1, name)
				}
				value = builtin
			}
			r[w.A()] = value
			fallthrough

		case code.RCall:
			argc := w.B()
			fn := f.base + w.A()
			if vm.MaxSteps > 0 {
				if err := vm.step(f, pc-1); err != nil {
					f.ip = pc
					return err
				}
			}
			// Calls of this VM's closures with the right number of
			// arguments, the usual case, start here
			if cl, ok := vm.stack[fn].(*object.Closure); ok && cl.Runner == vm &&
				argc == cl.Fn.SimpleCall && len(vm.frames) <= evaluator.MaxCallDepth {
				callee := fn + 1
				top := callee + cl.Fn.NumRegisters
				if top+256 > len(vm.stack) {
					vm.ensureStack(top + 256)
				}
				for i := callee + argc; i < callee+cl.Fn.NumLocals; i++ {
					vm.stack[i] = nil
				}
				if cl.Fn.HasThis {
					// this, after the parameters
					vm.stack[callee+argc] = NULL
				}
				vm.receiver = nil
				vm.sp = top
				f.ip = pc
				f = vm.pushFrame(cl, callee)
				words, r, pc = cl.Fn.Code, vm.stack[callee:], 0
				continue
			}
			vm.sp = fn + 1 + argc
			f.ip = pc
			switched, err := vm.call(f, pc-1, argc)
			if err != nil {
				return err
			}
			if switched {
				f = vm.frames[len(vm.frames)-1]
				words, pc = f.cl.Fn.Code, f.ip
				r = vm.stack[f.base:]
				continue
			}
			if s := vm.waitForPausedCall(f, pc-1, base); s != nil {
				return vm.awaitInto(f, s, w.A())
			}
			vm.sp = f.base + f.cl.Fn.NumRegisters
			r = vm.stack[f.base:]

		case code.RCallMethod:
			if vm.MaxSteps > 0 {
				if err := vm.step(f, pc-1); err != nil {
					f.ip = pc
					return err
				}
			}
			name := vm.constants[w.C()].(*object.String)
			argc := w.B()
			fn := f.base + w.A()
			// An object's own function runs like any call, so it can await
			receiver := vm.stack[fn]
			if cl := vm.ownMethod(receiver, name, &f.cl.Fn.Caches[pc-1]); cl != nil {
				vm.stack[fn] = cl
				if argc == cl.Fn.SimpleCall && len(vm.frames) <= evaluator.MaxCallDepth {
					// The usual case, as in RCall
					callee := fn + 1
					top := callee + cl.Fn.NumRegisters
					if top+256 > len(vm.stack) {
						vm.ensureStack(top + 256)
					}
					for i := callee + argc; i < callee+cl.Fn.NumLocals; i++ {
						vm.stack[i] = nil
					}
					if cl.Fn.HasThis {
						vm.stack[callee+argc] = receiver
					}
					vm.receiver = nil
					vm.sp = top
					f.ip = pc
					f = vm.pushFrame(cl, callee)
					words, r, pc = cl.Fn.Code, vm.stack[callee:], 0
					continue
				}
				vm.receiver = receiver
				vm.sp = fn + 1 + argc
				f.ip = pc
				if err := vm.callClosure(cl, argc); err != nil {
					return vm.positioned(f, pc-1, err)
				}
				f = vm.frames[len(vm.frames)-1]
				words, pc = f.cl.Fn.Code, f.ip
				r = vm.stack[f.base:]
				continue
			}
			args := make([]object.Object, argc)
			copy(args, vm.stack[fn+1:fn+1+argc])
			f.ip = pc
			result := evaluator.CallMethod(receiver, name.Value, args, 0, 0)
			r = vm.stack[f.base:]
			if err, ok := result.(*object.Error); ok {
				f.ip = pc
				return vm.positioned(f, pc-1, err)
			}
			if paused, ok := result.(*object.Promise); ok && paused.Paused {
				// A function of another VM, like a loaded file's, paused
				vm.sp = fn
				if s := vm.waitFor(f, pc-1, base, paused); s != nil {
					f.ip = pc
					return vm.awaitInto(f, s, w.A())
				}
				vm.sp = f.base + f.cl.Fn.NumRegisters
				continue
			}
			vm.stack[fn] = result

		case code.RCallSpread, code.RCallMethodSpread:
			if vm.MaxSteps > 0 {
				if err := vm.step(f, pc-1); err != nil {
					f.ip = pc
					return err
				}
			}
			f.ip = pc
			if s := vm.callSpread(f, pc-1, w, base); s != nil {
				return s
			}
			f = vm.frames[len(vm.frames)-1]
			words, pc = f.cl.Fn.Code, f.ip
			r = vm.stack[f.base:]

		case code.RReturn:
			value := r[w.A()]
			if value == nil {
				f.ip = pc
				return vm.notDefinedRegister(f, pc-1, w.A())
			}
			vm.frames = vm.frames[:len(vm.frames)-1]
			// The result goes where the function was, the register of the call
			vm.stack[f.base-1] = value
			if len(vm.frames) == base {
				vm.sp = f.base - 1
				f.ip = pc
				return value
			}
			f = vm.frames[len(vm.frames)-1]
			words, pc = f.cl.Fn.Code, f.ip
			vm.sp = f.base + f.cl.Fn.NumRegisters
			r = vm.stack[f.base:]

		case code.RIterStart:
			it, err := newIterator(r[w.B()], &vm.ints)
			if err != nil {
				f.ip = pc
				return vm.errorAt(f, pc-1, "%s", err.Error())
			}
			r[w.A()] = it

		case code.RRangeIter:
			start, end := r[w.B()], r[w.C()]
			s, okStart := start.(*object.Integer)
			e, okEnd := end.(*object.Integer)
			if !okStart || !okEnd {
				f.ip = pc
				return vm.errorAt(f, pc-1, "range bounds must be integers, got %s and %s", evaluator.TypeName(start), evaluator.TypeName(end))
			}
			r[w.A()] = &rangeIterator{next: s.Value, end: e.Value, ints: &vm.ints}

		case code.RIterNext, code.RIterLoop:
			if op == code.RIterLoop && vm.MaxSteps > 0 {
				if err := vm.step(f, pc-1); err != nil {
					f.ip = pc
					return err
				}
			}
			if it, ok := r[w.A()].(*rangeIterator); ok {
				// Ranges, the usual loops, count here
				if it.done || it.next > it.end {
					if op == code.RIterNext {
						pc = w.C()
					}
					continue
				}
				r[w.B()] = vm.ints.get(it.next)
				it.index++
				if it.next == it.end {
					it.done = true
				} else {
					it.next++
				}
				if op == code.RIterLoop {
					pc = w.C()
				}
				continue
			}
			it := r[w.A()].(iterator)
			key, value, ok := it.Next(it.KeysFirst())
			if !ok {
				if op == code.RIterNext {
					pc = w.C()
				}
				continue
			}
			if it.KeysFirst() {
				r[w.B()] = key
			} else {
				r[w.B()] = value
			}
			if op == code.RIterLoop {
				pc = w.C()
			}

		case code.RIterNextPair:
			it := r[w.A()].(iterator)
			key, value, ok := it.Next(true)
			if !ok {
				pc = w.C()
				continue
			}
			r[w.B()] = key
			r[w.B()+1] = value

		case code.RGetProp:
			name := vm.constants[w.C()].(*object.String)
			obj := r[w.B()]
			// Fast path for objects, through the cache of this instruction;
			// an object without a length property counts its keys
			if hash, ok := obj.(*object.Hash); ok {
				if value, ok := hash.Lookup(name.HashKey(), &f.cl.Fn.Caches[pc-1]); ok {
					r[w.A()] = value
					continue
				}
				if name.Value != "length" {
					r[w.A()] = NULL
					continue
				}
			}
			if obj == nil {
				f.ip = pc
				return vm.notDefinedRegister(f, pc-1, w.B())
			}
			f.ip = pc
			result := evaluator.GetProperty(obj, name.Value, 0, 0)
			r = vm.stack[f.base:]
			if err, ok := result.(*object.Error); ok {
				f.ip = pc
				return vm.positioned(f, pc-1, err)
			}
			r[w.A()] = result

		case code.RSetProp:
			name := vm.constants[w.B()].(*object.String)
			obj, value := r[w.A()], r[w.C()]
			if hash, ok := obj.(*object.Hash); ok {
				if !hash.Update(name.HashKey(), &f.cl.Fn.Caches[pc-1], value) {
					hash.Set(name, value)
				}
				continue
			}
			f.ip = pc
			if err := vm.setProperty(f, pc-1, obj, name.Value, value); err != nil {
				return err
			}
			r = vm.stack[f.base:]

		case code.RIndex, code.RSetIndex:
			// Arrays by position and objects by string keys run here; the
			// rest, with its errors, falls to wordRare
			container, index := r[w.B()], r[w.C()]
			if op == code.RSetIndex {
				container, index = r[w.A()], r[w.B()]
			}
			switch c := container.(type) {
			case *object.Array:
				if i, ok := index.(*object.Integer); ok && i.Value >= 0 && i.Value < int64(len(c.Elements)) {
					if op == code.RIndex {
						r[w.A()] = c.Elements[i.Value]
					} else {
						c.Elements[i.Value] = r[w.C()]
					}
					continue
				}
			case *object.Hash:
				// The keys change, so there's no cache of their places
				if key, ok := index.(*object.String); ok {
					if op == code.RSetIndex {
						c.Set(key, r[w.C()])
						continue
					}
					if pair, ok := c.Get(key.HashKey()); ok {
						r[w.A()] = pair.Value
					} else {
						r[w.A()] = NULL
					}
					continue
				}
			}
			fallthrough

		default:
			f.ip = pc
			if err := vm.wordRare(f, pc-1, w, base); err != nil {
				return err
			}
			f = vm.frames[len(vm.frames)-1]
			words, pc = f.cl.Fn.Code, f.ip
			r = vm.stack[f.base:]
		}
	}
}

// wordRare runs the instructions that are rare enough to stay out of
// loopWords, which Go optimizes better the smaller it is. It returns an
// error or a suspension, or nil.
func (vm *VM) wordRare(f *frame, ip int, w code.Word, base int) object.Object {
	regs := f.base
	r := func(i int) object.Object { return vm.stack[regs+i] }
	set := func(i int, value object.Object) { vm.stack[regs+i] = value }

	switch op := w.Op(); op {
	case code.RInRange:
		value, start, end := r(w.A()), r(w.A()+1), r(w.A()+2)
		above := evaluator.Infix(">=", value, start, 0, 0)
		below := evaluator.Infix("<=", value, end, 0, 0)
		// Values that can't be compared with the ends don't match
		set(w.A(), nativeBool(above == TRUE && below == TRUE))

	case code.RMinus, code.RNot, code.RIncr:
		value := r(w.B())
		if value == nil {
			return vm.notDefinedRegister(f, ip, w.B())
		}
		var result object.Object
		switch op {
		case code.RMinus:
			result = evaluator.Prefix("-", value, 0, 0)
		case code.RNot:
			result = evaluator.Prefix("!", value, 0, 0)
		default:
			operator := "++"
			if w.C() == 1 {
				operator = "--"
			}
			result = evaluator.Increment(value, operator, 0, 0)
		}
		if err, ok := result.(*object.Error); ok {
			return vm.positioned(f, ip, err)
		}
		set(w.A(), result)

	case code.RArray:
		elements := make([]object.Object, w.C())
		copy(elements, vm.stack[regs+w.B():regs+w.B()+w.C()])
		set(w.A(), &object.Array{Elements: elements})

	case code.RHash:
		start := regs + w.B()
		hash, bad := object.NewHashFromPairs(vm.stack[start:start+2*w.C()], &f.cl.Fn.Caches[ip])
		if bad >= 0 {
			return vm.errorAt(f, ip, "%s can't be an object key", evaluator.TypeName(vm.stack[start+bad]))
		}
		set(w.A(), hash)

	case code.RTemplate:
		text := ""
		for i := w.B(); i < w.B()+w.C(); i++ {
			text += evaluator.ToDisplayString(r(i))
		}
		set(w.A(), &object.String{Value: text})

	case code.RAppend:
		array := r(w.A()).(*object.Array)
		value := r(w.B())
		if w.C() == 0 {
			array.Elements = append(array.Elements, value)
			break
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

	case code.RHashSet:
		hash := r(w.A()).(*object.Hash)
		key, value := r(w.B()), r(w.B()+1)
		hashable, ok := key.(object.Hashable)
		if !ok {
			return vm.errorAt(f, ip, "%s can't be an object key", evaluator.TypeName(key))
		}
		hash.Set(hashable, value)

	case code.RHashMerge:
		hash := r(w.A()).(*object.Hash)
		from, ok := r(w.B()).(*object.Hash)
		if !ok {
			return vm.errorAt(f, ip, "cannot spread %s into an object, only objects", evaluator.TypeName(r(w.B())))
		}
		for _, pair := range from.OrderedPairs() {
			hash.Set(pair.Key.(object.Hashable), pair.Value)
		}

	case code.RWithout:
		names := vm.constants[w.C()].(*object.Array)
		from, ok := r(w.B()).(*object.Hash)
		if !ok {
			return vm.errorAt(f, ip, "cannot take the rest of %s, only of an object", evaluator.TypeName(r(w.B())))
		}
		rest := object.NewHash()
	pairs:
		for _, pair := range from.OrderedPairs() {
			for _, name := range names.Elements {
				if pair.Key.(object.Hashable).HashKey() == name.(*object.String).HashKey() {
					continue pairs
				}
			}
			rest.Set(pair.Key.(object.Hashable), pair.Value)
		}
		set(w.A(), rest)

	case code.RIndex:
		container, index := r(w.B()), r(w.C())
		if container == nil {
			return vm.notDefinedRegister(f, ip, w.B())
		}
		if index == nil {
			return vm.notDefinedRegister(f, ip, w.C())
		}
		result := evaluator.Index(container, index, 0, 0)
		if err, ok := result.(*object.Error); ok {
			return vm.positioned(f, ip, err)
		}
		set(w.A(), result)

	case code.RSetIndex:
		result := evaluator.SetIndex(r(w.A()), r(w.B()), r(w.C()), 0, 0)
		if err, ok := result.(*object.Error); ok {
			return vm.positioned(f, ip, err)
		}

	case code.RClosure:
		fn := vm.constants[w.B()].(*object.CompiledFunction)
		free := make([]*object.Cell, w.C())
		for i := range free {
			free[i] = r(w.A() + i).(*object.Cell)
		}
		set(w.A(), &object.Closure{Fn: fn, Free: free, Runner: vm})

	case code.RLoad:
		path := vm.constants[w.B()].(*object.String).Value
		shown := vm.constants[w.C()].(*object.String).Value
		module := vm.load(path, shown, 0, 0)
		if err, ok := module.(*object.Error); ok {
			return vm.positioned(f, ip, err)
		}
		set(w.A(), module)

	case code.RAwait:
		awaitable, ok := r(w.A()).(object.Awaitable)
		if !ok || !awaitable.IsAwaitable() {
			break
		}
		line, column := f.cl.Fn.PositionAt(ip)
		if !vm.canPause(base) {
			// Go code called this function and needs its result now
			return evaluator.NewError(line, column,
				"await can't pause here: this function was called by a builtin such as filter or sort")
		}
		return &suspension{awaited: awaitable, line: line, column: column, target: w.A()}

	case code.RTry:
		f.handlers = append(f.handlers, handler{catch: w.B(), reg: w.A()})

	case code.RTryEnd:
		f.handlers = f.handlers[:len(f.handlers)-w.A()]

	case code.RThrow:
		return vm.thrown(f, ip, r(w.A()))

	case code.RError:
		message := vm.constants[w.A()].(*object.String).Value
		return vm.errorAt(f, ip, "%s", message)

	default:
		return vm.errorAt(f, ip, "unknown instruction %d", int(op))
	}
	return nil
}

// callSpread calls a function or method with the elements of an array as
// its arguments. It returns an error or a suspension, or nil.
func (vm *VM) callSpread(f *frame, ip int, w code.Word, base int) object.Object {
	regs := f.base
	fn := regs + w.A()
	args := vm.stack[fn+1].(*object.Array).Elements
	vm.sp = fn + 1
	if w.Op() == code.RCallMethodSpread {
		name := vm.constants[w.C()].(*object.String)
		receiver := vm.stack[fn]
		// Calls with spread are rare enough to go without a cache
		var cache object.PropertyCache
		if cl := vm.ownMethod(receiver, name, &cache); cl != nil {
			vm.stack[fn] = cl
			for _, arg := range args {
				vm.push(arg)
			}
			vm.receiver = receiver
			if err := vm.callClosure(cl, len(args)); err != nil {
				return vm.positioned(f, ip, err)
			}
			return nil
		}
		result := evaluator.CallMethod(receiver, name.Value, append([]object.Object{}, args...), 0, 0)
		if err, ok := result.(*object.Error); ok {
			return vm.positioned(f, ip, err)
		}
		vm.stack[fn] = result
		vm.sp = fn + 1
	} else {
		for _, arg := range args {
			vm.push(arg)
		}
		switched, err := vm.call(f, ip, len(args))
		if err != nil {
			return err
		}
		if switched {
			return nil
		}
	}
	if s := vm.waitForPausedCall(f, ip, base); s != nil {
		return vm.awaitInto(f, s, w.A())
	}
	vm.sp = regs + f.cl.Fn.NumRegisters
	return nil
}

// awaitInto makes a suspension or error from waitFor put its result in
// register target, and gives the frame its registers back.
func (vm *VM) awaitInto(f *frame, result object.Object, target int) object.Object {
	vm.sp = f.base + f.cl.Fn.NumRegisters
	if s, ok := result.(*suspension); ok {
		s.target = target
		return s
	}
	return result
}

// binaryWords is the slow path of the binary operations of register code,
// with the registers of the operands for the error of a local not set yet
// (-1 for an operand that isn't in a register).
func (vm *VM) binaryWords(f *frame, ip int, op code.Opcode, a int, left object.Object, b int, right object.Object) (object.Object, *object.Error) {
	if left == nil {
		return nil, vm.notDefinedRegister(f, ip, a)
	}
	if right == nil {
		return nil, vm.notDefinedRegister(f, ip, b)
	}
	result := vm.binary(f, ip, op, left, right)
	if err, ok := result.(*object.Error); ok {
		return nil, err
	}
	return result, nil
}

// compareWords is the slow path of the comparisons of register code.
func (vm *VM) compareWords(f *frame, ip int, op code.Opcode, a int, left object.Object, b int, right object.Object) (bool, *object.Error) {
	if left == nil {
		return false, vm.notDefinedRegister(f, ip, a)
	}
	if right == nil {
		return false, vm.notDefinedRegister(f, ip, b)
	}
	return vm.compare(f, ip, op, left, right)
}

// setProperty sets a property of a value that isn't an object.
func (vm *VM) setProperty(f *frame, ip int, obj object.Object, name string, value object.Object) *object.Error {
	result := evaluator.SetProperty(obj, name, value, 0, 0)
	if err, ok := result.(*object.Error); ok {
		return vm.positioned(f, ip, err)
	}
	return nil
}

// notDefinedRegister is the error for reading a register with no value: a
// local read before its declaration ran.
func (vm *VM) notDefinedRegister(f *frame, ip int, r int) *object.Error {
	name := "a value"
	if r >= 0 && r < len(f.cl.Fn.LocalNames) {
		name = f.cl.Fn.LocalNames[r]
	}
	return vm.notDefined(f, ip, name)
}
