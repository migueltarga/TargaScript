![TargaScript Logo](logo.png)

# TargaScript 🚀

> If TypeScript can be rewritten in Go, why not create our own language with Go?

TargaScript is a modern, expressive programming language implemented entirely in Go. TargaScript files use the `.tg` extension.

## 🌟 Features

- Clean, intuitive syntax
- First-class functions
- Object literals and arrays
- Module loading with imports
- Loop constructs like `repeat...in`
- Conditionals with `if/else`
- Property access with dot notation
- Method calls
- No semicolons required!

## 📝 Code Examples

Hello World:
```
print("Hello, World!")
```

Variables and Functions:
```
let name = "Targa"
const version = 1.0
let isCool = true

fn greet(person) {
  print("Hello, " + person)
}

greet(name)  // Prints: Hello, Targa
```

Arrays and Objects:
```
// Arrays
let numbers = [1, 2, 3, 4, 5]
let mixed = [1, "a", true]

// Objects
let person = {
  name: "Targa",
  age: 30,
  hobbies: ["coding", "languages"]
}

// Property access
person.name = "Miguel"
let hobby = person.hobbies[0]
```

Conditional Logic:
```
if (name == "Targa") {
  greet(name)
} else {
  print("Who are you?")
}
```

Loops:
```
repeat i in numbers {
  print("number: " + i)
}
```

Importing Modules:
```
load math
load fs as file
```

## 🚧 Project Status

TargaScript is currently in early development. The parser is functional with support for arrays, objects, property access, method calls, and more.

## 🔨 Building & Running

To run the TargaScript REPL:
```sh
go run main.go
```

To run a TargaScript (`.tg`) file:
```sh
go run main.go path/to/script.tg
```

## 🤔 Why TargaScript?

Because creating a programming language is one of the most fun and educational projects a developer can undertake! TargaScript is an experiment in language design, compiler construction, and a way to better understand how programming languages work under the hood.

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