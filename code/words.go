package code

import (
	"fmt"
	"strings"
)

// Register machine code.
//
// The compiler makes stack code, which compiler/registers.go translates to
// the code the VM runs: instructions that name the registers they read and
// write. A function's registers are its locals, then the places its stack
// code would use for values on the stack.
//
// An instruction is a 64-bit Word: an operation in the low byte, and three
// 16-bit operands A, B and C above it. Operands are register numbers,
// constant indexes, instruction indexes for jumps, counts, or small signed
// integers, by operation.
type Word uint64

// Op is an operation of the register machine.
type Op byte

// MakeWord encodes an instruction.
func MakeWord(op Op, a, b, c int) Word {
	return Word(op) | Word(uint16(a))<<8 | Word(uint16(b))<<24 | Word(uint16(c))<<40
}

func (w Word) Op() Op { return Op(w) }
func (w Word) A() int { return int(uint16(w >> 8)) }
func (w Word) B() int { return int(uint16(w >> 24)) }
func (w Word) C() int { return int(uint16(w >> 40)) }

// SB and SC are the B and C operands as signed numbers, for immediates.
func (w Word) SB() int { return int(int16(w >> 24)) }
func (w Word) SC() int { return int(int16(w >> 40)) }

// The operations. R[x] is register x, K[x] constant x, and I a small
// integer in the instruction itself. Jump targets are instruction indexes.
const (
	RMove      Op = iota // R[A] = R[B]
	RLoadK               // R[A] = K[B]
	RLoadI               // R[A] = the integer sB
	RLoadNull            // R[A] = null
	RLoadTrue            // R[A] = true
	RLoadFalse           // R[A] = false

	RGetGlobal  // R[A] = global B
	RSetGlobal  // global B = R[A]
	RGetBuiltin // R[A] = builtin B
	RGetCell    // R[A] = the value in the cell in R[B]
	RSetCell    // the value in the cell in R[A] = R[B]
	RNewCell    // R[A] = a new cell holding R[B]
	RGetFree    // R[A] = the value of captured variable B
	RSetFree    // captured variable B = R[A]
	RFreeCell   // R[A] = the cell of captured variable B, to capture again
	RLocalCell  // R[A] = the cell of local B, made if it has none yet, to capture it

	// Binary operators: R[A] = R[B] op R[C], with a K form taking K[C] and
	// an I form taking sC
	RAdd
	RSub
	RMul
	RDiv
	RMod
	RPow
	RXor
	REqual
	RNotEqual
	RGreater
	RGreaterEqual
	RLess
	RLessEqual
	RRange
	RIn
	RAddK
	RSubK
	RMulK
	RDivK
	RModK
	RAddI
	RSubI

	RInRange // R[A] = R[A] is in R[A+1]..R[A+2]
	RMinus   // R[A] = -R[B]
	RNot     // R[A] = !R[B]
	RIncr    // R[A] = R[B] + 1, or - 1 when C is 1

	// Jumps
	RJump        // go to A
	RJumpFalsy   // go to B if R[A] is falsy
	RJumpTruthy  // go to B if R[A] is truthy
	RJumpNull    // go to B if R[A] is null
	RJumpNotNull // go to B if R[A] isn't null
	// Comparisons that jump to C when false: R[A] op R[B], or op sB for I,
	// or op K[B] for K
	RJumpLess
	RJumpLessEqual
	RJumpGreater
	RJumpGreaterEqual
	RJumpEqual
	RJumpNotEqual
	RJumpLessI
	RJumpLessEqualI
	RJumpGreaterI
	RJumpGreaterEqualI
	RJumpEqualI
	RJumpNotEqualI
	RJumpEqualK
	RJumpNotEqualK

	// Iterators
	RIterStart    // R[A] = an iterator over R[B]
	RRangeIter    // R[A] = an iterator over R[B]..R[C]
	RIterNext     // R[B] = the next value of iterator R[A]; go to C when done
	RIterNextPair // R[B], R[B+1] = the next key and value of iterator R[A]; go to C when done

	// Values
	RArray     // R[A] = [R[B], ..., R[B+C-1]]
	RHash      // R[A] = {R[B]: R[B+1], ...} with C pairs
	RTemplate  // R[A] = the text of R[B], ..., R[B+C-1] joined
	RAppend    // add R[B] to the array R[A], or its elements when C is 1
	RHashSet   // R[A][R[B]] = R[B+1], in an object literal
	RHashMerge // copy the properties of R[B] into the object R[A]
	RWithout   // R[A] = R[B] without the keys in the array K[C]
	RIndex     // R[A] = R[B][R[C]]
	RSetIndex  // R[A][R[B]] = R[C]
	RGetProp   // R[A] = R[B].K[C]
	RSetProp   // R[A].K[B] = R[C]

	// Calls, with the function in R[A] and the arguments after it; the
	// result goes in R[A]
	RCall             // B arguments
	RCallGlobal       // R[A] = the global C, then a call with B arguments
	RCallMethod       // the method K[C] of R[A], with B arguments
	RCallSpread       // the arguments are the elements of the array R[A+1]
	RCallMethodSpread // the method K[C] of R[A], with the elements of the array R[A+1]
	RReturn           // return R[A]
	RClosure          // R[A] = a closure of K[B] capturing the C cells in R[A]...
	RLoad             // R[A] = the names of the file K[B], shown as K[C]
	RAwait            // R[A] = the result of awaiting R[A]

	// Errors
	RTry    // start a try block whose catch block starts at B with the error in R[A]
	RTryEnd // end the innermost try block, A times
	RThrow  // raise an error with R[A]
	RError  // stop with the error message K[A]

	RIsArray // R[A] = whether R[B] is an array, for patterns
)

