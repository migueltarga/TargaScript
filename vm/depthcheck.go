//go:build !vmcheck

package vm

// checkDepths is set by the vmcheck tag, for testing the depths the
// compiler works out: go test -tags vmcheck ./...
const checkDepths = false

func (vm *VM) checkDepth(*frame, int) {}
