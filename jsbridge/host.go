//go:build tinygo && wasm

package jsbridge

import "unsafe"

// Functions provided by targascript.js. JavaScript values are passed as
// handles, indexes into a table kept on the JavaScript side. Handles 0 to 4
// are undefined, null, true, false and globalThis. Operations that can
// throw return the handle of the error message with failed set.

const failed = 1 << 31

//go:wasmimport targascript get
func hostGet(object uint32, name *byte, nameLen uint32) uint32

//go:wasmimport targascript set
func hostSet(object uint32, name *byte, nameLen uint32, value uint32) uint32

//go:wasmimport targascript call
func hostCall(object uint32, name *byte, nameLen uint32, args *uint32, argsLen uint32) uint32

//go:wasmimport targascript invoke
func hostInvoke(fn uint32, args *uint32, argsLen uint32) uint32

//go:wasmimport targascript construct
func hostConstruct(constructor uint32, args *uint32, argsLen uint32) uint32

//go:wasmimport targascript kind
func hostKind(value uint32) uint32

//go:wasmimport targascript number
func hostNumber(value uint32) float64

//go:wasmimport targascript makeNumber
func hostMakeNumber(value float64) uint32

//go:wasmimport targascript stringLength
func hostStringLength(value uint32) uint32

//go:wasmimport targascript stringRead
func hostStringRead(buffer *byte)

//go:wasmimport targascript makeString
func hostMakeString(text *byte, textLen uint32) uint32

// hostBytesLength readies the bytes of a Uint8Array, Uint8ClampedArray or
// ArrayBuffer to be read with hostStringRead, and returns how many there are.
//
//go:wasmimport targascript bytesLength
func hostBytesLength(value uint32) uint32

// hostMakeBytes makes a Uint8Array with a copy of the bytes.
//
//go:wasmimport targascript makeBytes
func hostMakeBytes(data *byte, dataLen uint32) uint32

//go:wasmimport targascript length
func hostLength(value uint32) uint32

//go:wasmimport targascript index
func hostIndex(value uint32, i uint32) uint32

//go:wasmimport targascript makeArray
func hostMakeArray() uint32

//go:wasmimport targascript push
func hostPush(array uint32, value uint32)

//go:wasmimport targascript makeObject
func hostMakeObject() uint32

//go:wasmimport targascript makeFunction
func hostMakeFunction(id uint32) uint32

//go:wasmimport targascript retain
func hostRetain(value uint32)

//go:wasmimport targascript release
func hostRelease(value uint32)

// Kinds returned by hostKind
const (
	kindUndefined = iota
	kindNull
	kindBoolean
	kindNumber
	kindString
	kindArray
	kindObject
	kindFunction
	kindBytes // a Uint8Array, Uint8ClampedArray or ArrayBuffer
)

const (
	handleUndefined = 0
	handleNull      = 1
	handleTrue      = 2
	handleFalse     = 3
	handleGlobal    = 4
)

func stringPtr(s string) (*byte, uint32) {
	if len(s) == 0 {
		return nil, 0
	}
	return unsafe.StringData(s), uint32(len(s))
}

func argsPtr(args []uint32) (*uint32, uint32) {
	if len(args) == 0 {
		return nil, 0
	}
	return &args[0], uint32(len(args))
}

// readString reads a JavaScript value as a string.
func readString(handle uint32) string {
	n := hostStringLength(handle)
	if n == 0 {
		return ""
	}
	buffer := make([]byte, n)
	hostStringRead(&buffer[0])
	return string(buffer)
}
