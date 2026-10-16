//go:build tinygo && wasm

package jsbridge

import "unsafe"

//go:wasmimport targascript math
func hostMath(name *byte, nameLen uint32, a, b float64) float64

// Math computes sin, log and the like with JavaScript's Math, which keeps
// Go's versions out of the WebAssembly build. It is one call to JavaScript
// with the numbers as they are, since programs call these in tight loops.
type Math struct{}

func (Math) Math(name string, args ...float64) float64 {
	var b float64
	if len(args) > 1 {
		b = args[1]
	}
	return hostMath(unsafe.StringData(name), uint32(len(name)), args[0], b)
}
