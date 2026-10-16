package vm

import (
	"github.com/migueltarga/TargaScript/compiler"
	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/order"
)

// modules are the files loaded by a program and the files it loads, each
// run once.
type modules struct {
	loaded  map[string]*object.Hash
	loading map[string]bool
}

func newModules() *modules {
	return &modules{loaded: map[string]*object.Hash{}, loading: map[string]bool{}}
}

// load runs the file at path, once, and returns an object of its top-level
// names. shown is the path as written in the code, for errors.
func (vm *VM) load(path, shown string, line, column int) object.Object {
	if module, ok := vm.modules.loaded[path]; ok {
		return module
	}
	if vm.modules.loading[path] {
		return evaluator.NewError(line, column, "circular load of %q", shown)
	}

	program, message := compiler.ReadModule(path, shown)
	if message != "" {
		return evaluator.NewError(line, column, "%s", message)
	}

	c := compiler.New()
	c.Dir = compiler.ModuleDir(path)
	bytecode, err := c.Compile(program)
	if err != nil {
		return evaluator.NewError(line, column, "cannot load %q: %s", shown, err)
	}

	module := New(c.Globals())
	module.modules = vm.modules
	if vm.host != nil {
		module.SetHost(vm.host)
	}

	vm.modules.loading[path] = true
	result := module.Run(bytecode)
	delete(vm.modules.loading, path)

	if errObj, ok := result.(*object.Error); ok {
		return evaluator.NewError(line, column, "%s:%d:%d: %s", shown, errObj.Line, errObj.Column, errObj.Message)
	}

	names := append([]string{}, c.Globals().Names()...)
	order.Strings(names)
	exports := object.NewHash()
	for _, name := range names {
		if value := module.global(name); value != nil {
			exports.Set(&object.String{Value: name}, value)
		}
	}

	vm.modules.loaded[path] = exports
	return exports
}

func (vm *VM) builtin(name string) (*object.Builtin, bool) {
	for i, n := range evaluator.BuiltinNames() {
		if n == name {
			return vm.builtins[i], true
		}
	}
	return nil, false
}

// Global returns the value of a global by name, or nil if it isn't set.
func (vm *VM) Global(name string) object.Object {
	return vm.global(name)
}

// global returns the value of a global by name, or nil if it isn't set.
func (vm *VM) global(name string) object.Object {
	for i, n := range vm.names.Names() {
		if n == name && i < len(vm.globals) {
			return vm.globals[i]
		}
	}
	return nil
}
