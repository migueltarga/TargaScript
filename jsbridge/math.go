//go:build tinygo && wasm

package jsbridge

import "github.com/migueltarga/TargaScript/object"

// Math computes sin, log and the like with JavaScript's Math, which keeps
// Go's versions out of the WebAssembly build.
type Math struct{}

func (Math) Math(name string, args ...float64) float64 {
	values := make([]object.Object, len(args))
	for i, arg := range args {
		values[i] = &object.Float{Value: arg}
	}
	switch result := Global().Get("Math").(*Value).CallMethod(name, values).(type) {
	case *object.Float:
		return result.Value
	case *object.Integer:
		return float64(result.Value)
	}
	return 0
}
