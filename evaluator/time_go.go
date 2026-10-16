//go:build !tinygo

package evaluator

import "time"

// goTime tells the time with Go's time package, outside the browser build.
type goTime struct{}

func (goTime) Now() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

func (goTime) Offset(unix int64) int64 {
	_, offset := time.Unix(unix, 0).Zone()
	return int64(offset)
}

func init() {
	SetTimeHost(goTime{})
}
