// Package evaluator implements what TargaScript values do: operators,
// indexing, properties, methods, builtins and printing. The bytecode VM in
// package vm calls it for every operation, so the rules live in one place.
//
// It was a tree-walking interpreter before the VM replaced it, hence the
// name.
package evaluator
