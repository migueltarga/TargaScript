package compiler

import (
	"errors"
	"strconv"

	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/object"
)

// The register machine.
//
// The compiler makes code for a stack machine, which is simple to make
// right: x + 1 pushes x, pushes 1, and adds the top two values. translate
// turns it into the code the VM runs, for a register machine: x + 1 becomes
// one instruction that reads the register of x and the number 1 and writes
// the result in a register. Fewer instructions run, and the values don't go
// through a stack.
//
// A function's registers are its locals, then one for each place its stack
// code would use: the value the stack code keeps at depth i is in register
// locals + i, so the translation knows where every value is from the stack
// depth of each instruction, which analyze works out.
//
// Better still, a value doesn't have to be moved to its place at once: the
// translation keeps, for each value on the stack, where it is now, like a
// local or a constant, and the instruction that uses it reads it from
// there. Values are moved to their places only where the code joins, at
// jumps and where they go, and where instructions need values one after
// the other, like the arguments of a call.

// where is the place of a value the stack code would have pushed: a
// register, or a constant.
type where struct {
	register int
	constant int // index of the constant, or -1
}

func inRegister(r int) where { return where{register: r, constant: -1} }

// translation is the state of translating one function.
type translation struct {
	in    instructions
	flow  *emitFlow
	c     *Compiler
	fn    *functionScope
	scope *compilation
	at    int // the next stack instruction to translate

	out       []code.Word
	positions []object.Position
	oldPos    []object.Position
	newAt     []int // newAt[old offset] is the index of its first new instruction
	fixups    []fixup

	stack []where
	// lastWrite is the index of the last instruction emitted that only
	// writes its result to register A, for an assignment after it to make
	// it write the local instead
	lastWrite int
	// iterNext is the index of the instruction just emitted that takes
	// the next value of an iterator, for an assignment of it to make it
	// write the variable instead, or -1
	iterNext int
	// layouts are where the values are at the targets of jumps that leave
	// some in the variables or constants they came from
	layouts map[int][]where
	// straight is where the code that runs straight to the end starts,
	// with no jumps from or to it
	straight int
}

// fixup is a jump to point at the new place of its target: the operand
// (0 for A, 1 for B, 2 for C) of instruction at, going where the stack
// instruction src goes, which may be set after it's translated.
type fixup struct {
	at, operand, src int
}

// translationError stops a translation that met code it can't handle.
type translationError struct{ message string }

// newTranslation starts translating the stack code of a function to
// register code, which advance does as the compiler makes it.
func newTranslation(c *Compiler, fn *functionScope, scope *compilation) *translation {
	return &translation{c: c, fn: fn, scope: scope, flow: scope.flow, lastWrite: -1, iterNext: -1, layouts: map[int][]where{}}
}

// advance translates the instructions made so far that can be: the ones
// whose next instruction is made, and for jumps whose target is. At the end
// of the function, final translates the rest.
func (t *translation) advance(final bool) {
	if t.flow.err != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(translationError)
			if !ok {
				panic(r)
			}
			t.flow.err = errors.New(e.message)
		}
	}()
	t.in, t.oldPos = t.scope.instructions, t.scope.positions
	for t.at < len(t.in) && (final || t.ready(t.at)) {
		t.at = t.step(t.at)
	}
}

// ready reports whether the instruction at can be translated now.
func (t *translation) ready(at int) bool {
	if at+1 >= len(t.in) {
		return false
	}
	switch t.in[at].op {
	case code.OpJump, code.OpJumpNotTruthy, code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep,
		code.OpJumpNotNullKeep, code.OpJumpNullKeep:
		// They look at their target
		target := t.in[at].args[0]
		return target != placeholder && target < len(t.in)
	}
	return true
}

// finish translates what's left of the function and returns its register
// code, its positions, and how many registers it needs.
func (t *translation) finish(numLocals int) (out []code.Word, pos []object.Position, registers int, err error) {
	t.advance(true)
	if t.flow.err != nil {
		return nil, nil, 0, t.flow.err
	}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(translationError)
			if !ok {
				panic(r)
			}
			err = errors.New(e.message)
		}
	}()
	t.end()
	relocate(t.out, numLocals)
	return t.out, t.positions, numLocals + maxDepth(t.flow.depths) + 2, nil
}

func maxDepth(depths []int) int {
	most := 0
	for _, d := range depths {
		most = max(most, d)
	}
	return most
}

