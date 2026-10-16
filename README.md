![TargaScript Logo](logo.png)

# TargaScript 🚀

> If TypeScript can be rewritten in Go, why not create our own language with Go?

TargaScript is a lightweight scripting language with a clean, intuitive syntax. It's designed to be easy to learn while providing powerful features for various programming tasks. TargaScript files use the `.tg` extension.

## 🌟 Features

- Clean, intuitive syntax
- First-class functions
- Objects and arrays with method-style dot notation
- Rich set of built-in functions and methods
- Expressive loop constructs with `repeat`
- Variable scoping that behaves intuitively
- String interpolation and formatting
- Powerful data transformation capabilities

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
- `print` writes to the browser console
- scripts can have a `src`, and all scripts on a page share their variables
- in a worker, `new Worker("targascript.js?src=worker.tg")` runs `worker.tg`, and `targascript.run(source, onOutput)` runs another program in a fresh environment
- JavaScript can call `targascriptRun(source, onOutput)`, which returns the error message or `null`

Build it with [TinyGo](https://tinygo.org) 0.39:

```sh
./web/build.sh build
```

and serve `build/targascript.js` and `build/targascript.wasm` (about 260 KB, 120 KB compressed) next to your page. In the browser, recursion is limited to about 300 nested calls, because the browser's stack is much smaller than a native one.

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