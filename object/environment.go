package object

type Environment struct {
	store     map[string]Object
	outer     *Environment
	constants map[string]bool // Tracks which variables are constants
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

	// Check if the variable exists in an outer environment
	if e.outer != nil {
		if _, exists := e.outer.Get(name); exists {
			// Update in outer environment
			return e.outer.Set(name, val)
		}
	}

	// Not found anywhere, create in current environment
	e.store[name] = val
	return val, true
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.constants[name] = true
	return val
}

// GetAll returns all variables in the current environment
func (e *Environment) GetAll() map[string]Object {
	return e.store
}

func (e *Environment) GetAllWithOuter() map[string]Object {
	result := make(map[string]Object)

	current := e
	for current.outer != nil {
		current = current.outer
	}

	for current != nil {
		for k, v := range current.store {
			result[k] = v
		}
		current = findNextInnerEnv(current, e)
	}

	return result
}

func findNextInnerEnv(current, target *Environment) *Environment {
	if current == target {
		return nil
	}

	var directChildren []*Environment
	temp := target
	for temp != nil {
		if temp.outer == current {
			directChildren = append(directChildren, temp)
		}
		temp = temp.outer
	}

	if len(directChildren) == 0 {
		return nil
	}

	for _, child := range directChildren {
		path := child
		for path != nil {
			if path == target {
				return child
			}
			path = findNextInnerEnv(path, target)
		}
	}

	return nil
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