func (t *translation) fail(message string, at int) {
	panic(translationError{message + " " + strconv.Itoa(at)})
}

// own is the register of the value at depth i.
func (t *translation) own(i int) int { return tempBase + i }

func (t *translation) push(w where) { t.stack = append(t.stack, w) }

func (t *translation) pop() where {
	if len(t.stack) == 0 {
		t.fail("stack underflow at", len(t.out))
	}
	w := t.stack[len(t.stack)-1]
	t.stack = t.stack[:len(t.stack)-1]
	return w
}

// emit appends an instruction made for the stack instruction at old.
func (t *translation) emit(old int, op code.Op, a, b, c int) int {
	at := len(t.out)
	t.out = append(t.out, code.MakeWord(op, a, b, c))
	line, column := positionAt(t.oldPos, old)
	if n := len(t.positions); n == 0 || t.positions[n-1].Line != line || t.positions[n-1].Column != column {
		t.positions = append(t.positions, object.Position{Offset: at, Line: line, Column: column})
	}
	t.lastWrite, t.iterNext = -1, -1
	return at
}

// emitWrite appends an instruction that only writes R[A], which an
// assignment right after can point at a local instead.
func (t *translation) emitWrite(old int, op code.Op, a, b, c int) {
	t.emit(old, op, a, b, c)
	t.lastWrite = len(t.out) - 1
}

// load puts a value in register r.
func (t *translation) load(old int, r int, w where) {
	if w.constant >= 0 {
		if n, ok := t.c.constants[w.constant].(*object.Integer); ok && n.Value >= -32768 && n.Value <= 32767 {
			t.emitWrite(old, code.RLoadI, r, int(n.Value), 0)
		} else {
			t.emitWrite(old, code.RLoadK, r, w.constant, 0)
		}
		return
	}
	if w.register != r {
		t.emitWrite(old, code.RMove, r, w.register, 0)
	}
}

// register returns a register holding a value, loading a constant into
// spare, the register of the value's own place.
func (t *translation) register(old int, w where, spare int) int {
	if w.constant < 0 {
		return w.register
	}
	t.keep(old, spare)
	t.load(old, spare, w)
	return spare
}

// keep moves the values that are only noted as in register r to their
// own places, before r changes.
func (t *translation) keep(old int, r int) {
	for i, w := range t.stack {
		if w.constant < 0 && w.register == r && t.own(i) != r {
			t.load(old, t.own(i), w)
			t.stack[i] = inRegister(t.own(i))
		}
	}
}

// place moves the values from depth from up to their own places, for
// instructions that take values one after the other.
func (t *translation) place(old int, from int) {
	for i := from; i < len(t.stack); i++ {
		w := t.stack[i]
		if w.constant >= 0 || w.register != t.own(i) {
			t.keep(old, t.own(i))
			t.load(old, t.own(i), w)
			t.stack[i] = inRegister(t.own(i))
		}
	}
}

// keepBelow moves the values below depth from that are noted as in
// registers from r up to their own places, before those registers change.
func (t *translation) keepBelow(old int, from, r int) {
	for i := 0; i < from && i < len(t.stack); i++ {
		w := t.stack[i]
		if w.constant < 0 && w.register >= r && t.own(i) != w.register {
			t.load(old, t.own(i), w)
			t.stack[i] = inRegister(t.own(i))
		}
	}
}

// secure moves a value taken off the stack to spare, its own place, when
// it is noted as in the place of a value below it, which place may change.
func (t *translation) secure(old int, w where, spare int) where {
	if w.constant < 0 && w.register >= tempBase && w.register < spare {
		t.keep(old, spare)
		t.load(old, spare, w)
		return inRegister(spare)
	}
	return w
}

// referenced reports whether a value on the stack is noted as in register r.
func (t *translation) referenced(r int) bool {
	for _, w := range t.stack {
		if w.constant < 0 && w.register == r {
			return true
		}
	}
	return false
}

func (t *translation) u16(at, n int) int { return t.in.arg(at, n) }

func (t *translation) jump(old int, op code.Op, a, b, c, operand, src int) {
	at := t.emit(old, op, a, b, c)
	t.fixups = append(t.fixups, fixup{at, operand, src})
	t.straight = len(t.out)
}

