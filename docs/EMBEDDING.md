# Embedding TargaScript in Go

TargaScript can run inside a Go program, so its users can write rules, formulas, plugins or configuration in a small, readable language, without access to your files or your process.

```sh
go get github.com/migueltarga/TargaScript
```

```go
import targascript "github.com/migueltarga/TargaScript"

script := targascript.New()
script.Set("order", order)            // any Go value: maps, slices, structs' fields...
script.Set("shipping", shippingCost)  // Go functions scripts can call

total, err := script.Run(userRule)    // the value of the last expression
```

## Values

Values cross like JSON:

| Go | TargaScript | Back in Go |
|---|---|---|
| `nil` | `null` | `nil` |
| `bool` | boolean | `bool` |
| integers | integer | `int64` |
| floats | float | `float64` |
| `string` | string | `string` |
| slices and arrays | array | `[]any` |
| maps | object, keys sorted | `map[string]any` |
| functions | function | `func(args ...any) (any, error)` |

A Go function's arguments are converted to its parameter types, and numbers convert between integers and floats. When its last result is an `error` that isn't `nil`, the call fails in the script, where `try` can catch it.

## A Script

A `Script` keeps what it defines between runs, so a program can load the user's functions once and call them many times:

```go
script := targascript.New()
script.Run(`fn score(player) { player.wins * 3 + player.draws }`)

points, err := script.Call("score", map[string]any{"wins": 10, "draws": 2})  // 32
count := script.Get("someGlobal")
```

- `SetOutput(w)` sends `print` somewhere else than standard output.
- `SetMaxSteps(n)` stops a script after that many loop iterations and calls, so a user's endless loop can't hang your program.
- Errors are `*targascript.Error`, with `Message`, `Line` and `Column`. Their `Error()` shows the line of code with a caret.

A `Script` isn't safe to use from several goroutines at once. Give each goroutine its own.

## What scripts can do

Scripts can compute, call the functions you give them, `await`, `fetch` and `sleep`. They can't read or write files, read the environment or exit. Those builtins only exist in the `targascript` command.
