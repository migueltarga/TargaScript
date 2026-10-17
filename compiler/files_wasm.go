//go:build tinygo && wasm

package compiler

import "path"

// The browser build has no files of its own, and leaves out the os package
// with what it pulls in, like the time package.

// ReadFile reads the files programs load; the browser sets it to read them
// from the page's server. It returns ErrNotExist for a file that isn't
// there.
var ReadFile = func(string) ([]byte, error) { return nil, ErrNotExist }

// ErrNotExist is the error of ReadFile for a missing file.
var ErrNotExist error = notExist{}

type notExist struct{}

func (notExist) Error() string { return "file not found" }

// isNotExist tells ErrNotExist by its type, as comparing errors with ==
// makes TinyGo compare them with reflection
func isNotExist(err error) bool {
	_, ok := err.(notExist)
	return ok
}

var (
	isAbs    = path.IsAbs
	joinPath = path.Join
	dirOf    = path.Dir
)

func absPath(file string) (string, error) { return path.Clean(file), nil }
