package object

import "sort"

type Environment struct {
	store     map[string]Object
	outer     *Environment
	constants map[string]bool // Tracks which variables are constants

	// Dir is the directory of the file this environment's code comes from.
	// It is set on a file's top-level environment and used to resolve
	// `load` paths.
	Dir string
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	c := make(map[string]bool)
	return &Environment{store: s, constants: c, outer: nil}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) (Object, bool) {
	// Check if this is a constant that's already defined in this environment
	if isConst, exists := e.constants[name]; exists && isConst {
		return &Error{
			Message: "Cannot reassign to constant '" + name + "'",
		}, false
	}

	// If variable exists in current environment, update it
	if _, exists := e.store[name]; exists {
		e.store[name] = val
		return val, true
	}

	if e.outer == nil {
		e.store[name] = val
		return val, true
	}

	if _, ok := e.outer.Get(name); ok {
		return e.outer.Set(name, val)
	}

	e.store[name] = val
	return val, true
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.constants[name] = true
	return val
}

// ModuleDir returns the directory of the file the code running in this
// environment was loaded from, or "" if it's unknown.
func (e *Environment) ModuleDir() string {
	for env := e; env != nil; env = env.outer {
		if env.Dir != "" {
			return env.Dir
		}
	}
	return ""
}

// Names returns every name visible from this environment, including those
// of the enclosing environments.
func (e *Environment) Names() []string {
	names := []string{}
	for env := e; env != nil; env = env.outer {
		for name := range env.store {
			names = append(names, name)
		}
	}
	return names
}

// LocalNames returns the names defined directly in this environment, sorted.
func (e *Environment) LocalNames() []string {
	names := make([]string, 0, len(e.store))
	for name := range e.store {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (e *Environment) HasLocal(name string) bool {
	_, ok := e.store[name]
	return ok
}

func (e *Environment) IsConstant(name string) bool {
	isConst, exists := e.constants[name]
	if !exists && e.outer != nil {
		return e.outer.IsConstant(name)
	}
	return exists && isConst
}

func (e *Environment) IsLocalConstant(name string) bool {
	return e.constants[name]
}

func (e *Environment) SetLocal(name string, val Object) Object {
	if isConst, exists := e.constants[name]; exists && isConst {
		return &Error{
			Message: "Cannot reassign to constant '" + name + "'",
		}
	}

	e.store[name] = val
	return val
}