// smallInt returns the value of a constant that fits an instruction.
func (t *translation) smallInt(w where) (int, bool) {
	if w.constant < 0 {
		return 0, false
	}
	n, ok := t.c.constants[w.constant].(*object.Integer)
	if !ok || n.Value < -32768 || n.Value > 32767 {
		return 0, false
	}
	return int(n.Value), true
}

// binaryOp returns the register operation of a binary operator, and its
// form with a constant (0 when it has none).
func binaryOp(op code.Opcode) (code.Op, code.Op) {
	switch op {
	case code.OpAdd:
		return code.RAdd, code.RAddK
	case code.OpSub:
		return code.RSub, code.RSubK
	case code.OpMul:
		return code.RMul, code.RMulK
	case code.OpDiv:
		return code.RDiv, code.RDivK
	case code.OpMod:
		return code.RMod, code.RModK
	case code.OpPow:
		return code.RPow, 0
	case code.OpXor:
		return code.RXor, 0
	case code.OpEqual:
		return code.REqual, 0
	case code.OpNotEqual:
		return code.RNotEqual, 0
	case code.OpGreaterThan:
		return code.RGreater, 0
	case code.OpGreaterEqual:
		return code.RGreaterEqual, 0
	case code.OpLessThan:
		return code.RLess, 0
	case code.OpLessEqual:
		return code.RLessEqual, 0
	case code.OpRange:
		return code.RRange, 0
	}
	return code.RIn, 0
}

// jumpOps returns the comparisons that jump when false, in the forms with
// two registers, a small integer, and a constant (0 when it has none), and
// whether op is a comparison.
func jumpOps(op code.Opcode) ([3]code.Op, bool) {
	switch op {
	case code.OpLessThan:
		return [3]code.Op{code.RJumpLess, code.RJumpLessI, 0}, true
	case code.OpLessEqual:
		return [3]code.Op{code.RJumpLessEqual, code.RJumpLessEqualI, 0}, true
	case code.OpGreaterThan:
		return [3]code.Op{code.RJumpGreater, code.RJumpGreaterI, 0}, true
	case code.OpGreaterEqual:
		return [3]code.Op{code.RJumpGreaterEqual, code.RJumpGreaterEqualI, 0}, true
	case code.OpEqual:
		return [3]code.Op{code.RJumpEqual, code.RJumpEqualI, code.RJumpEqualK}, true
	case code.OpNotEqual:
		return [3]code.Op{code.RJumpNotEqual, code.RJumpNotEqualI, code.RJumpNotEqualK}, true
	}
	return [3]code.Op{}, false
}

// step translates the stack instruction at, and returns the next one to
// translate.
func (t *translation) step(at int) int {
	in := t.in
	op := in[at].op
	next := at + 1
	t.setNewAt(at)
	d := t.flow.depths[at]
	if d < 0 {
		// Nothing reaches this code
		return next
	}
	if layout := t.layouts[at]; layout != nil {
		// Jumps come here with the values the layout keeps where they
		// came from, and the others in their own places
		if len(t.stack) == d {
			t.placeFor(at, layout, at)
		}
		t.stack = append(t.stack[:0], layout...)
		t.lastWrite = -1
		t.straight = len(t.out)
	} else if t.flow.targets[at] {
		// Jumps come here with every value in its own place
		if len(t.stack) == d {
			t.placeFor(at, nil, at)
		}
		t.stack = t.stack[:0]
		for i := 0; i < d; i++ {
			t.push(inRegister(t.own(i)))
		}
		t.lastWrite = -1
		t.straight = len(t.out)
	}
	if len(t.stack) != d {
		t.fail("wrong stack depth at", at)
	}
	t.setNewAt(at)
	next = t.instruction(at, op, next, d)
	switch op {
	case code.OpJump, code.OpReturnValue, code.OpThrow, code.OpError, code.OpBreak:
		// What comes next is reached only by jumps, with every value
		// in its place
		t.stack = t.stack[:0]
		if next < len(in) {
			for i := 0; i < t.flow.depths[next]; i++ {
				t.push(inRegister(t.own(i)))
			}
		}
		t.lastWrite = -1
	}
	return next
}

// setNewAt notes that the stack instruction at starts at the end of the
// register code.
func (t *translation) setNewAt(at int) {
	for len(t.newAt) <= at {
		t.newAt = append(t.newAt, -1)
	}
	t.newAt[at] = len(t.out)
}

