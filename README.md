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

// Use a loop to iterate through an array
print("Skills:")
repeat skill in person.skills {
  print("- " + skill)
}
```

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