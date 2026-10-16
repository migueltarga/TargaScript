package compiler

import (
	"errors"
	"strconv"

	"github.com/migueltarga/TargaScript/code"
)

// stackDepths returns, for each instruction of a function, how many values
// are on its stack above the locals when it runs, or -1 for the bytes that
// don't start an instruction that runs. It follows every way the code can
// go and fails when two reach an instruction with different depths, which
// would be a bug of the compiler.
//
// The depths let instructions name the slot they read and write instead of
// taking them from the top of the stack.
func stackDepths(ins code.Instructions) ([]int, error) {
	flow, err := analyze(ins)
	if err != nil {
		return nil, err
	}
	return flow.depths, nil
}

// flow is what analyze works out about each instruction of a function.
type flow struct {
	depths []int         // values on the stack, or -1 when nothing reaches it
	tries  []int         // try blocks open
	loops  [][]loopDepth // loops open, innermost last
}

// analyze follows every way the code of a function can go, as stackDepths
// does, and also counts the try blocks open at each instruction.
func analyze(ins code.Instructions) (*flow, error) {
	depths := make([]int, len(ins))
	for i := range depths {
		depths[i] = -1
	}
	tries := make([]int, len(ins))
	// loops holds, for each instruction, the depths at which the loops open
	// there started, innermost last, for break and continue
	loops := make([][]loopDepth, len(ins))

	type state struct {
		at    int
		depth int
		tries int
		loops []loopDepth
	}
	work := []state{{0, 0, 0, nil}}
	open, openTries := []loopDepth(nil), 0
	visit := func(at, depth int, open []loopDepth) error {
		if at >= len(ins) {
			return flowError("jump past the end at", at)
		}
		if depth < 0 {
			return flowError("stack underflow at", at)
		}
		if depths[at] >= 0 {
			if depths[at] != depth {
				return flowError("different stack depths at", at)
			}
			if tries[at] != openTries {
				return flowError("different try blocks open at", at)
			}
			return nil
		}
		depths[at], loops[at], tries[at] = depth, open, openTries
		work = append(work, state{at, depth, openTries, open})
		return nil
	}
	if len(ins) == 0 {
		return &flow{depths, tries, loops}, nil
	}
	depths[0] = 0

	for len(work) > 0 {
		s := work[len(work)-1]
		work = work[:len(work)-1]
		var i, d int
		i, d, open, openTries = s.at, s.depth, s.loops, s.tries
		op := code.Opcode(ins[i])
		next := i + instructionWidth(ins, i)
		u16 := func(offset int) int { return int(code.ReadUint16(ins[i+offset:])) }

		var err error
		follow := func(at, depth int) {
			if err == nil {
				err = visit(at, depth, open)
			}
		}
		switch op {
		case code.OpReturnValue, code.OpThrow, code.OpError:
			// The function ends

		case code.OpJump:
			follow(u16(1), d)
		case code.OpJumpNotTruthy:
			follow(u16(1), d-1)
			follow(next, d-1)
		case code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep, code.OpJumpNotNullKeep:
			follow(u16(1), d)
			follow(next, d-1)
		case code.OpJumpNullKeep:
			follow(u16(1), d)
			follow(next, d)

		case code.OpLoopEnter:
			loop := loopDepth{depth: d, tries: openTries, breakTo: u16(1), continueTo: u16(3)}
			open = append(open[:len(open):len(open)], loop)
			follow(next, d)
		case code.OpLoopExit:
			if len(open) == 0 {
				return nil, flowError("loop exit outside a loop at", i)
			}
			open = open[:len(open)-1]
			follow(next, d)
		case code.OpBreak:
			if len(open) == 0 {
				return nil, flowError("break outside a loop at", i)
			}
			loop := open[len(open)-1]
			// break closes the try blocks opened in the loop
			openTries = loop.tries
			if ins[i+1] == 1 {
				follow(loop.continueTo, loop.depth)
			} else {
				follow(loop.breakTo, loop.depth)
			}
		case code.OpIterNext:
			follow(u16(1), d)
			if ins[i+3] == 1 {
				follow(next, d+2)
			} else {
				follow(next, d+1)
			}
		case code.OpTry:
			// The catch block starts with the error on the stack, and its
			// try block closed
			follow(u16(1), d+1)
			openTries++
			follow(next, d)
		case code.OpTryEnd:
			if openTries == 0 {
				return nil, flowError("try end outside a try at", i)
			}
			openTries--
			follow(next, d)

		default:
			effect, ok := stackEffect(op, ins[i+1:])
			if !ok {
				return nil, flowError("no stack effect for "+definitionName(op)+" at", i)
			}
			follow(next, d+effect)
		}
		if err != nil {
			return nil, err
		}
	}
	return &flow{depths, tries, loops}, nil
}

