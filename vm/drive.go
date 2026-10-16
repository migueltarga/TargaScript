package vm

import (
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
)

// driver runs map, filter, reduce, find, some, every, count or sortBy over
// an array with a function of this VM, in the VM's loop: each call of the
// function returns to it, which notes the result and calls the function
// again with the next element. Going through the evaluator, each call would
// start the loop again. A call of the function of map can pause at an
// await, as there, and then its promise is the element's result; for the
// other methods an await fails, as there.
type driver struct {
	method string
	arr    *object.Array
	cl     *object.Closure
	elems  []object.Object
	i      int // the element of the call running
	out    []object.Object
	acc    object.Object // the accumulator of reduce, or the result once known
	count  int64
}

// drive returns a driver for receiver.method with argc arguments from the
// register after fn, when the VM can run it, or nil.
func (vm *VM) drive(receiver object.Object, method string, argc, fn int) *driver {
	arr, ok := receiver.(*object.Array)
	if !ok || len(vm.frames) > evaluator.MaxCallDepth {
		return nil
	}
	switch method {
	case "map", "filter", "find", "some", "every", "count", "sortBy":
		if argc != 1 {
			return nil
		}
	case "reduce":
		if argc != 2 {
			return nil
		}
	default:
		return nil
	}
	cl, ok := vm.stack[fn+1].(*object.Closure)
	if !ok || cl.Runner != vm || cl.Fn.SimpleCall < 0 || cl.Fn.HasThis {
		return nil
	}
	d := &driver{method: method, arr: arr, cl: cl, elems: arr.Elements}
	switch method {
	case "reduce":
		d.acc = vm.stack[fn+2]
	case "map", "filter", "sortBy":
		d.out = make([]object.Object, 0, len(arr.Elements))
	}
	return d
}

// result is what the method returns, once done, or its error.
func (d *driver) result() object.Object {
	switch d.method {
	case "sortBy":
		// The keys, in the order of the elements
		return evaluator.SortByKeys(d.arr, d.out)
	case "map", "filter":
		return &object.Array{Elements: d.out}
	case "count":
		return &object.Integer{Value: d.count}
	}
	if d.acc != nil {
		return d.acc
	}
	// No element decided it
	switch d.method {
	case "some":
		return FALSE
	case "every":
		return TRUE
	}
	return NULL
}

// enter starts the call of the function for the current element, with its
// registers from base, as map, filter and reduce pass its arguments: the
// element, and its index when the function takes it, after the
// accumulator for reduce.
func (vm *VM) enter(d *driver, base int) *frame {
	fn := d.cl.Fn
	vm.ensureStack(base + fn.NumRegisters + 256)
	vm.stack[base-1] = d.cl
	var args [3]object.Object
	n := 0
	if d.method == "reduce" {
		args[n] = d.acc
		n++
	}
	args[n] = d.elems[d.i]
	n++
	if fn.NumParams > n {
		args[n] = vm.ints.get(int64(d.i))
		n++
	}
	for p := 0; p < fn.NumParams; p++ {
		value := object.Object(NULL)
		if p < n {
			value = args[p]
		}
		vm.stack[base+p] = value
	}
	for i := base + fn.NumParams; i < base+fn.NumLocals; i++ {
		vm.stack[i] = nil
	}
	vm.receiver = nil
	vm.sp = base + fn.NumRegisters
	f := vm.pushFrame(d.cl, base)
	f.driver = d
	return f
}

// returned notes the result of a call of the function, and reports
// whether the method is done.
func (d *driver) returned(value object.Object) bool {
	switch d.method {
	case "map", "sortBy":
		d.out = append(d.out, value)
	case "filter":
		if evaluator.IsTruthy(value) {
			d.out = append(d.out, d.elems[d.i])
		}
	case "reduce":
		d.acc = value
	case "count":
		if evaluator.IsTruthy(value) {
			d.count++
		}
	case "find":
		if evaluator.IsTruthy(value) {
			d.acc = d.elems[d.i]
			return true
		}
	case "some":
		if evaluator.IsTruthy(value) {
			d.acc = TRUE
			return true
		}
	case "every":
		if !evaluator.IsTruthy(value) {
			d.acc = FALSE
			return true
		}
	}
	d.i++
	return d.i >= len(d.elems)
}

// countCall counts a call of the function against MaxSteps, as calls from
// the evaluator are.
func (vm *VM) countCall() *object.Error {
	if vm.MaxSteps > 0 {
		vm.steps++
		if vm.steps > vm.MaxSteps {
			return evaluator.NewError(0, 0, "the program took more than %d steps", vm.MaxSteps)
		}
	}
	return nil
}

// innerDriver returns the index of the innermost frame from base that runs
// the function of a method for a driver, or -1.
func (vm *VM) innerDriver(base int) int {
	for i := len(vm.frames) - 1; i >= base; i-- {
		if vm.frames[i].driver != nil {
			return i
		}
	}
	return -1
}

// pauseCall pauses, when the run from base stopped at an await in a call
// of the function of map, that call alone, as the evaluator's map does: its
// frames go in a fiber of their own, its promise is the element's result,
// and map goes on with the next element. It reports whether it did.
func (vm *VM) pauseCall(base int, s *suspension) bool {
	at := vm.innerDriver(base)
	if at < 0 {
		return false
	}
	call := vm.frames[at]
	d, callBase := call.driver, call.base
	call.driver = nil
	fb := &fiber{promise: &object.Promise{Paused: true}}
	vm.suspend(at, fb, s)
	if !d.returned(fb.promise) {
		vm.enter(d, callBase)
		return true
	}
	vm.stack[callBase-1] = d.result()
	caller := vm.frames[len(vm.frames)-1]
	vm.sp = caller.base + caller.cl.Fn.NumRegisters
	return true
}
