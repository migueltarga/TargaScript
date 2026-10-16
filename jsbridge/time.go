//go:build tinygo && wasm

package jsbridge

import "github.com/migueltarga/TargaScript/object"

// Time tells the time with JavaScript's Date, which keeps Go's time package
// out of the WebAssembly build and gives the browser's time zone.
type Time struct{}

func (Time) Now() float64 {
	return number(date().CallMethod("now", nil)) / 1000
}

func (Time) Offset(unix int64) int64 {
	d, ok := date().New([]object.Object{&object.Integer{Value: unix * 1000}}).(*Value)
	if !ok {
		return 0
	}
	return -int64(number(d.CallMethod("getTimezoneOffset", nil))) * 60
}

func date() *Value {
	return Global().Get("Date").(*Value)
}

func number(value object.Object) float64 {
	switch value := value.(type) {
	case *object.Float:
		return value.Value
	case *object.Integer:
		return float64(value.Value)
	}
	return 0
}