// loopDepth is a loop open at an instruction: the depth it started at and
// where break and continue go.
type loopDepth struct {
	depth               int
	tries               int // try blocks open when the loop started
	breakTo, continueTo int
}

// stackEffect returns how many values an instruction that always goes on to
// the next one adds to the stack, or removes when negative.
func stackEffect(op code.Opcode, operands []byte) (int, bool) {
	u16 := func(offset int) int { return int(code.ReadUint16(operands[offset:])) }
	switch op {
	case code.OpConstant, code.OpTrue, code.OpFalse, code.OpNull, code.OpDup,
		code.OpGetGlobal, code.OpGetLocal, code.OpGetLocalCell, code.OpGetFree,
		code.OpGetFreeCell, code.OpGetBuiltin, code.OpLoad:
		return 1, true
	case code.OpDup2:
		return 2, true
	case code.OpWithout, code.OpMinus, code.OpBang, code.OpIncrement,
		code.OpSetGlobal, code.OpSetLocal, code.OpNewCell, code.OpSetFree,
		code.OpGetProperty, code.OpAwait, code.OpTryEnd, code.OpIterStart:
		return 0, true
	case code.OpPop, code.OpAppend, code.OpHashMerge, code.OpIndex, code.OpSetProperty,
		code.OpCallArray, code.OpCallMethodArray, code.OpRangeIter,
		code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow, code.OpXor,
		code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterEqual,
		code.OpLessThan, code.OpLessEqual, code.OpRange, code.OpIn:
		return -1, true
	case code.OpHashSet, code.OpInRange, code.OpSetIndex:
		return -2, true
	case code.OpArray, code.OpTemplate:
		return 1 - u16(0), true
	case code.OpHash:
		return 1 - 2*u16(0), true
	case code.OpSlide:
		return -int(operands[0]), true
	case code.OpCall:
		return -int(operands[0]), true
	case code.OpCallMethod:
		return -int(operands[2]), true
	case code.OpClosure:
		return 1 - int(operands[2]), true
	}
	return 0, false
}

func definitionName(op code.Opcode) string {
	if def, err := code.Lookup(byte(op)); err == nil {
		return def.Name
	}
	return "opcode " + strconv.Itoa(int(op))
}

// functionName is how errors about a function name it.
func functionName(name string) string {
	if name == "" {
		return "the main program"
	}
	return name
}

// jumpTargets returns the offsets that jumps, loops and try blocks go to.
func jumpTargets(ins code.Instructions) map[int]bool {
	targets := map[int]bool{}
	for i := 0; i < len(ins); i += instructionWidth(ins, i) {
		switch code.Opcode(ins[i]) {
		case code.OpJump, code.OpJumpNotTruthy, code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep,
			code.OpJumpNotNullKeep, code.OpJumpNullKeep, code.OpIterNext, code.OpTry:
			targets[int(code.ReadUint16(ins[i+1:]))] = true
		case code.OpLoopEnter:
			targets[int(code.ReadUint16(ins[i+1:]))] = true
			targets[int(code.ReadUint16(ins[i+3:]))] = true
		}
	}
	return targets
}

func instructionWidth(ins code.Instructions, i int) int {
	def, err := code.Lookup(ins[i])
	if err != nil {
		return 1
	}
	width := 1
	for _, w := range def.OperandWidths {
		width += w
	}
	return width
}

// flowError is an error of analyze, about the instruction at an offset: a
// bug of the compiler. It formats without fmt, which is large in the
// WebAssembly build.
func flowError(message string, at int) error {
	return errors.New(message + " " + strconv.Itoa(at))
}