// end points the jumps at their targets, now that all are known, and
// shortens the code.
func (t *translation) end() {
	t.setNewAt(len(t.in))
	for _, f := range t.fixups {
		src := t.in[f.src]
		target := src.args[0]
		if src.op == code.OpBreak {
			target = t.flow.breaks[f.src].target
		}
		at := t.newAt[target]
		if at < 0 {
			t.fail("jump to code not translated at", target)
		}
		w := t.out[f.at]
		a, b, c := w.A(), w.B(), w.C()
		switch f.operand {
		case 0:
			a = at
		case 1:
			b = at
		default:
			c = at
		}
		t.out[f.at] = code.MakeWord(w.Op(), a, b, c)
	}
	t.shortenReturns()
	t.loopAtEnd()
}

// loopAtEnd makes the jump back to the start of a loop over an iterator,
// right before the loop's end, take the next value itself and go on with
// the body.
func (t *translation) loopAtEnd() {
	for i, w := range t.out {
		if w.Op() != code.RJump || w.A() >= i {
			continue
		}
		next := t.out[w.A()]
		if next.Op() == code.RIterNext && next.C() == i+1 {
			t.out[i] = code.MakeWord(code.RIterLoop, next.A(), next.B(), w.A()+1)
		}
	}
}

// shortenReturns returns values where they are: a jump to a return
// becomes the return, and a move into the register a return reads becomes
// the return. The instructions they stand for stay, for other jumps to them.
func (t *translation) shortenReturns() {
	for i, w := range t.out {
		if w.Op() == code.RJump && t.out[w.A()].Op() == code.RReturn {
			t.out[i] = t.out[w.A()]
		}
	}
	for i, w := range t.out {
		if w.Op() == code.RMove && i+1 < len(t.out) && t.out[i+1].Op() == code.RReturn && t.out[i+1].A() == w.A() {
			t.out[i] = code.MakeWord(code.RReturn, w.B(), 0, 0)
		}
	}
}

// globalLoad returns the index of the instruction that loaded the global
// in register r, the function of a call, when the instructions after it
// can't change it or see it, so the call can load it itself; or -1.
func (t *translation) globalLoad(r int) int {
	for i := len(t.out) - 1; i >= t.straight; i-- {
		w := t.out[i]
		if w.A() == r {
			if w.Op() == code.RGetGlobal {
				return i
			}
			return -1
		}
		switch w.Op() {
		case code.RLoadK, code.RLoadI, code.RLoadNull, code.RLoadTrue, code.RLoadFalse, code.RGetBuiltin, code.RGetGlobal:
		case code.RMove, code.RAddI, code.RSubI:
			if w.B() == r {
				return -1
			}
		default:
			return -1
		}
	}
	return -1
}

// remove takes out the instruction at i, in the code that runs straight
// to the end.
func (t *translation) remove(i int) {
	t.out = append(t.out[:i], t.out[i+1:]...)
	for old, at := range t.newAt {
		if at > i {
			t.newAt[old] = at - 1
		}
	}
	for j := 0; j < len(t.positions); j++ {
		p := &t.positions[j]
		if p.Offset == i && (j+1 == len(t.positions) || t.positions[j+1].Offset == i+1) {
			// The instruction after it starts a position of its own
			t.positions = append(t.positions[:j], t.positions[j+1:]...)
			j--
			continue
		}
		if p.Offset > i {
			p.Offset--
		}
	}
}

// keptLayout returns where the values on the stack can be at the target
// of a jump that skips the rest of an expression: in their own places, but
// for the ones in variables nothing in between changes, or constants. The
// value on top always goes in its own place.
func (t *translation) keptLayout(at, next, target, d int) []where {
	layout := make([]where, d)
	for i := range layout {
		layout[i] = inRegister(t.own(i))
	}
	// Only code between the jump and its target may jump into it
	for _, j := range jumpList(t.in) {
		if j[1] > at && j[1] <= target && (j[0] < at || j[0] >= target) {
			return layout
		}
	}
	written := map[int]bool{}
	for i := next; i < target; i++ {
		switch t.in[i].op {
		case code.OpSetLocal, code.OpNewCell:
			written[t.u16(i, 0)] = true
		}
	}
	for i := 0; i < d-1 && i < len(t.stack); i++ {
		w := t.stack[i]
		switch {
		case w.constant >= 0:
			layout[i] = w
		case w.register < tempBase && !written[w.register] && !(w.register < len(t.fn.cells) && t.fn.cells[w.register]):
			layout[i] = w
		}
	}
	return layout
}

