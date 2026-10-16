# Embedding TargaScript in Go

TargaScript can run inside a Go program, so its users can write rules, formulas, plugins or configuration in a small, readable language. Embedded scripts cannot load files, read the environment, run commands or exit the process. Network access is disabled until you allow specific hosts.

```sh
go get github.com/migueltarga/TargaScript
```

```go
import targascript "github.com/migueltarga/TargaScript"

script := targascript.New()
script.Set("order", order)            // maps, slices, structs, numbers, strings...
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
| structs | object of the exported fields, named by their `json` tags | `map[string]any` |
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
- `SetMaxSteps(n)` stops scripts after that many loop iterations and calls. This is a computation limit, not a memory limit or a deadline for Go functions you expose.
- `AllowFetchHosts("api.example.com")` allows `fetch` to contact that exact hostname, including redirects only to allowed hosts. By default, `fetch` is denied.
- `SetMaxFetchBytes(n)` limits the response body size of allowed requests. The default is 1 MiB.
- Errors are `*targascript.Error`, with `Message`, `Line` and `Column`. Their `Error()` shows the line of code with a caret.

A `Script` isn't safe to use from several goroutines at once. Give each goroutine its own; separate scripts have separate event loops.

## What scripts can do

Scripts can compute, call the functions you give them, `await` and `sleep`. They can use `fetch` only for hosts you explicitly allow. They cannot read or write files, load files, read the environment, run commands or exit through the language's builtins. Those builtins only exist in the `targascript` command. Go functions that you expose can still perform their own I/O, so only provide functions you intend scripts to use.
