package compiler

import (
	"errors"
	"strconv"

	"github.com/migueltarga/TargaScript/code"
)

// stackEffect returns how many values an instruction that always goes on to
// the next one adds to the stack, or removes when negative.
func stackEffect(op code.Opcode, args [3]int) (int, bool) {
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
		return 1 - args[0], true
	case code.OpHash:
		return 1 - 2*args[0], true
	case code.OpSlide:
		return -args[0], true
	case code.OpCall:
		return -args[0], true
	case code.OpCallMethod:
		return -args[1], true
	case code.OpClosure:
		return 1 - args[1], true
	}
	return 0, false
}

func definitionName(op code.Opcode) string {
	return "opcode " + strconv.Itoa(int(op))
}

// functionName is how errors about a function name it.
func functionName(name string) string {
	if name == "" {
		return "the main program"
	}
	return name
}

// flowError is an error of analyze, about the instruction at an offset: a
// bug of the compiler. It formats without fmt, which is large in the
// WebAssembly build.
func flowError(message string, at int) error {
	return errors.New(message + " " + strconv.Itoa(at))
}

// jumpList returns the jumps of stack code, as their offset and their
// target's.
func jumpList(ins instructions) [][2]int {
	jumps := [][2]int{}
	for i := range ins {
		switch ins[i].op {
		case code.OpJump, code.OpJumpNotTruthy, code.OpJumpNotTruthyKeep, code.OpJumpTruthyKeep,
			code.OpJumpNotNullKeep, code.OpJumpNullKeep, code.OpIterNext, code.OpTry:
			jumps = append(jumps, [2]int{i, ins[i].args[0]})
		case code.OpLoopEnter:
			jumps = append(jumps, [2]int{i, ins[i].args[0]}, [2]int{i, ins[i].args[1]})
		}
	}
	return jumps
}