// placeFor moves the values a layout of target has in their own places
// there, or all the values without a layout. A value the target pops
// right away needs no place; a target of -1 has no such value.
func (t *translation) placeFor(old int, layout []where, target int) {
	n := len(t.stack)
	if target >= 0 && target < len(t.in) && t.in[target].op == code.OpPop && t.flow.depths[target] == n {
		n--
	}
	for i := 0; i < n; i++ {
		if i < len(layout) && layout[i] != inRegister(t.own(i)) {
			// The value is where the layout has it, or a copy of it
			continue
		}
		w := t.stack[i]
		if w.constant >= 0 || w.register != t.own(i) {
			t.keep(old, t.own(i))
			t.load(old, t.own(i), w)
			t.stack[i] = inRegister(t.own(i))
		}
	}
}

// fused reports whether the instruction at next is op and nothing jumps
// to it, so the one before can do its work.
func (t *translation) fused(next int, op code.Opcode) bool {
	return next < len(t.in) && !t.flow.targets[next] && t.in[next].op == op
}

// instruction translates the stack instruction at at, and returns where
// the next one to translate is.
func (t *translation) instruction(at int, op code.Opcode, next int, d int) int {
	switch op {
	case code.OpConstant:
		t.push(where{constant: t.u16(at, 0)})

	case code.OpGetLocal:
		x := t.u16(at, 0)
		if x < len(t.fn.cells) && t.fn.cells[x] {
			t.keep(at, t.own(d))
			t.emitWrite(at, code.RGetCell, t.own(d), x, 0)
			t.push(inRegister(t.own(d)))
		} else {
			t.push(inRegister(x))
		}

	case code.OpTrue, code.OpFalse, code.OpNull:
		t.keep(at, t.own(d))
		rop := code.RLoadNull
		if op == code.OpTrue {
			rop = code.RLoadTrue
		} else if op == code.OpFalse {
			rop = code.RLoadFalse
		}
		t.emitWrite(at, rop, t.own(d), 0, 0)
		t.push(inRegister(t.own(d)))

	case code.OpGetGlobal, code.OpGetBuiltin, code.OpGetFree, code.OpGetFreeCell:
		t.keep(at, t.own(d))
		operand := t.u16(at, 0)
		if op == code.OpGetFree || op == code.OpGetFreeCell {
			operand = t.u16(at, 0)
		}
		rop := code.RGetGlobal
		switch op {
		case code.OpGetBuiltin:
			rop = code.RGetBuiltin
		case code.OpGetFree:
			rop = code.RGetFree
		case code.OpGetFreeCell:
			rop = code.RFreeCell
		}
		t.emitWrite(at, rop, t.own(d), operand, 0)
		t.push(inRegister(t.own(d)))

	case code.OpGetLocalCell:
		// The cell itself, to capture it
		t.keep(at, t.own(d))
		t.emitWrite(at, code.RLocalCell, t.own(d), t.u16(at, 0), 0)
		t.push(inRegister(t.own(d)))

	case code.OpPop:
		t.pop()

	case code.OpDup:
		w := t.pop()
		t.push(w)
		t.push(w)

	case code.OpDup2:
		a, b := t.stack[d-2], t.stack[d-1]
		t.push(a)
		t.push(b)

	case code.OpSlide:
		n := t.u16(at, 0)
		top := t.pop()
		t.stack = t.stack[:len(t.stack)-n]
		t.push(top)

	case code.OpSetLocal:
		x := t.u16(at, 0)
		value := t.pop()
		if x < len(t.fn.cells) && t.fn.cells[x] {
			r := t.register(at, value, t.own(d-1))
			t.emit(at, code.RSetCell, x, r, 0)
			t.push(inRegister(r))
			break
		}
		if it := t.iterNext; it >= 0 {
			t.iterNext = -1
			if value.constant < 0 && value.register == t.out[it].B() && !t.referenced(x) {
				// The iterator writes x itself
				w := t.out[it]
				t.out[it] = code.MakeWord(w.Op(), w.A(), x, w.C())
				t.push(inRegister(x))
				break
			}
		}
		// The values noted as in x keep the value x had; with none, the
		// instruction that made the value can write x itself
		if t.lastWrite >= 0 && value.constant < 0 && value.register == t.own(d-1) &&
			t.out[t.lastWrite].A() == t.own(d-1) && !t.referenced(x) {
			// The instruction that made the value writes x instead
			w := t.out[t.lastWrite]
			t.out[t.lastWrite] = code.MakeWord(w.Op(), x, w.B(), w.C())
		} else {
			t.keep(at, x)
			t.load(at, x, value)
		}
		t.lastWrite = -1
		t.push(inRegister(x))

	case code.OpNewCell:
		x := t.u16(at, 0)
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		t.emit(at, code.RNewCell, x, r, 0)
		t.push(inRegister(r))

	case code.OpSetGlobal, code.OpSetFree:
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		operand := t.u16(at, 0)
		rop := code.RSetGlobal
		if op == code.OpSetFree {
			operand, rop = t.u16(at, 0), code.RSetFree
		}
		t.emit(at, rop, r, operand, 0)
		t.push(inRegister(r))

	case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow, code.OpXor,
		code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterEqual,
		code.OpLessThan, code.OpLessEqual, code.OpRange, code.OpIn:
		return t.binary(at, op, next, d)

	case code.OpInRange:
		t.place(at, d-3)
		t.pop()
		t.pop()
		t.pop()
		t.emit(at, code.RInRange, t.own(d-3), 0, 0)
		t.push(inRegister(t.own(d - 3)))

	case code.OpMinus, code.OpBang, code.OpIncrement:
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		t.keep(at, t.own(d-1))
		rop, c := code.RMinus, 0
		switch op {
		case code.OpBang:
			rop = code.RNot
		case code.OpIncrement:
			rop, c = code.RIncr, t.u16(at, 0)
		}
		t.emitWrite(at, rop, t.own(d-1), r, c)
		t.push(inRegister(t.own(d - 1)))

	case code.OpJump:
		t.placeFor(at, nil, t.u16(at, 0))
		t.jump(at, code.RJump, 0, 0, 0, 0, at)

	case code.OpJumpNotTruthy:
		value := t.secure(at, t.pop(), t.own(d-1))
		t.placeFor(at, nil, t.u16(at, 0))
		r := t.register(at, value, t.own(d-1))
		t.jump(at, code.RJumpFalsy, r, 0, 0, 1, at)

	case code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep, code.OpJumpNotNullKeep, code.OpJumpNullKeep:
		// The value goes with the jump, in its own place; the ones below
		// it can stay where they are
		target := t.u16(at, 0)
		layout := t.layouts[target]
		if layout == nil && t.flow.depths[target] == d {
			layout = t.keptLayout(at, next, target, d)
			t.layouts[target] = layout
		}
		// The value tested goes in its place even if the target pops it
		t.placeFor(at, layout, -1)
		rop := code.RJumpFalsy
		switch op {
		case code.OpJumpTruthyKeep:
			rop = code.RJumpTruthy
		case code.OpJumpNotNullKeep:
			rop = code.RJumpNotNull
		case code.OpJumpNullKeep:
			rop = code.RJumpNull
		}
		t.jump(at, rop, t.own(d-1), 0, 0, 1, at)
		if op != code.OpJumpNullKeep {
			t.pop()
		}

	case code.OpLoopEnter, code.OpLoopExit:
		// Loops need no instructions: break and continue are jumps

	case code.OpBreak:
		b := t.flow.breaks[at]
		if b.tries > 0 {
			t.emit(at, code.RTryEnd, b.tries, 0, 0)
		}
		// The values the loop started with are in their places already,
		// as the loop's start is a place jumps go to
		t.stack = t.stack[:b.depth]
		t.place(at, 0)
		t.jump(at, code.RJump, 0, 0, 0, 0, at)

	case code.OpIterStart:
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		t.keep(at, t.own(d-1))
		t.emitWrite(at, code.RIterStart, t.own(d-1), r, 0)
		t.push(inRegister(t.own(d - 1)))

	case code.OpRangeIter:
		end, start := t.pop(), t.pop()
		rs := t.register(at, start, t.own(d-2))
		re := t.register(at, end, t.own(d-1))
		t.keep(at, t.own(d-2))
		t.emitWrite(at, code.RRangeIter, t.own(d-2), rs, re)
		t.push(inRegister(t.own(d - 2)))

	case code.OpIterNext:
		// The loop's start, so every value is in its place
		rop := code.RIterNext
		if t.u16(at, 1) == 1 {
			rop = code.RIterNextPair
		}
		t.jump(at, rop, t.own(d-1), t.own(d), 0, 2, at)
		t.push(inRegister(t.own(d)))
		if rop == code.RIterNextPair {
			t.push(inRegister(t.own(d + 1)))
		} else if t.fused(next, code.OpSetLocal) {
			t.iterNext = len(t.out) - 1
		}

	case code.OpArray, code.OpTemplate:
		n := t.u16(at, 0)
		t.keepBelow(at, d-n, t.own(d-n))
		t.place(at, d-n)
		t.stack = t.stack[:d-n]
		rop := code.RArray
		if op == code.OpTemplate {
			rop = code.RTemplate
		}
		t.emitWrite(at, rop, t.own(d-n), t.own(d-n), n)
		t.push(inRegister(t.own(d - n)))

	case code.OpHash:
		n := t.u16(at, 0)
		t.keepBelow(at, d-2*n, t.own(d-2*n))
		t.place(at, d-2*n)
		t.stack = t.stack[:d-2*n]
		t.emitWrite(at, code.RHash, t.own(d-2*n), t.own(d-2*n), n)
		t.push(inRegister(t.own(d - 2*n)))

	case code.OpAppend:
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		array := t.register(at, t.stack[d-2], t.own(d-2))
		t.emit(at, code.RAppend, array, r, t.u16(at, 0))

	case code.OpHashSet:
		t.place(at, d-2)
		t.pop()
		t.pop()
		hash := t.register(at, t.stack[d-3], t.own(d-3))
		t.emit(at, code.RHashSet, hash, t.own(d-2), 0)

	case code.OpHashMerge:
		source := t.register(at, t.pop(), t.own(d-1))
		hash := t.register(at, t.stack[d-2], t.own(d-2))
		t.emit(at, code.RHashMerge, hash, source, 0)

	case code.OpWithout:
		value := t.pop()
		r := t.register(at, value, t.own(d-1))
		t.keep(at, t.own(d-1))
		t.emitWrite(at, code.RWithout, t.own(d-1), r, t.u16(at, 0))
		t.push(inRegister(t.own(d - 1)))

	case code.OpIndex:
		index, container := t.pop(), t.pop()
		rc := t.register(at, container, t.own(d-2))
		ri := t.register(at, index, t.own(d-1))
		t.keep(at, t.own(d-2))
		t.emitWrite(at, code.RIndex, t.own(d-2), rc, ri)
		t.push(inRegister(t.own(d - 2)))

	case code.OpSetIndex:
		value, index, container := t.pop(), t.pop(), t.pop()
		rc := t.register(at, container, t.own(d-3))
		ri := t.register(at, index, t.own(d-2))
		rv := t.register(at, value, t.own(d-1))
		t.emit(at, code.RSetIndex, rc, ri, rv)
		t.push(inRegister(rv))

	case code.OpGetProperty:
		obj := t.pop()
		r := t.register(at, obj, t.own(d-1))
		t.keep(at, t.own(d-1))
		t.emitWrite(at, code.RGetProp, t.own(d-1), r, t.u16(at, 0))
		t.push(inRegister(t.own(d - 1)))

	case code.OpSetProperty:
		value, obj := t.pop(), t.pop()
		ro := t.register(at, obj, t.own(d-2))
		rv := t.register(at, value, t.own(d-1))
		t.emit(at, code.RSetProp, ro, t.u16(at, 0), rv)
		t.push(inRegister(rv))

	case code.OpCall, code.OpCallMethod:
		argc := t.u16(at, 0)
		name := 0
		rop := code.RCall
		if op == code.OpCallMethod {
			name, argc, rop = t.u16(at, 0), t.u16(at, 1), code.RCallMethod
		}
		fn := d - 1 - argc
		// The call uses the registers from the function's up
		t.keepBelow(at, fn, t.own(fn))
		t.place(at, fn)
		t.stack = t.stack[:fn]
		if load := t.globalLoad(t.own(fn)); rop == code.RCall && load >= 0 {
			rop, name = code.RCallGlobal, t.out[load].B()
			t.remove(load)
		}
		t.emit(at, rop, t.own(fn), argc, name)
		t.push(inRegister(t.own(fn)))

	case code.OpCallArray, code.OpCallMethodArray:
		name := 0
		rop := code.RCallSpread
		if op == code.OpCallMethodArray {
			name, rop = t.u16(at, 0), code.RCallMethodSpread
		}
		t.keepBelow(at, d-2, t.own(d-2))
		t.place(at, d-2)
		t.stack = t.stack[:d-2]
		t.emit(at, rop, t.own(d-2), 0, name)
		t.push(inRegister(t.own(d - 2)))

	case code.OpReturnValue:
		value := t.pop()
		t.emit(at, code.RReturn, t.register(at, value, t.own(d-1)), 0, 0)

	case code.OpClosure:
		n := t.u16(at, 1)
		t.keepBelow(at, d-n, t.own(d-n))
		t.place(at, d-n)
		t.stack = t.stack[:d-n]
		// It reads its cells from A, so it can't write elsewhere
		t.emit(at, code.RClosure, t.own(d-n), t.u16(at, 0), n)
		t.push(inRegister(t.own(d - n)))

	case code.OpLoad:
		t.keep(at, t.own(d))
		t.emitWrite(at, code.RLoad, t.own(d), t.u16(at, 0), t.u16(at, 1))
		t.push(inRegister(t.own(d)))

	case code.OpAwait:
		value := t.pop()
		t.keep(at, t.own(d-1))
		t.load(at, t.own(d-1), value)
		t.emit(at, code.RAwait, t.own(d-1), 0, 0)
		t.push(inRegister(t.own(d - 1)))

	case code.OpTry:
		// The catch block is a place jumps go to, with the error after the
		// values the try started with
		t.place(at, 0)
		t.jump(at, code.RTry, t.own(d), 0, 0, 1, at)

	case code.OpTryEnd:
		t.emit(at, code.RTryEnd, 1, 0, 0)

	case code.OpThrow:
		value := t.pop()
		t.emit(at, code.RThrow, t.register(at, value, t.own(d-1)), 0, 0)

	case code.OpError:
		t.emit(at, code.RError, t.u16(at, 0), 0, 0)

	default:
		t.fail("no translation for "+definitionName(op)+" at", at)
	}
	return next
}

