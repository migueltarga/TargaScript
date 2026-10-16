package compiler

import (
	"fmt"

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
	depths := make([]int, len(ins))
	for i := range depths {
		depths[i] = -1
	}
	// loops holds, for each instruction, the depths at which the loops open
	// there started, innermost last, for break and continue
	loops := make([][]loopDepth, len(ins))

	type state struct {
		at    int
		depth int
		loops []loopDepth
	}
	work := []state{{0, 0, nil}}
	visit := func(at, depth int, open []loopDepth) error {
		if at >= len(ins) {
			return fmt.Errorf("jump to %d, past the end", at)
		}
		if depth < 0 {
			return fmt.Errorf("stack underflow at %d", at)
		}
		if depths[at] >= 0 {
			if depths[at] != depth {
				return fmt.Errorf("instruction %d reached with %d and %d values", at, depths[at], depth)
			}
			return nil
		}
		depths[at], loops[at] = depth, open
		work = append(work, state{at, depth, open})
		return nil
	}
	if len(ins) == 0 {
		return depths, nil
	}
	depths[0] = 0

	for len(work) > 0 {
		s := work[len(work)-1]
		work = work[:len(work)-1]
		i, d, open := s.at, s.depth, s.loops
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
		case code.OpCompareJump:
			follow(u16(2), d-2)
			follow(next, d-2)
		case code.OpCompareConstJump:
			follow(u16(4), d-1)
			follow(next, d-1)
		case code.OpLocalCompareConstJump:
			follow(u16(6), d)
			follow(next, d)

		case code.OpLoopEnter:
			loop := loopDepth{depth: d, breakTo: u16(1), continueTo: u16(3)}
			open = append(open[:len(open):len(open)], loop)
			follow(next, d)
		case code.OpLoopExit:
			if len(open) == 0 {
				return nil, fmt.Errorf("loop exit outside a loop at %d", i)
			}
			open = open[:len(open)-1]
			follow(next, d)
		case code.OpBreak:
			if len(open) == 0 {
				return nil, fmt.Errorf("break outside a loop at %d", i)
			}
			loop := open[len(open)-1]
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
			// The catch block starts with the error on the stack
			follow(u16(1), d+1)
			follow(next, d)

		default:
			effect, ok := stackEffect(op, ins[i+1:])
			if !ok {
				return nil, fmt.Errorf("no stack effect for %s at %d", definitionName(op), i)
			}
			follow(next, d+effect)
		}
		if err != nil {
			return nil, err
		}
	}
	return depths, nil
}

// loopDepth is a loop open at an instruction: the depth it started at and
// where break and continue go.
type loopDepth struct {
	depth               int
	breakTo, continueTo int
}

// stackEffect returns how many values an instruction that always goes on to
// the next one adds to the stack, or removes when negative.
func stackEffect(op code.Opcode, operands []byte) (int, bool) {
	u16 := func(offset int) int { return int(code.ReadUint16(operands[offset:])) }
	switch op {
	case code.OpConstant, code.OpTrue, code.OpFalse, code.OpNull, code.OpDup,
		code.OpGetGlobal, code.OpGetLocal, code.OpGetLocalCell, code.OpGetFree,
		code.OpGetFreeCell, code.OpGetBuiltin, code.OpLoad,
		code.OpLocalBinaryConst, code.OpLocalLocalBinary:
		return 1, true
	case code.OpDup2:
		return 2, true
	case code.OpWithout, code.OpMinus, code.OpBang, code.OpIncrement,
		code.OpSetGlobal, code.OpSetLocal, code.OpNewCell, code.OpSetFree,
		code.OpGetProperty, code.OpAwait, code.OpTryEnd, code.OpIterStart,
		code.OpBinaryConst:
		return 0, true
	case code.OpPop, code.OpAppend, code.OpHashMerge, code.OpIndex, code.OpSetProperty,
		code.OpCallArray, code.OpCallMethodArray, code.OpRangeIter,
		code.OpSetLocalPop, code.OpSetGlobalPop,
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
	return fmt.Sprintf("opcode %d", op)
}

// functionName is how errors about a function name it.
func functionName(name string) string {
	if name == "" {
		return "the main program"
	}
	return name
}