var opNames = [...]string{
	RMove: "MOVE", RLoadK: "LOADK", RLoadI: "LOADI", RLoadNull: "LOADNULL", RLoadTrue: "LOADTRUE", RLoadFalse: "LOADFALSE",
	RGetGlobal: "GETGLOBAL", RSetGlobal: "SETGLOBAL", RGetBuiltin: "GETBUILTIN", RGetCell: "GETCELL", RSetCell: "SETCELL",
	RNewCell: "NEWCELL", RGetFree: "GETFREE", RSetFree: "SETFREE", RFreeCell: "FREECELL", RLocalCell: "LOCALCELL",
	RAdd: "ADD", RSub: "SUB", RMul: "MUL", RDiv: "DIV", RMod: "MOD", RPow: "POW", RXor: "XOR",
	REqual: "EQ", RNotEqual: "NE", RGreater: "GT", RGreaterEqual: "GE", RLess: "LT", RLessEqual: "LE",
	RRange: "RANGE", RIn: "IN", RAddK: "ADDK", RSubK: "SUBK", RMulK: "MULK", RDivK: "DIVK", RModK: "MODK",
	RAddI: "ADDI", RSubI: "SUBI", RInRange: "INRANGE", RMinus: "MINUS", RNot: "NOT", RIncr: "INCR",
	RJump: "JMP", RJumpFalsy: "JFALSY", RJumpTruthy: "JTRUTHY", RJumpNull: "JNULL", RJumpNotNull: "JNOTNULL",
	RJumpLess: "JLT", RJumpLessEqual: "JLE", RJumpGreater: "JGT", RJumpGreaterEqual: "JGE", RJumpEqual: "JEQ",
	RJumpNotEqual: "JNE", RJumpLessI: "JLTI", RJumpLessEqualI: "JLEI", RJumpGreaterI: "JGTI",
	RJumpGreaterEqualI: "JGEI", RJumpEqualI: "JEQI", RJumpNotEqualI: "JNEI", RJumpEqualK: "JEQK", RJumpNotEqualK: "JNEK",
	RIterStart: "ITERSTART", RRangeIter: "RANGEITER", RIterNext: "ITERNEXT", RIterNextPair: "ITERNEXT2",
	RArray: "ARRAY", RHash: "HASH", RTemplate: "TEMPLATE", RAppend: "APPEND", RHashSet: "HASHSET",
	RHashMerge: "HASHMERGE", RWithout: "WITHOUT", RIndex: "INDEX", RSetIndex: "SETINDEX", RGetProp: "GETPROP",
	RSetProp: "SETPROP", RCall: "CALL", RCallGlobal: "CALLGLOBAL", RCallMethod: "CALLMETHOD", RCallSpread: "CALLSPREAD",
	RCallMethodSpread: "CALLMETHODSPREAD", RReturn: "RETURN", RClosure: "CLOSURE", RLoad: "LOAD", RAwait: "AWAIT",
	RTry: "TRY", RTryEnd: "TRYEND", RThrow: "THROW", RError: "ERROR", RIsArray: "ISARRAY",
}

func (op Op) String() string {
	if int(op) < len(opNames) && opNames[op] != "" {
		return opNames[op]
	}
	return fmt.Sprintf("OP%d", op)
}

// WordsString prints register code, one instruction a line.
func WordsString(code []Word) string {
	var out strings.Builder
	for i, w := range code {
		fmt.Fprintf(&out, "%04d %-12s %d %d %d\n", i, w.Op(), w.A(), int16(w.B()), int16(w.C()))
	}
	return out.String()
}
