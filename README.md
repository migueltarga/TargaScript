![TargaScript Logo](logo.png)

# TargaScript 🚀

> If TypeScript can be rewritten in Go, why not create our own language with Go?

TargaScript is a modern, expressive programming language implemented entirely in Go. TargaScript files use the `.tg` extension.

## 🌟 Features

- Clean, intuitive syntax
- First-class functions
- Direct function declarations with `fn` keyword
- Object literals and arrays
- Array and object methods with dot notation
- Module loading with imports
- Loop constructs like `repeat...in`
- Conditionals with `if/else`
- Property access with dot notation
- Method calls
- Unicode and emoji support ✨
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
let emptyValue = null

// Variable naming rules:
// - Must start with a letter or underscore (not a digit)
// - Can contain letters, digits, and underscores after the first character
let validName = 123       // Valid
let _alsoValid = true     // Valid
let with2numbers = "ok"   // Valid
// let 3invalid = "error"  // Invalid: cannot start with a digit

// Direct function declaration
fn greet(person) {
  print("Hello, " + person)
}

// Function expression assigned to a variable
let double = fn(x) {
  return x * 2
}

greet(name)    // Prints: Hello, Targa
print(double(5))  // Prints: 10
```

Arrays and Objects:
```
// Arrays
let numbers = [1, 2, 3, 4, 5]
let mixed = [1, "a", true]

// Array methods with dot notation
let first = numbers.first()  // 1
let last = numbers.last()    // 5
let length = numbers.length  // 5
let newArray = numbers.insert(6)  // [1, 2, 3, 4, 5, 6]
let remaining = numbers.rest()    // [2, 3, 4, 5]

// Array indexing
let third = numbers[2]       // Accessing element (zero-based, returns 3)
numbers[0] = 10              // Modify array element

// Strings with dot notation
let greeting = "Hello"
let strLength = greeting.length  // 5

// Objects
let person = {
  "name": "Targa",
  "age": 30,
  "skills": ["coding", "languages"]
}

// Property access using dot notation
let name = person.name      // "Targa"
let skill = person.skills[0]  // "coding"

// Property access using bracket notation
let key = "age"  
let age = person[key]        // 30

// Property assignment (dot notation)
person.name = "Miguel"         // Update existing property
person.location = "New York"   // Add new property

// Property assignment (bracket notation)
person["active"] = true        // Add new property

// Array access by index
let item = numbers[0]     // Access first item
numbers[1] = 20           // Modify array element
```

Conditional Logic:
```
if name == "Targa" {
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

Operators:
```
// Arithmetic operators
let a = 10 + 5  // Addition: 15
let b = 10 - 5  // Subtraction: 5
let c = 10 * 5  // Multiplication: 50
let d = 10 / 5  // Division: 2
let e = 10 % 3  // Modulo (remainder): 1

// Bitwise operators
let f = 10 ^ 5  // Bitwise XOR: 15 (1010 ^ 0101 = 1111)

// Comparison operators
let g = 10 == 5 // Equality: false
let h = 10 != 5 // Inequality: true
let i = 10 > 5  // Greater than: true
let j = 10 < 5  // Less than: false
let k = 10 >= 5 // Greater than or equal: true
let l = 10 <= 5 // Less than or equal: false

// Logical operators
let m = true && false // Logical AND: false
let n = true || false // Logical OR: true
let o = !true         // Logical NOT: false
```

## 📦 Built-in Functions

TargaScript includes several built-in functions for common operations:

- `print(value)` - Prints the value to the console

Additionally, arrays, strings, and objects support properties and methods via dot notation:

For arrays:
- `array.first()` - Returns the first element of an array
- `array.last()` - Returns the last element of an array
- `array.rest()` - Returns a new array containing all elements except the first
- `array.insert(element)` - Returns a new array with the element added to the end
- `array.length` - Property that returns the length of the array

For strings:
- `string.length` - Property that returns the length of the string

For objects:
- `object.keys()` - Returns an array of all keys in the object
- `object.values()` - Returns an array of all values in the object
- `object.has(key)` - Returns a boolean indicating if the object has the specified key
- `object.length` - Property that returns the number of key-value pairs in the object

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

Debug mode with AST visualization (with syntax highlighting):
```sh
go run main.go --debug path/to/script.tg
```

Debug mode with AST visualization (no colors, better for terminals that don't support ANSI colors):
```sh
go run main.go --debug-no-color path/to/script.tg
```

## 🤔 Why TargaScript?

Because creating a programming language is one of the most fun and educational projects a developer can undertake! TargaScript is an experiment in language design, compiler construction, and a way to better understand how programming languages work under the hood.

## 💡 Inspiration

This project is inspired by and based on the concepts from the excellent book [**"Writing An Interpreter In Go"**](https://interpreterbook.com/) by Thorsten Ball. If you're interested in learning how programming languages work or want to create your own, I highly recommend this resource. The book takes you step-by-step through the process of building an interpreter for the Monkey programming language, from lexing and parsing to evaluation.

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