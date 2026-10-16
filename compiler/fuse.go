package compiler

import "github.com/migueltarga/TargaScript/code"

// fuse replaces common pairs of instructions with a superinstruction that
// does both in one step of the VM, like a comparison followed by a jump.
// A superinstruction takes the same bytes as the pair it replaces, so jumps
// and positions stay where they are. A pair isn't fused when a jump goes to
// its second instruction, which would then be gone.
func fuse(ins code.Instructions) {
	targets := jumpTargets(ins)
	fusePairs(ins, targets)
	fuseLocals(ins, targets)
}

// fuseLocals fuses reading a local with the instruction that uses it, after
// fusePairs: a local and a constant, as in n - 1; a local compared with a
// constant, as in if n < 2; and two locals, as in a + b.
func fuseLocals(ins code.Instructions, targets map[int]bool) {
	for i := 0; i < len(ins); {
		next := i + instructionWidth(ins, i)
		if code.Opcode(ins[i]) == code.OpGetLocal && next < len(ins) && !targets[next] {
			local := int(code.ReadUint16(ins[i+1:]))
			switch code.Opcode(ins[next]) {
			case code.OpBinaryConst:
				op, constant := int(ins[next+1]), int(code.ReadUint16(ins[next+2:]))
				copy(ins[i:], code.Make(code.OpLocalBinaryConst, local, op, constant, 0))
				i += 7
				continue
			case code.OpCompareConstJump:
				op, constant, target := int(ins[next+1]), int(code.ReadUint16(ins[next+2:])), int(code.ReadUint16(ins[next+4:]))
				copy(ins[i:], code.Make(code.OpLocalCompareConstJump, local, op, constant, target, 0))
				i += 10
				continue
			case code.OpGetLocal:
				third := next + 3
				if third < len(ins) && !targets[third] && isBinary(code.Opcode(ins[third])) {
					other := int(code.ReadUint16(ins[next+1:]))
					copy(ins[i:], code.Make(code.OpLocalLocalBinary, local, other, int(ins[third]), 0))
					i += 7
					continue
				}
			}
		}
		i = next
	}
}

// fusePairs fuses the pairs of instructions that don't read locals.
func fusePairs(ins code.Instructions, targets map[int]bool) {
	for i := 0; i < len(ins); {
		op := code.Opcode(ins[i])
		width := instructionWidth(ins, i)
		next := i + width
		if next < len(ins) && !targets[next] {
			second := code.Opcode(ins[next])
			// A constant, a comparison and a jump, as in if n < 2
			if third := next + 1; op == code.OpConstant && isComparison(second) && third < len(ins) && !targets[third] &&
				code.Opcode(ins[third]) == code.OpJumpNotTruthy {
				constant := code.ReadUint16(ins[i+1:])
				target := code.ReadUint16(ins[third+1:])
				copy(ins[i:], code.Make(code.OpCompareConstJump, int(second), int(constant), int(target), 0))
				i += 7
				continue
			}
			switch {
			case op == code.OpConstant && isBinary(second):
				constant := code.ReadUint16(ins[i+1:])
				copy(ins[i:], code.Make(code.OpBinaryConst, int(second), int(constant)))
				i += 4
				continue
			case isComparison(op) && second == code.OpJumpNotTruthy:
				target := code.ReadUint16(ins[next+1:])
				copy(ins[i:], code.Make(code.OpCompareJump, int(op), int(target)))
				i += 4
				continue
			case (op == code.OpSetLocal || op == code.OpSetGlobal) && second == code.OpPop:
				fused := code.OpSetLocalPop
				if op == code.OpSetGlobal {
					fused = code.OpSetGlobalPop
				}
				copy(ins[i:], code.Make(fused, int(code.ReadUint16(ins[i+1:])), 0))
				i += 4
				continue
			}
		}
		i = next
	}
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

// isBinary reports whether an operator takes its two operands from the
// stack and pushes one result, so it can take its right one from a constant.
func isBinary(op code.Opcode) bool {
	switch op {
	case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow, code.OpXor, code.OpIn:
		return true
	}
	return isComparison(op)
}

func isComparison(op code.Opcode) bool {
	switch op {
	case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterEqual, code.OpLessThan, code.OpLessEqual:
		return true
	}
	return false
}
