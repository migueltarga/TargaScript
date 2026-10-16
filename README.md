![TargaScript Logo](logo.png)

# TargaScript 🚀

> If TypeScript can be rewritten in Go, why not create our own language with Go?

TargaScript is a lightweight scripting language with a clean, intuitive syntax. It's designed to be easy to learn while providing powerful features for various programming tasks. TargaScript files use the `.tg` extension.

**[▶ Try TargaScript in your browser](https://migueltarga.github.io/TargaScript/)**, no install needed. The playground itself is written in TargaScript and runs on WebAssembly.

## 🌟 Why TargaScript

- **One language for scripts and web pages.** The same code runs from a 6 MB command-line binary and in the browser from a WebAssembly runtime under 200 KB: add `<script type="text/targascript">` to a page, with no build step.
- **Asynchronous code that reads top to bottom.** `await fetch(url)` and `await sleep(ms)` work in scripts and in the browser, with no callbacks.
- **Errors that help.** Mistakes point at the exact spot in your code, and a typo gets a "Did you mean...?".
- **Small, modern syntax.** String interpolation, destructuring, spread, `??` and `?.`, default and rest parameters, `try`/`catch`, closures.
- **Built for working with data.** JSON, `map`/`filter`/`reduce`, `groupBy`, `sortBy`, `sum`, `unique` and more, on arrays and objects that keep their order.
- **A bytecode VM.** Programs compile to bytecode, which runs several times faster than the original tree-walking interpreter and allows 10,000 nested calls, in the browser too.

## 📝 A Taste

```
// Count the latest commits of a repository by author
let url = "https://api.github.com/repos/migueltarga/TargaScript/commits?per_page=50"
let response = await fetch(url)
if !response.ok { throw "GitHub answered ${response.status}" }

let byAuthor = response.json().groupBy(fn({commit}) { commit.author.name })

repeat name, commits in byAuthor {
    print("${name.padEnd(20)} ${commits.length} commits")
}
```

And the basics:

```
fn greet(name, greeting = "Hello") {
    "${greeting}, ${name}!"
}

let person = {"name": "Alice", "skills": ["programming", "design"]}
print(greet(person.name))                  // Hello, Alice!

repeat i, skill in person.skills {
    print("${i + 1}. ${skill}")
}

let [first, ...rest] = [1, 2, 3]
print(person.address?.city ?? "unknown")   // unknown
```

## 🔀 Matching Values

`match` picks the first arm whose values match, comparing with `==`. A range matches the numbers between its ends, and `_` matches anything:

```
fn describe(n) {
    match n {
        0 => "none"
        1, 2 => "a few"
        3..9 => "several"
        _ => "many"
    }
}

let grade = match true {
    score >= 90 => "A"
    score >= 80 => "B"
    _ => "C"
}
```

An arm can run a block, `=> { ... }`, and a match with no matching arm is null.

## 🧯 Handling Errors

`try` runs a block and, if anything in it fails, runs the `catch` block with the error. `throw` raises an error of your own:

```
fn parseAge(text) {
    let age = toInt(text)
    if age < 0 { throw "age can't be negative" }
    age
}

let age = try {
    parseAge("-3")
} catch err {
    print("Invalid age:", err.message, "at line", err.line)
    0
}
```

The error is an object with `message`, `line` and `column`. Like `if`, `try` is an expression: its value is the value of whichever block ran last. `throw err` inside a `catch` raises the same error again.

## 🎛️ Default Parameters

Parameters can have a default value, used when the argument is missing or null. It can use the parameters before it:

```
fn greet(name, greeting = "Hello") {
    "${greeting}, ${name}!"
}
print(greet("Ana"))          // Hello, Ana!
print(greet("Ana", "Hi"))    // Hi, Ana!

fn range2(start, end = start + 10) { start..end }
```

## 🧩 Taking Values Apart

`let` and `const` can take arrays and objects apart:

```
let [first, second] = [10, 20, 30]
let {name, age} = {"name": "Ana", "age": 30}

fn minmax(list) { [list.sort().first(), list.last()] }
let [lowest, highest] = minmax([5, 1, 9])
```

Missing elements and properties are null. Patterns also work in loops and parameters, and `{name, age}` is short for `{"name": name, "age": age}`:

```
repeat [x, y] in [[1, 2], [3, 4]] { print(x * y) }

fn area({width, height}) { width * height }
print(area({"width": 3, "height": 4}))

let name = "Ana"
let age = 30
print({name, age})    // {"name": "Ana", "age": 30}
```

`...` spreads arrays into arrays and calls, and objects into objects:

```
let all = [...first, ...second, 99]
let config = {...defaults, "port": 8080}
print(max(...numbers))
```

A last parameter written `...name` collects the extra arguments in an array:

```
fn log(level, ...parts) {
    print("[${level}]", parts.join(" "))
}
log("info", "server", "started")    // [info] server started
```

## 🫥 Working with null

`a ?? b` is `b` when `a` is null, and `a?.b` is null when `a` is null instead of an error. A `?.` skips the rest of the chain, so `user?.name.upper()` is null when there's no user:

```
let config = parseJSON(text)
let port = config.port ?? 8080
let city = user?.address?.city ?? "unknown"
```

## ⏳ Waiting with await

`await` waits for a promise and gives its result, without blocking. `sleep(ms)` returns a promise that resolves after some milliseconds:

```
fn countdown(n) {
    repeat i in 1..n {
        print(n - i + 1)
        await sleep(1000)
    }
    print("Go!")
}

countdown(3)
```

The same code runs in the CLI, where an event loop keeps the program alive while it waits, and in the browser, where `await` also works with JavaScript promises like `fetch`.

`fetch(url, options)` makes HTTP requests, in the CLI and in the browser alike:

```
let response = await fetch("https://api.github.com/repos/migueltarga/TargaScript")
let repo = response.json()
print(repo.full_name, "has", repo.stargazers_count, "stars")

let created = await fetch("https://example.com/api/items", {
    "method": "POST",
    "headers": {"Authorization": "Bearer ..."},
    "body": {"name": "new item"}
})
print(created.status, created.ok)
```

The response has `status`, `ok`, `url`, `headers`, `text` (the body) and `json()`. A body that isn't a string is sent as JSON.

## 📦 Loading Other Files

Split code across files with `load`. A loaded file runs once, and its top-level variables, constants and functions become available:

```
// lib/math.tg
const PI = 3.14159
fn square(x) { x * x }

// main.tg
load "lib/math.tg"          // brings PI and square into scope
print(square(3))

load "lib/math.tg" as math  // or keeps them in an object
print(math.PI * math.square(2))
```

Paths are relative to the file that contains the `load`.

## 📚 Examples Directory

For more comprehensive examples showcasing TargaScript's capabilities, please check out the [`examples/`](./examples/) directory, which contains:

- Fundamental language features demonstrations
- Data processing examples
- Function concepts (closures, recursion, higher-order functions)
- Practical applications (todo list, contact manager)
- And much more!

Each example in the directory is extensively documented and designed to demonstrate specific language features.

## 📥 Installing

Download `targascript` for Linux, macOS or Windows from the [latest release](https://github.com/migueltarga/TargaScript/releases/latest), or install it with Go:

```sh
go install github.com/migueltarga/TargaScript/cmd/targascript@latest
```

Then run a script with `targascript script.tg`, or start the REPL with `targascript`. Each release also has `targascript.js` and `targascript.wasm` for running TargaScript in web pages.

## 🚀 Running TargaScript

To run the TargaScript REPL:
```sh
go run ./cmd/targascript
```

To run a TargaScript (`.tg`) file:
```sh
go run ./cmd/targascript path/to/script.tg
```

Debug mode with AST visualization:
```sh
go run ./cmd/targascript --debug path/to/script.tg
```

## 🌐 TargaScript in the Browser

TargaScript also runs in web pages. The interpreter is compiled to WebAssembly, and scripts reach JavaScript and the DOM through `js`, which is `globalThis`:

```html
<button id="hello">Say hello</button>

<script src="targascript.js"></script>
<script type="text/targascript">
let button = js.document.getElementById("hello")
let clicks = 0

button.addEventListener("click", fn(event) {
  clicks++
  button.textContent = "Hello ${clicks}x!"
})
</script>
```

- `x.name`, `x.name = v`, `x.method(args)` and `x(args)` work on JavaScript values, and `x.new(args)` is `new x(args)`
- numbers, strings, booleans, `null`/`undefined` and arrays become TargaScript values. Arrays and objects passed to JavaScript are copied
- TargaScript functions passed to JavaScript, such as event handlers, can be called from JavaScript
- a JavaScript exception becomes a TargaScript error
- `await` waits for JavaScript promises, and a TargaScript function that awaits returns a promise to JavaScript
- `print` writes to the browser console
- scripts can have a `src`, and all scripts on a page share their variables
- `load` works too, with paths relative to the script: `targascript.js` fetches the files a script loads from the same server before running it
- in a worker, `new Worker("targascript.js?src=worker.tg")` runs `worker.tg`, and `targascript.run(source, onOutput)` runs another program in a fresh environment
- JavaScript can call `targascriptRun(source, onOutput)`, which returns the error message or `null`

`await` makes asynchronous code read like ordinary code:

```html
<script type="text/targascript">
let response = await js.fetch("https://api.github.com/repos/migueltarga/TargaScript")
let repo = await response.json()
js.document.body.append("${repo.full_name} has ${repo.stargazers_count} stars")
</script>
```

Build it with [TinyGo](https://tinygo.org) 0.39:

```sh
./web/build.sh build
```

and serve `build/targascript.js` and `build/targascript.wasm` (about 420 KB, 190 KB compressed) next to your page. Recursion goes up to 10,000 calls in the browser too. Only functions called back by builtins or by JavaScript, like the function given to `map` or an event handler, can nest at most 100 deep there.

## 🤔 Why TargaScript?

Creating a programming language is one of the most educational projects a developer can undertake! TargaScript is an experiment in language design, compiler construction, and a way to better understand how programming languages work under the hood.

## 💡 Inspiration

This project is inspired by and based on the concepts from the excellent book [**"Writing An Interpreter In Go"**](https://interpreterbook.com/) by Thorsten Ball. If you're interested in learning how programming languages work or want to create your own, I highly recommend this resource.

## 📚 License

MIT License

Copyright (c) 2025 Miguel Targa

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

*This README is a work in progress and will be expanded as the language develops.*