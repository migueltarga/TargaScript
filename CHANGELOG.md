# Changelog

## 0.1.0 (unreleased)

The first release. TargaScript went from a tree-walking interpreter to a language with a bytecode VM, a browser runtime, a standard toolkit for scripts and servers, and tools for editors.

### Runs everywhere

- A bytecode compiler and VM, with [benchmarks](bench/) against Python, Node and Lua
- The same language in the browser: a WebAssembly runtime of about 200 KB compressed, `<script type="text/targascript">` in any page, and a [playground](https://script.targa.me/) that is itself written in TargaScript
- `targascript build app.tg` makes a single executable with the program and everything it loads
- Output to files and pipes is buffered, so printing many lines doesn't make a system call for each; it's written out when the program waits, reads input, runs a command, ends or gets Ctrl+C
- A Go package to [embed](docs/EMBEDDING.md) TargaScript in Go programs, for rules, formulas and plugins

### Language

- Enums, with `match` checking that every value has an arm: `enum Status { Active, Blocked }`
- Structs, with fields, defaults and methods: `struct Point { x = 0, y = 0, fn length() { ... } }`
- `match` with ranges, several values per arm and guards: `_ if n > 100 => ...`
- Destructuring of arrays and objects, with rest, defaults and renaming: `let {port = 8080, host: h, ...rest} = config`
- Spread in arrays, objects and calls; rest parameters
- Arrow functions, default parameters, and `this` in methods
- `it` for short functions: `people.filter(it.age > 25).map(it.name)`
- The pipe operator: `data |> parse |> summarize`, and into methods: `xs |> filter(it > 0) |> sum`
- `??`, `?.`, `?.[` and `??=`; `**`; `+=` and the other compound assignments
- `try`/`catch`/`finally`/`throw`, with `throw` usable as a value: `x ?? throw "missing"`
- `await` anywhere, `all(promises)`, and `map` running functions that await side by side
- `in` for arrays, objects, strings and ranges
- Strings in backticks for JSON and HTML, interpolation with format specs: `${price:.2f}`
- A `#!/usr/bin/env targascript` first line, to run scripts as programs
- Number literals like `1_000_000`, `0xff` and `0b1010`; negative indexes; `[1] + [2]`, `"-" * 20`; arrays compare element by element

### Library

- JSON that keeps the order of keys, regular expressions (with function replacements), dates with readable layouts (`timeFormat`, `parseTime`)
- Array methods like `groupBy`, `sortBy`, `chunk`, `zip`, `toObject`, `unique`, `sum`; callbacks get the index when they ask for it
- `fetch`, `sleep`, `input`, files and directories, environment variables, exit codes, and `exec` to run other programs
- `serve(port, handler)` for HTTP servers, with `response`, `staticFile`, `escapeHTML` and handlers that can await
- `load` for other files, also from web addresses

### Tools

- Errors that show the code, the calls that led to them, "Did you mean...?" for typos, and what to write instead of other languages' names and syntax (`push`, `len`, `for`, `===`...)
- `targascript test` with `test`, `describe`, `assert` and `assertEqual`, and `-run` to pick tests by name
- `--watch` to run a program or the tests again whenever a file changes
- `targascript check` to find problems without running, and `targascript fmt` to lay files out (also in VS Code, the playground and the language server)
- A REPL with line editing, history and Tab completion
- A VS Code extension with highlighting, problems as you type, hover docs, completion, snippets and Run File
- `targascript lsp` for other editors: problems, completion, hover, go to definition, outline and formatting
- A [reference](docs/REFERENCE.md) and a [cookbook](docs/COOKBOOK.md) whose examples are run by the tests
