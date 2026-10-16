package evaluator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

// fakeExternal is an external value backed by a map. Calling it returns
// its arguments joined, a method call records the method and calls the
// property with that name if it holds a function, and New creates another
// fakeExternal.
type fakeExternal struct {
	props map[string]object.Object
	calls []string
}

func newFake() *fakeExternal {
	return &fakeExternal{props: map[string]object.Object{}}
}

func (f *fakeExternal) Type() object.ObjectType { return object.EXTERNAL_OBJ }
func (f *fakeExternal) Inspect() string         { return "<fake>" }

func (f *fakeExternal) Get(name string) object.Object {
	if value, ok := f.props[name]; ok {
		return value
	}
	if name == "broken" {
		return &object.Error{Message: "cannot read broken"}
	}
	return NULL
}

func (f *fakeExternal) Set(name string, value object.Object) object.Object {
	if name == "readonly" {
		return &object.Error{Message: "readonly is read-only"}
	}
	f.props[name] = value
	return value
}

func (f *fakeExternal) Call(args []object.Object) object.Object {
	parts := []string{}
	for _, arg := range args {
		parts = append(parts, arg.Inspect())
	}
	return &object.String{Value: "called(" + strings.Join(parts, ", ") + ")"}
}

func (f *fakeExternal) CallMethod(name string, args []object.Object) object.Object {
	f.calls = append(f.calls, name)
	if fn, ok := f.props[name]; ok && isCallable(fn) {
		return CallFunction(fn, args)
	}
	return &object.Error{Message: fmt.Sprintf("%s is not a function", name)}
}

func (f *fakeExternal) New(args []object.Object) object.Object {
	created := newFake()
	created.props["args"] = &object.Array{Elements: args}
	return created
}

func evalWithExternal(input string, ext *fakeExternal) object.Object {
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &object.Error{Message: "parser errors: " + strings.Join(p.Errors(), "; ")}
	}
	env := object.NewEnvironment()
	env.Set("ext", ext)
	return Eval(program, env)
}

func TestExternalValues(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`ext.title = "hi"
ext.title`, "hi"},
		{`ext["size"] = 3
ext.size + 1`, "4"},
		{`ext[0] = "zero"
ext["0"]`, "zero"},
		{`ext.missing`, "null"},
		{`ext(1, "a")`, "called(1, a)"},
		{`ext.new(1, 2).args`, "[1, 2]"},
		{`ext.inner = ext.new()
ext.inner.name = "nested"
ext.inner.name`, "nested"},
		{`ext.count = 0
ext.count++
ext.count++
ext.count`, "2"},
		{`ext.double = fn(x) { x * 2 }
ext.double(21)`, "42"},
		{`"value: ${ext}"`, "value: <fake>"},
		{`type(ext)`, "EXTERNAL"},
	}

	for _, tt := range tests {
		evaluated := evalWithExternal(tt.input, newFake())
		got := evaluated.Inspect()
		if s, ok := evaluated.(*object.String); ok {
			got = s.Value
		}
		if got != tt.expected {
			t.Errorf("%q: got=%s, want=%s", tt.input, got, tt.expected)
		}
	}
}

func TestExternalCallbacks(t *testing.T) {
	ext := newFake()
	evaluated := evalWithExternal(`
let clicks = 0
ext.onclick = fn(n) { clicks = clicks + n
clicks }
ext.onclick(2)
ext.onclick(3)
clicks`, ext)
	testIntegerObject(t, evaluated, 5)

	if strings.Join(ext.calls, ",") != "onclick,onclick" {
		t.Errorf("wrong method calls: %v", ext.calls)
	}

	// Code outside the evaluator calls the stored function later
	result := CallFunction(ext.props["onclick"], []object.Object{&object.Integer{Value: 10}})
	testIntegerObject(t, result, 15)
}

func TestExternalErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`ext.broken`, "cannot read broken"},
		{`ext.readonly = 1`, "readonly is read-only"},
		{`ext.nothing()`, "nothing is not a function"},
		{`ext[true]`, "index must be a string or an integer, got: BOOLEAN"},
		{`ext[1.5] = 1`, "index must be a string or an integer, got: FLOAT"},
	}

	for _, tt := range tests {
		errObj, ok := evalWithExternal(tt.input, newFake()).(*object.Error)
		if !ok {
			t.Errorf("%q: expected an error", tt.input)
			continue
		}
		if errObj.Message != tt.expected {
			t.Errorf("%q: got=%q, want=%q", tt.input, errObj.Message, tt.expected)
		}
		if errObj.Line != 1 {
			t.Errorf("%q: error has no position: line %d", tt.input, errObj.Line)
		}
	}
}