// binary translates an operator, with the form of the instruction that fits
// its operands, and a comparison that decides a jump as one instruction.
func (t *translation) binary(at int, op code.Opcode, next int, d int) int {
	right, left := t.pop(), t.pop()
	p := d - 2

	if forms, ok := jumpOps(op); ok && t.fused(next, code.OpJumpNotTruthy) {
		left, right = t.secure(at, left, t.own(p)), t.secure(at, right, t.own(p+1))
		t.place(at, 0)
		l := t.register(at, left, t.own(p))
		switch n, small := t.smallInt(right); {
		case small:
			t.jump(at, forms[1], l, n, 0, 2, next)
		case right.constant >= 0 && forms[2] != 0:
			t.jump(at, forms[2], l, right.constant, 0, 2, next)
		default:
			r := t.register(at, right, t.own(p+1))
			t.jump(at, forms[0], l, r, 0, 2, next)
		}
		return next + 1
	}

	l := t.register(at, left, t.own(p))
	t.keep(at, t.own(p))
	if n, small := t.smallInt(right); small && (op == code.OpAdd || op == code.OpSub) {
		rop := code.RAddI
		if op == code.OpSub {
			rop = code.RSubI
		}
		t.emitWrite(at, rop, t.own(p), l, n)
	} else if rop, withConstant := binaryOp(op); withConstant != 0 && right.constant >= 0 {
		t.emitWrite(at, withConstant, t.own(p), l, right.constant)
	} else {
		r := t.register(at, right, t.own(p+1))
		t.emitWrite(at, rop, t.own(p), l, r)
	}
	t.push(inRegister(t.own(p)))
	return next
}

// positionAt is object.CompiledFunction.PositionAt on a list of positions.
func positionAt(positions []object.Position, offset int) (int, int) {
	return (&object.CompiledFunction{Positions: positions}).PositionAt(offset)
}

// tempBase is the first register of the values on the stack while a
// function is translated, before its number of locals is known. relocate
// moves them after the locals.
const tempBase = 1 << 15

// relocate moves the registers of the stack values in code from tempBase to
// right after the locals.
func relocate(words []code.Word, numLocals int) {
	for i, w := range words {
		mask := code.Registers(w.Op())
		a, b, c := w.A(), w.B(), w.C()
		if mask&1 != 0 && a >= tempBase {
			a += numLocals - tempBase
		}
		if mask&2 != 0 && b >= tempBase {
			b += numLocals - tempBase
		}
		if mask&4 != 0 && c >= tempBase {
			c += numLocals - tempBase
		}
		words[i] = code.MakeWord(w.Op(), a, b, c)
	}
}
