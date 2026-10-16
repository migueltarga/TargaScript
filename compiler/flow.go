package compiler

import "github.com/migueltarga/TargaScript/code"

// emitFlow is what the compiler works out about the stack code of a
// function while it makes it, for the translation to register code: how
// deep the stack is at each instruction, and where each break goes.
type emitFlow struct {
	depths []int // values on the stack, or -1 when nothing reaches it
	breaks map[int]*breakFlow

	depth int  // values on the stack now
	live  bool // whether the next instruction can run
	tries int  // try blocks open now

	// arrivals are the depths and try blocks with which jumps arrive at
	// instructions not made yet
	arrivals map[int]arrival
	// jumps are, for jumps whose target isn't known yet, the depth and
	// try blocks with which they arrive there
	jumps map[int]arrival
	loops []*loopFlow
	err   error
}

type arrival struct{ depth, tries int }

// loopFlow is a loop being compiled: the depth and try blocks it started
// with, and its breaks, to send to its end.
type loopFlow struct {
	depth, tries int
	breaks       []*breakFlow
}

// breakFlow is a break or continue: where it goes, the depth of its loop,
// and how many try blocks it leaves.
type breakFlow struct {
	target, depth, tries int
	continues            bool
}

func newEmitFlow() *emitFlow {
	return &emitFlow{live: true, breaks: map[int]*breakFlow{}, arrivals: map[int]arrival{}, jumps: map[int]arrival{}}
}

func (f *emitFlow) fail(message string, at int) {
	if f.err == nil {
		f.err = flowError(message, at)
	}
}

// arrive notes that a jump from at reaches target with depth and tries.
func (f *emitFlow) arrive(at, target, depth, tries int) {
	switch {
	case target == placeholder:
		f.jumps[at] = arrival{depth, tries}
	case target < len(f.depths):
		// Back to an instruction made already
		if f.depths[target] >= 0 && f.depths[target] != depth {
			f.fail("different stack depths at", target)
		}
	default:
		if a, ok := f.arrivals[target]; ok && a != (arrival{depth, tries}) {
			f.fail("different stack depths at", target)
		}
		f.arrivals[target] = arrival{depth, tries}
	}
}

// placeholder is the target of a jump made before its target is known.
const placeholder = 9999

// retarget notes the target a jump at gets once it's known.
func (f *emitFlow) retarget(at, target int) {
	if a, ok := f.jumps[at]; ok {
		delete(f.jumps, at)
		f.arrive(at, target, a.depth, a.tries)
	}
}

// add follows the instruction at, made next.
func (f *emitFlow) add(at int, ins instruction) {
	if a, ok := f.arrivals[at]; ok {
		delete(f.arrivals, at)
		if f.live && (f.depth != a.depth || f.tries != a.tries) {
			f.fail("different stack depths at", at)
		}
		f.live, f.depth, f.tries = true, a.depth, a.tries
	}
	// Loops open and close where they're written, even in code nothing
	// reaches
	switch ins.op {
	case code.OpLoopEnter:
		f.loops = append(f.loops, &loopFlow{depth: f.depth, tries: f.tries})
	case code.OpLoopExit:
		if len(f.loops) == 0 {
			f.fail("loop exit outside a loop at", at)
		} else {
			f.loops = f.loops[:len(f.loops)-1]
		}
	}
	if !f.live {
		f.depths = append(f.depths, -1)
		return
	}
	f.depths = append(f.depths, f.depth)
	d := f.depth
	switch ins.op {
	case code.OpReturnValue, code.OpThrow, code.OpError:
		f.live = false
	case code.OpJump:
		f.arrive(at, ins.args[0], d, f.tries)
		f.live = false
	case code.OpJumpNotTruthy:
		f.arrive(at, ins.args[0], d-1, f.tries)
		f.depth--
	case code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep, code.OpJumpNotNullKeep:
		f.arrive(at, ins.args[0], d, f.tries)
		f.depth--
	case code.OpJumpNullKeep:
		f.arrive(at, ins.args[0], d, f.tries)
	case code.OpLoopEnter, code.OpLoopExit:
	case code.OpBreak:
		if len(f.loops) == 0 {
			f.fail("break outside a loop at", at)
			return
		}
		loop := f.loops[len(f.loops)-1]
		b := &breakFlow{depth: loop.depth, tries: f.tries - loop.tries, target: placeholder, continues: ins.args[0] == 1}
		f.breaks[at] = b
		loop.breaks = append(loop.breaks, b)
		f.live = false
	case code.OpIterNext:
		f.arrive(at, ins.args[0], d, f.tries)
		f.depth += 1 + ins.args[1]
	case code.OpTry:
		// The catch block starts with the error on the stack, and its try
		// block closed
		f.arrive(at, ins.args[0], d+1, f.tries)
		f.tries++
	case code.OpTryEnd:
		if f.tries == 0 {
			f.fail("try end outside a try at", at)
		}
		f.tries--
	default:
		effect, ok := stackEffect(ins.op, ins.args)
		if !ok {
			f.fail("no stack effect for "+definitionName(ins.op)+" at", at)
			return
		}
		f.depth += effect
		if f.depth < 0 {
			f.fail("stack underflow at", at)
		}
	}
}

// endLoop sends the breaks of the innermost loop to its end, and its
// continues to its start, before the end is made.
func (f *emitFlow) endLoop(end, start int) {
	if len(f.loops) == 0 {
		return
	}
	loop := f.loops[len(f.loops)-1]
	for _, b := range loop.breaks {
		b.target = end
		if b.continues {
			b.target = start
		}
		f.arrive(-1, b.target, loop.depth, loop.tries)
	}
}
