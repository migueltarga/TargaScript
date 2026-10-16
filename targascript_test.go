package targascript

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRunWithGoValues(t *testing.T) {
	script := New()
	script.Set("price", 120.0)
	script.Set("tags", []string{"a", "b"})
	script.Set("user", map[string]any{"name": "Ana", "admin": true})
	script.Set("discount", func(p float64) float64 { return p * 0.9 })
	script.Set("lookup", func(id int) (string, error) {
		if id == 0 {
			return "", errors.New("no such id")
		}
		return "item", nil
	})

	result, err := script.Run(`
let total = discount(price) |> (p => round(p, 2))
[total, tags.length, user.name, user.admin, lookup(7)]
`)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{108.0, int64(2), "Ana", true, "item"}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %#v, want %#v", result, want)
	}

	if _, err := script.Run("lookup(0)"); err == nil || !strings.Contains(err.Error(), "no such id") {
		t.Errorf("a Go error should fail the script: %v", err)
	}
}

func TestStateCallsAndGet(t *testing.T) {
	script := New()
	if _, err := script.Run(`let count = 0
fn add(n) { count += n; count }
fn greet(name = "you") { "Hello, ${name}!" }`); err != nil {
		t.Fatal(err)
	}
	script.Call("add", 5)
	total, _ := script.Call("add", 2)
	if total != int64(7) || script.Get("count") != int64(7) {
		t.Errorf("add: %v, count %v", total, script.Get("count"))
	}
	if greeting, _ := script.Call("greet"); greeting != "Hello, you!" {
		t.Errorf("greet() = %v", greeting)
	}
	if _, err := script.Call("missing"); err == nil {
		t.Error("calling a missing function should fail")
	}

	// A TargaScript function comes to Go as a Go function
	result, _ := script.Run("x => x * 2")
	double := result.(func(args ...any) (any, error))
	if got, _ := double(21); got != int64(42) {
		t.Errorf("double(21) = %v", got)
	}
}

func TestErrorsOutputAndLimits(t *testing.T) {
	script := New()
	var out bytes.Buffer
	script.SetOutput(&out)
	script.Run(`print("hi", [1, 2])`)
	if out.String() != "hi [1, 2]\n" {
		t.Errorf("printed %q", out.String())
	}

	_, err := script.Run("let a = 1\nnull.x")
	var scriptErr *Error
	if !errors.As(err, &scriptErr) || scriptErr.Line != 2 || !strings.Contains(err.Error(), " 2 | null.x") {
		t.Errorf("error: %#v", err)
	}
	if _, err := script.Run("let = 1"); err == nil {
		t.Error("a syntax error should fail")
	}

	script.SetMaxSteps(1000)
	if _, err := script.Run("repeat true {}"); err == nil || !strings.Contains(err.Error(), "more than 1000 steps") {
		t.Errorf("an endless loop: %v", err)
	}

	// Scripts can't touch files
	if _, err := script.Run(`readFile("/etc/passwd")`); err == nil {
		t.Error("readFile should be unavailable")
	}
}

func TestAwaitAndCycles(t *testing.T) {
	script := New()
	result, err := script.Run("await sleep(5)\n\"slept\"")
	if err != nil || result != "slept" {
		t.Errorf("got %v, %v", result, err)
	}
	result, _ = script.Run("let a = [1]\na.insert(a)\na")
	if !reflect.DeepEqual(result, []any{int64(1), nil}) {
		t.Errorf("a value inside itself: %#v", result)
	}
}
