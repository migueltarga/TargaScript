package targascript

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/migueltarga/TargaScript/evaluator"
)

type exposedSystem struct{}

func (exposedSystem) ReadFile(string) (string, error)      { return "secret", nil }
func (exposedSystem) WriteFile(string, string, bool) error { return nil }
func (exposedSystem) ListDir(string) ([]string, error)     { return nil, nil }
func (exposedSystem) Env(string) (string, bool)            { return "secret", true }
func (exposedSystem) Exit(int)                             { panic("embedded script reached exit") }

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
	if _, err := script.Run(`load "/etc/passwd" as private`); err == nil || !strings.Contains(err.Error(), "load is not available") {
		t.Errorf("load should be unavailable: %v", err)
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

func TestSeparateScriptsCanAwaitConcurrently(t *testing.T) {
	const count = 8
	start := make(chan struct{})
	results := make(chan error, count)
	var ready sync.WaitGroup
	ready.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			script := New()
			ready.Done()
			<-start
			value, err := script.Run(`await sleep(5); 42`)
			if err == nil && value != int64(42) {
				err = errors.New("wrong result from concurrent script")
			}
			results <- err
		}()
	}
	ready.Wait()
	close(start)
	for i := 0; i < count; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("independent scripts blocked while awaiting")
		}
	}
}

func TestMaxStepsCountsAllCalls(t *testing.T) {
	for _, source := range []string{
		`fn f(...args) { f(...args) }; f()`,
		`let obj = {f: fn() { this.f() }}; obj.f()`,
		`(1..100).map(x => x + 1)`,
	} {
		script := New()
		script.SetMaxSteps(2)
		if _, err := script.Run(source); err == nil || !strings.Contains(err.Error(), "more than 2 steps") {
			t.Errorf("step limit skipped for %q: %v", source, err)
		}
	}
}

func TestEmbeddedFetchNeedsOptIn(t *testing.T) {
	script := New()
	if _, err := script.Run(`await fetch("https://example.com/")`); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("fetch without opt-in: %v", err)
	}
}

func TestEmbeddedSystemIsIsolatedFromGlobalEvaluator(t *testing.T) {
	evaluator.SetSystem(exposedSystem{})
	t.Cleanup(func() { evaluator.SetSystem(nil) })
	script := New()
	for _, source := range []string{`readFile("secret")`, `env("SECRET")`, `exit(0)`} {
		if _, err := script.Run(source); err == nil || !strings.Contains(err.Error(), "not available in embedded scripts") {
			t.Errorf("ambient system access from %q: %v", source, err)
		}
	}
}

func TestReturnedFunctionUsesItsScriptsLoop(t *testing.T) {
	script := New()
	value, err := script.Run(`fn delayed() { await sleep(5); 42 }; delayed`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := value.(func(...any) (any, error))()
	if err != nil || result != int64(42) {
		t.Fatalf("returned function: %v, %v", result, err)
	}
}

func TestStructs(t *testing.T) {
	type Item struct {
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Internal string  `json:"-"`
		Stock    int
		secret   string
	}
	script := New()
	script.Set("item", Item{Name: "Pen", Price: 2.5, Internal: "x", Stock: 3, secret: "y"})
	script.Set("items", []*Item{{Name: "A"}, nil})
	result, err := script.Run("[item.name, item.price, item.Stock, item.keys(), items[0].name, items[1]]")
	if err != nil {
		t.Fatal(err)
	}
	want := []any{"Pen", 2.5, int64(3), []any{"name", "price", "Stock"}, "A", nil}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %#v, want %#v", result, want)
	}
}
