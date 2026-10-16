package compiler

import "github.com/migueltarga/TargaScript/code"

// instruction is an instruction of the stack code the compiler makes before
// translating it to register code: its opcode and its operands.
type instruction struct {
	op   code.Opcode
	args [3]int
}

// arg returns operand n of the instruction at.
func (ins instructions) arg(at, n int) int { return ins[at].args[n] }

// instructions is the stack code of a function.
type instructions []instruction
