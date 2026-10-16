# TargaScript Reference

Everything in the language, in one place. Code blocks marked `targascript` run as written; `// →` shows what a line prints or evaluates to.

- [Basics](#basics)
- [Operators](#operators)
- [Strings](#strings)
- [Arrays](#arrays)
- [Objects](#objects)
- [Control flow](#control-flow)
- [Functions](#functions)
- [Taking values apart](#taking-values-apart)
- [Null](#null)
- [Errors](#errors)
- [Asynchronous code](#asynchronous-code)
- [Loading files](#loading-files)
- [Builtin functions](#builtin-functions)
- [Command line](#command-line)
- [Web pages](#web-pages)

## Basics

```targascript
// A line comment
/* A block comment,
   over several lines */

let count = 1          // a variable
const limit = 10       // a constant, which can't be assigned again
count = count + 1
print("count:", count) // → count: 2

print(type(42), type(1.5), type("a"), type(true), type(null))
// → integer float string boolean null
print(type([1]), type({"a": 1}), type(fn() { 1 }))
// → array object function
```

Statements end at the end of the line; `;` can separate several on one line. Blocks are in braces, and variables declared in a block only exist in it. Everything is an expression: a block's value is the value of its last statement.

Numbers are 64-bit integers or floats. Integer arithmetic that overflows is an error, and integer division truncates: `7 / 2` is `3`, `7.0 / 2` is `3.5`. Dividing a float by zero gives `Inf` or `NaN`; dividing an integer by zero is an error.

## Operators

From loosest to tightest:

| Operators | Meaning |
|---|---|
| `=` `+=` `-=` `*=` `/=` `%=` `**=` `??=` | assignment |
| `??` | the right side when the left is null |
| `\|\|` | or: the left side if truthy, else the right side |
| `&&` | and: the left side if falsy, else the right side |
| `==` `!=` | equality; arrays and objects compare by content, and `1 == 1.0` |
| `<` `>` `<=` `>=` `in` | comparison of numbers or strings; `x in list` (an element), `"key" in object`, `"part" in text`, `n in 1..10` (between the ends, without building the range) |
| `..` | inclusive range of integers: `1..3` is `[1, 2, 3]` |
| `+` `-` `^` | addition (and joining strings or arrays), subtraction, bitwise xor |
| `*` `/` `%` | multiplication (and repeating strings: `"-" * 3`), division, remainder |
| `**` | power, grouping to the right: `2 ** 3 ** 2` is `512` |
| `-x` `!x` | negation, not |
| `x++` `x--` `f()` `a[i]` `o.p` `o?.p` | increment, call, index, property, optional property |

`false`, `null`, `0`, `0.0`, `""`, `[]` and `{}` are falsy; everything else is truthy.

## Strings

```targascript
let name = "Ana"
print("Hello, ${name}! 2 + 2 = ${2 + 2}")   // → Hello, Ana! 2 + 2 = 4
print("${3.14159:.2f} [${7:3d}] [${"x":-3s}]")  // → 3.14 [  7] [x  ]
print("line one\nline two\ttabbed \"quoted\"")

let s = "Hello, World"
print(s.length, s[0], s[-1], s.slice(-5))   // → 12 H d World
print(s.upper(), s.lower(), s.reverse())
print(s.contains("World"), s.startsWith("He"), s.indexOf("o"))  // → true true 4
print(s.split(", "), s.replace("World", "there"))
print("  pad ".trim(), "7".padStart(3, "0"), "ab".repeat(3))   // → pad 007 ababab
repeat c in "abc" { print(c) }
```

Escapes are `\n`, `\t`, `\r`, `\"`, `\\` and `\$` (for a literal `${`); other backslashes stay as written, so `"\d+"` is `\d+`. `${value:spec}` formats like [`format`](#builtin-functions). Strings can be joined with `+`, which turns the right side into text: `"n = " + 5`.

Patterns are regular expressions in strings:

```targascript
let text = "Ana 31, Bia 25"
print(text.matches("\d+"))                               // → true
print(text.match("(\w+) (\d+)"))                         // → ["Ana 31", "Ana", "31"]
print(text.matchAll("(\w+) (\d+)").map(([_, n, a]) => n)) // → ["Ana", "Bia"]
print(text.replacePattern("(\w+) (\d+)", "$2:$1"))       // → 31:Ana, 25:Bia
print("a1b22c".splitPattern("\d+"))                       // → ["a", "b", "c"]
```

`match` gives the match and its groups, or `null`. The command line uses Go's regular expressions and the browser JavaScript's; their common syntax (classes like `\d` and `\w`, groups, `*`, `+`, `?`, `{n,m}`, `^`, `$`, `|`) works the same in both.

String methods: `length`, `upper`, `lower`, `trim`, `trimStart`, `trimEnd`, `split(sep)`, `replace(old, new)`, `contains(sub)`, `startsWith(prefix)`, `endsWith(suffix)`, `indexOf(sub)`, `count(sub)`, `slice(start, end)`, `chars()`, `lines()`, `repeat(n)`, `padStart(width, fill)`, `padEnd(width, fill)`, `reverse()`, `capitalize()`, and with patterns `matches`, `match`, `matchAll`, `replacePattern` and `splitPattern`. Positions count characters, and `str[i]` is `null` outside the string.

## Arrays

```targascript
let numbers = [5, 3, 8, 1]
print(numbers.length, numbers[0], numbers[-1], numbers.first())  // → 4 5 1 5
numbers.insert(9)                      // add at the end
print(numbers.map(n => n * 2))         // → [10, 6, 16, 2, 18]
print(numbers.filter(n => n > 4))      // → [5, 8, 9]
print(numbers.reduce((a, b) => a + b, 0), numbers.sum())  // → 26 26
print(numbers.sort())                  // sorts in place → [1, 3, 5, 8, 9]
print([1, [2, [3]]].flatten(), [1, 1, 2].unique())  // → [1, 2, [3]] [1, 2]
print(["bb", "a"].sortBy(s => s.length))            // → ["a", "bb"]
print(["apple", "avocado", "kiwi"].groupBy(w => w[0]))
// → {"a": ["apple", "avocado"], "k": ["kiwi"]}
```

Array methods:

- **Reading**: `length`, `first()`, `last()`, `rest()`, `slice(start, end)`, `contains(value)`, `indexOf(value)`, `find(fn)`, `some(fn)`, `every(fn)`, `count(value or fn)`, `min()`, `max()`, `sum()`.
- **Transforming**: `map(fn)`, `filter(fn)`, `reduce(fn, initial)`, `join(sep)`, `unique()`, `flatten(depth)`, `groupBy(fn)`.
- **Changing the array in place**:
  - `insert(value)` and `unshift(value)` add at the end or the start;
  - `pop()` and `shift()` remove from the end or the start;
  - `removeAt(i)` removes at a position;
  - `sort()`, `sort(compare)`, `sortBy(fn)` and `reverse()` reorder it.

`array[i]` is `null` outside the array, and negative positions in `slice` and `removeAt` count from the end.

## Objects

Objects map keys to values and keep keys in the order they were added. Bare names are string keys.

```targascript
let person = {"name": "Ana", age: 31}
person.city = "Lisbon"
person["age"] += 1
print(person.name, person.age, person.keys())   // → Ana 32 ["name", "age", "city"]
print(person.has("city"), person.get("email", "none"))  // → true none
print(person.entries())
// → [["name", "Ana"], ["age", 32], ["city", "Lisbon"]]
person.remove("city")

let x = 1
let y = 2
print({x, y})                                    // → {"x": 1, "y": 2}

let counter = {"count": 0, "inc": fn() { counter.count += 1 }}
counter.inc()
print(counter.count)                             // → 1
```

Object methods: `keys()`, `values()`, `entries()`, `has(key)`, `get(key, default)`, `remove(key)`, `length`. A missing property reads as `null`.

## Control flow

```targascript
let n = 7
if n > 5 { print("big") } else if n > 2 { print("medium") } else { print("small") }

let size = match n {
    0 => "none"
    1, 2 => "a few"
    3..9 => "several"
    _ => "many"
}
print(size)                                   // → several

repeat i in 1..3 { print(i) }                 // 1, 2, 3
repeat 2 { print("twice") }                   // a number of times
repeat i in 2 { print("time", i) }           // an integer n counts from 1 to n
repeat i, name in ["a", "b"] { print(i, name) }        // index and element
repeat key, value in {"x": 1} { print(key, value) }    // key and value
let k = 0
repeat k < 3 { k++ }                          // while the condition holds
repeat i in 1..10 {
    if i % 2 == 0 { continue }
    if i > 5 { break }
    print(i)                                  // 1, 3, 5
}
```

`if` and `match` are expressions. `match` compares each arm's values with `==`, a range matches the numbers between its ends, and `_` matches anything; without a match it's null. `repeat x in` works on ranges, arrays, strings (characters), objects (keys) and integers (`1..n`).

## Functions

```targascript
fn add(a, b) {
    return a + b          // or just `a + b`: the last value is returned
}
let double = fn(x) { x * 2 }
let triple = x => x * 3
let multiply = (a, b) => a * b

fn greet(name, greeting = "Hello") { "${greeting}, ${name}!" }
fn sum(...numbers) { numbers.reduce((a, b) => a + b, 0) }
print(greet("Ana"), sum(1, 2, 3), sum(...[4, 5]))    // → Hello, Ana! 6 9

fn makeCounter() {
    let count = 0
    () => count++        // closures can change the variables they capture
}
let next = makeCounter()
next()
print(next())            // → 2

fn fact(n) { if n <= 1 { 1 } else { n * fact(n - 1) } }
print(fact(20))          // → 2432902008176640000
```

Missing arguments are `null` (or their default), and extra ones are ignored. Recursion can go 10,000 calls deep.

## Taking values apart

```targascript
let [first, second, ...others] = [1, 2, 3, 4]
let {name, ...details} = {"name": "Ana", "age": 31, "city": "Rio"}
print(first, others, name, details)   // → 1 [3, 4] Ana {"age": 31, "city": "Rio"}

repeat [x, y] in [[1, 2], [3, 4]] { print(x * y) }
fn area({width, height}) { width * height }
print(area({"width": 2, "height": 3}))     // → 6
print([[1, 2]].map(([a, b]) => a + b))     // → [3]

let defaults = {"port": 80, "host": "localhost"}
print({...defaults, "port": 8080}, [...[1, 2], 3])
```

Patterns work in `let`, `const`, `repeat` and parameters. Missing parts are `null`.

## Null

```targascript
let config = {"theme": "dark"}
print(config.size ?? 14)                     // → 14
let user = {"address": null}
print(user.address?.city ?? "no city")       // → no city
config.size ??= 12                           // assigns only when null
```

`a?.b`, `a?.method()` give `null` when `a` is null, skipping the rest of the chain.

## Errors

```targascript
fn parse(text) {
    let n = toInt(text)
    if n < 0 { throw "negative: ${n}" }
    n
}

let result = try {
    parse("-5")
} catch err {
    print(err.message, "at line", err.line)   // → negative: -5 at line 3
    0
}
print(result)                                // → 0
```

An error stops the program, showing where it happened, unless a `try` around it catches it. The caught error is an object with `message`, `line` and `column`; `throw err` raises it again. `try` is an expression, like `if`.

## Asynchronous code

```targascript
print("waiting...")
await sleep(100)
print("done")

fn later(value) {
    await sleep(10)
    value * 2
}
print(later(21))                              // → 42
```

`await` waits for a promise and gives its result; on anything else it gives the value itself. Functions that await pause the whole program until the promise settles, so calling them needs no `await`. `sleep(ms)` and `fetch(url, options)` return promises; in the browser, so do many JavaScript functions. `await` can't pause inside a function called back by a builtin like `map`.

```
let response = await fetch("https://api.github.com/repos/migueltarga/TargaScript")
if !response.ok { throw "status ${response.status}" }
print(response.json().full_name)

let created = await fetch(url, {"method": "POST", "headers": {"X-Token": token}, "body": {"name": "x"}})
```

A response has `status`, `ok`, `url`, `headers`, `text` and `json()`. A `body` that isn't a string is sent as JSON.

## Loading files

```
load "lib/math.tg"            // its top-level names become available here
load "lib/text.tg" as text    // or are kept in an object: text.shout(...)
```

Paths are relative to the file with the `load`. A file runs once, however many times it's loaded. In web pages, `targascript.js` fetches the files from the same server.

## Builtin functions

| Function | Does |
|---|---|
| `print(...values)` | prints values separated by spaces |
| `printf(format, ...values)` / `format(format, ...values)` | prints / returns values formatted with `%d`, `%s`, `%.2f`, `%5d`, `%-10s`, `%x`, `%v`... |
| `type(value)` | `"integer"`, `"float"`, `"string"`, `"boolean"`, `"null"`, `"array"`, `"object"`, `"function"`, `"promise"` |
| `toString(v)`, `toInt(v)`, `toFloat(v)` | convert values |
| `abs`, `sqrt`, `pow`, `round`, `floor`, `ceil`, `min(a, b)`, `max(a, b)` | math |
| `random()` | a float between 0 and 1 |
| `range(start, end)` | the array `start..end` |
| `now()` | the current Unix time in seconds |
| `timeFormat(seconds, layout)` | formats a time with `YYYY`, `YY`, `MMMM`, `MMM`, `MM`, `DD`, `dddd`, `ddd`, `HH`, `hh`, `mm`, `ss` and `A` (AM/PM), e.g. `timeFormat(now(), "DD/MM/YYYY HH:mm")` |
| `clock()` | milliseconds since the program started, for timing code |
| `parseJSON(text)`, `toJSON(value, indent)` | JSON, keeping the order of keys |
| `sleep(ms)`, `fetch(url, options)` | promises for `await` |
| `readFile`, `writeFile`, `appendFile`, `listDir`, `env`, `exit` | [command line](#command-line) only |

## Command line

```sh
targascript                       # the REPL; unfinished code continues on the next line
targascript script.tg a b         # run a file; it sees ["a", "b"] in args
targascript -e 'print(2 ** 10)'   # run code given on the command line
cat script.tg | targascript -     # run code from standard input
```

Scripts see their arguments in `args`, and can use `readFile(path)`, `writeFile(path, text)`, `appendFile(path, text)`, `listDir(path)`, `env(name, default)` and `exit(code)`. Paths are relative to the working directory.

## Web pages

```html
<script src="targascript.js"></script>
<script type="text/targascript">
let button = js.document.getElementById("hello")
button.addEventListener("click", () => {
    button.textContent = "Clicked!"
})
</script>
```

`js` is JavaScript's `globalThis`:
- **JavaScript values**:
  - `x.name`, `x.name = v`, `x.method(args)` and `x(args)` work on them;
  - `x.new(args)` is `new x(args)`;
  - a JavaScript exception is a TargaScript error.
- **Conversions**: numbers, strings, booleans, `null` and arrays convert to TargaScript values. Arrays and objects passed to JavaScript are copied.
- **Functions**: TargaScript functions passed to JavaScript, like event handlers, can be called from JavaScript, and they return a promise if they await.
- **`print`**: it goes to the console.

Scripts can have a `src` and share their variables. In a worker created with `new Worker("targascript.js?src=worker.tg")`, `targascript.run(source, onOutput)` runs another program in a fresh environment. JavaScript can call `targascriptRun(source, onOutput)` and `targascriptCheck(source)`.
