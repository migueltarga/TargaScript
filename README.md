![TargaScript Logo](logo.png)

# TargaScript 🚀

> If TypeScript can be rewritten in Go, why not create our own language with Go?

TargaScript is a lightweight scripting language with a clean, intuitive syntax. It's designed to be easy to learn while providing powerful features for various programming tasks. TargaScript files use the `.tg` extension.

**[▶ Try TargaScript in your browser](https://migueltarga.github.io/TargaScript/)**, no install needed. The playground itself is written in TargaScript and runs on WebAssembly.

## 🌟 Features

- Clean, intuitive syntax
- First-class functions
- Objects and arrays with method-style dot notation
- Rich set of built-in functions and methods
- Expressive loop constructs with `repeat`
- Variable scoping that behaves intuitively
- String interpolation and formatting
- Powerful data transformation capabilities
- Runs in web pages, with access to JavaScript and the DOM

## 📝 Simple Example

```
// Define a greeting function
fn greet(name) {
  return "Hello, " + name + "!"
}

// Create and use a simple data structure
let person = {
  "name": "Alice",
  "age": 30,
  "skills": ["programming", "design"]
}

// Print a greeting using dot notation to access properties
print(greet(person.name))  // Prints: Hello, Alice!

// Embed expressions in strings with ${...}
print("${person.name} is ${person.age} years old")  // Prints: Alice is 30 years old

// Use a loop to iterate through an array
print("Skills:")
repeat skill in person.skills {
  print("- " + skill)
}
```

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

## 🚀 Running TargaScript

To run the TargaScript REPL:
```sh
go run main.go
```

To run a TargaScript (`.tg`) file:
```sh
go run main.go path/to/script.tg
```

Debug mode with AST visualization:
```sh
go run main.go --debug path/to/script.tg
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

and serve `build/targascript.js` and `build/targascript.wasm` (about 330 KB, 150 KB compressed) next to your page. Recursion goes up to 10,000 calls in the browser too. Only functions called back by builtins or by JavaScript, like the function given to `map` or an event handler, can nest at most 100 deep there.

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