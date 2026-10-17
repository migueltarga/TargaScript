# TargaScript Reference

Everything in the language, in one place. For complete programs that do everyday jobs, see the [cookbook](COOKBOOK.md). Code blocks marked `targascript` run as written; `// →` shows what a line prints or evaluates to.

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
- [Testing](#testing)
- [Command line](#command-line)
- [Web pages](#web-pages)
- [Embedding in Go](EMBEDDING.md), in its own page

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

Statements end at the end of the line, unless the next line starts with an operator like `|>`, `.`, `+` or `&&`, which continues the expression; `(`, `[` and `-` at the start of a line always begin a new statement. `;` can separate several statements on one line. Blocks are in braces, and variables declared in a block only exist in it. Everything is an expression: a block's value is the value of its last statement.

Numbers are 64-bit integers or floats. Integer arithmetic that overflows is an error, and integer division truncates: `7 / 2` is `3`, `7.0 / 2` is `3.5`. Dividing a float by zero gives `Inf` or `NaN`; dividing an integer by zero is an error.

```targascript
print(1_000_000, 0xff, 0b1010, 0o17, 2.5e3)   // → 1000000 255 10 15 2500
```

## Operators

From loosest to tightest:

| Operators | Meaning |
|---|---|
| `=` `+=` `-=` `*=` `/=` `%=` `**=` `??=` | assignment |
| `\|>` | pipe: `x \|> f` is `f(x)` and `x \|> f(a)` is `f(x, a)`; when `f` isn't a function, they are the methods `x.f()` and `x.f(a)`, as in `xs \|> map(it * 2) \|> sum`; `x \|> it.trim()` uses `it` for `x` |
| `??` | the right side when the left is null |
| `\|\|` | or: the left side if truthy, else the right side |
| `&&` | and: the left side if falsy, else the right side |
| `==` `!=` | equality; arrays and objects compare by content, and `1 == 1.0` |
| `<` `>` `<=` `>=` `in` | comparison of numbers, strings, or arrays element by element; `x in list` (an element), `"key" in object`, `"part" in text`, `n in 1..10` (between the ends, without building the range) |
| `..` | inclusive range of integers: `1..3` is `[1, 2, 3]` |
| `+` `-` `^` | addition (and joining strings or arrays), subtraction, bitwise xor |
| `*` `/` `%` | multiplication (and repeating strings: `"-" * 3`), division, remainder |
| `**` | power, grouping to the right: `2 ** 3 ** 2` is `512` |
| `-x` `!x` | negation, not |
| `x++` `x--` `f()` `a[i]` `o.p` `o?.p` `a?.[i]` | increment, call, index, property, optional property and index |

`false`, `null`, `0`, `0.0`, `""`, `[]` and `{}` are falsy; everything else is truthy.

The pipe passes a value through functions from left to right, as the first argument, and a line can start with it:

```targascript
fn double(x) { x * 2 }
fn add(x, n) { x + n }

let result = 5
    |> add(3)
    |> double
print(result)   // → 16
```

## Strings

```targascript
let name = "Ana"
print("Hello, ${name}! 2 + 2 = ${2 + 2}")   // → Hello, Ana! 2 + 2 = 4
print("${3.14159:.2f} [${7:3d}] [${"x":-3s}]")  // → 3.14 [  7] [x  ]
print("${1234567:,} ${9876.5:,.2f}")            // → 1,234,567 9,876.50
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

Strings in backticks can span lines and take text as written, quotes and backslashes included, which suits JSON, HTML and patterns. Only `` \` `` and `\${` are escapes, and `${...}` still interpolates:

```targascript
let user = parseJSON(`{"name": "Ana", "tags": ["admin"]}`)
let card = `<div class="card">
  <b>${user.name}</b>
</div>`
print(card.lines().length)   // → 3
```

Patterns are regular expressions in strings:

```targascript
let text = "Ana 31, Bia 25"
print(text.matches("\d+"))                               // → true
print(text.match("(\w+) (\d+)"))                         // → ["Ana 31", "Ana", "31"]
print(text.matchAll("(\w+) (\d+)").map(([_, n, a]) => n)) // → ["Ana", "Bia"]
print(text.replacePattern("(\w+) (\d+)", "$2:$1"))       // → 31:Ana, 25:Bia
print("a1b22c".splitPattern("\d+"))                       // → ["a", "b", "c"]
```

`replacePattern` also takes a function, which gets the match and its groups and returns the text to put in its place: `"{{name}}".replacePattern("\{\{(\w+)\}\}", (whole, key) => data[key])`. `match` gives the match and its groups, or `null`. The command line uses Go's regular expressions and the browser JavaScript's; their common syntax (classes like `\d` and `\w`, groups, `*`, `+`, `?`, `{n,m}`, `^`, `$`, `|`) works the same in both.

String methods: `length`, `upper`, `lower`, `trim`, `trimStart`, `trimEnd`, `split(sep)`, `replace(old, new)`, `contains(sub)`, `startsWith(prefix)`, `endsWith(suffix)`, `indexOf(sub)`, `count(sub)`, `slice(start, end)`, `chars()`, `lines()`, `repeat(n)`, `padStart(width, fill)`, `padEnd(width, fill)`, `reverse()`, `capitalize()`, and with patterns `matches`, `match`, `matchAll`, `replacePattern` and `splitPattern`. Positions count characters, and `str[i]` is `null` outside the string.

## Bytes

`bytes` holds binary data, such as the contents of a file or an image: numbers from 0 to 255, read and changed by position.

```targascript
let data = bytes("Hi")                  // the UTF-8 of a text
print(data, data.length, data[0])       // → bytes(48 69) 2 72
data[0] = 104
print(data.text(), data.hex())          // → hi 6869

let header = bytes(8)                   // eight zero bytes
header.write(0, 4, 0x04034b50)          // a 4-byte number, little-endian
header.writeBE(4, 2, 513)               // a 2-byte one, big-endian
print(header, header.read(0, 4) == 0x04034b50, header.readBE(4, 2))
// → bytes(50 4b 03 04 02 01 00 00) true 513
print(bytes([1, 2]) + bytes([255]), bytes([1]) == bytes([1]))   // → bytes(01 02 ff) true
```

`bytes(n)` makes n zero bytes, `bytes(text)` the UTF-8 of a string, `bytes(list)` the bytes of a list of numbers from 0 to 255, and `bytes(data)` a copy.

Bytes methods: `length`; `slice(start, end)`, a copy, with negative positions counting from the end; `text()`, the bytes read as UTF-8; `hex()`; `toArray()`; `read(position, size)` and `readBE(position, size)`, a number of 1, 2, 4 or 8 bytes, little-endian or big-endian; and `write(position, size, value)` and `writeBE(position, size, value)`, which store one. `repeat b in data` goes over the numbers, `+` joins bytes, and `==` compares them. `data[i]` is `null` outside the data, and storing anything other than a number from 0 to 255 is an error.

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
print([["Rio", "Bia"], ["SP", "Ana"], ["Rio", "Ana"]].sort())  // arrays sort element by element
// → [["Rio", "Ana"], ["Rio", "Bia"], ["SP", "Ana"]]
print([1, 2, 3].chunk(2), ["a", "b"].zip([1, 2]))   // → [[1, 2], [3]] [["a", 1], ["b", 2]]
print(["apple", "avocado", "kiwi"].groupBy(w => w[0]))
// → {"a": ["apple", "avocado"], "k": ["kiwi"]}
```

Array methods:

- **Reading**: `length`, `first()`, `last()`, `rest()`, `slice(start, end)`, `contains(value)`, `indexOf(value)`, `find(fn)`, `some(fn)`, `every(fn)`, `count(value or fn)`, `min()`, `max()`, `sum()`.
- **Transforming**: `map(fn)`, `filter(fn)`, `reduce(fn, initial)`, `join(sep)`, `unique()`, `flatten(depth)`, `groupBy(fn)`, `chunk(size)`, `zip(other)`, `toObject()` (from `[key, value]` pairs, the opposite of `entries()`).
- **Changing the array in place**:
  - `insert(value)` and `unshift(value)` add at the end or the start;
  - `pop()` and `shift()` remove from the end or the start;
  - `removeAt(i)` removes at a position;
  - `sort()`, `sort(compare)`, `sortBy(fn)` and `reverse()` reorder it.

An argument that uses `it` is a function of one value, called `it`, so short functions need no parameter:

```targascript
let people = [{name: "Ana", age: 30}, {name: "Bo", age: 20}]
print(people.filter(it.age > 25).map(it.name))   // → ["Ana"]
print([3, 1, 2].sortBy(-it))                     // → [3, 2, 1]
print([[1, 2], [3]].map(it.map(it * 10)))        // the innermost argument has its own it → [[10, 20], [30]]
```

Only the argument that uses `it` itself becomes the function, not one around the call it's in, and an `it` inside a function written out is that function's. `it` alone as an argument, as in `max(it, 0)`, is the `it` around the call. After `|>`, `it` is the value going through: `"  Hi " |> it.trim().lower()`. Where `it` is a variable, it's the variable.

Functions given to `map`, `filter`, `find`, `some`, `every`, `count`, `sortBy` and `groupBy` get the position too when they take a second parameter, as in `list.map((item, i) => "${i + 1}. ${item}")`; `reduce` passes it third.

`array[i]` is `null` outside the array, and negative positions in `array[i]`, `slice` and `removeAt` count from the end.

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

let counter = {"count": 0, "inc": fn() { this.count += 1 }}
counter.inc()
print(counter.count)                             // → 1

fn point(x, y) {
    {x, y, plus: fn(other) { point(this.x + other.x, this.y + other.y) }}
}
print(point(1, 2).plus(point(3, 4)).x)           // → 4
```

Inside a function called as a method, `obj.method()`, `this` is the object. Arrow functions see the `this` of the function around them, so `this.items.map(i => this.total + i)` works. Elsewhere `this` is `null`.

Object methods: `keys()`, `values()`, `entries()`, `has(key)`, `get(key, default)`, `remove(key)`, and `length`, the number of keys. An object's own properties come first: `{"length": 300}.length` is `300`, and an object's own `keys` function replaces the method. A missing property reads as `null`.

## Structs

A struct names a kind of object: its fields, in order, with defaults, and its methods. Calling the struct makes a value; a field left out takes its default, or `null`.

```targascript
struct Point {
    x = 0, y = 0
    fn length() { (this.x ** 2 + this.y ** 2) ** 0.5 }
    fn plus(other) { Point(this.x + other.x, this.y + other.y) }
}
let p = Point(3, 4)
print(p, p.length())                             // → Point {"x": 3, "y": 4} 5
print(p.plus(Point(1)))                          // → Point {"x": 4, "y": 4}
print(Point(), type(p), type(Point))             // → Point {"x": 0, "y": 0} Point struct
print(p == Point(3, 4), p == {x: 3, y: 4})       // → true false
```

A struct's values are objects, so `p.x = 10`, `p.keys()` and the other object methods work on them; their own properties come before the struct's methods. Two values are equal when they are of the same struct and have the same fields. Calling a struct with more values than it has fields is an error. `struct` is only a keyword before a name, so it can still name a variable.

## Enums

An enum is a set of named values. Each value is equal only to itself, prints with the enum's name, and becomes its name in JSON.

```targascript
enum Status { Active, Blocked, Pending }

fn label(s) {
    match s {
        Status.Active => "on"
        Status.Blocked, Status.Pending => "off"
    }
}
print(Status.Active, label(Status.Pending))      // → Status.Active off
print(Status.values(), Status.Blocked.name)      // → [Status.Active, Status.Blocked, Status.Pending] Blocked
print(type(Status.Active), toJSON({s: Status.Active}))  // → Status {"s":"Active"}
```

A `match` whose arms are all values of one enum, with no `_` arm, must have an arm for each value: one that leaves some out is an error, which `targascript check` and editors show before the program runs, and which the program raises if a value with no arm gets there. Arms with a guard don't count, since they may not match. A misspelled value, as in `Status.Actve`, is an error found the same way.

## Control flow

```targascript
let n = 7
if n > 5 { print("big") } else if n > 2 { print("medium") } else { print("small") }

let size = match n {
    0 => "none"
    1, 2 => "a few"
    3..9 => "several"
    _ if n > 1000 => "lots"                   // a guard: the arm needs it to be true too
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
let point = (x, y) => {x, y}          // braces with name: or name, are an object
print(point(1, 2))                    // → {"x": 1, "y": 2}

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

Patterns work in `let`, `const`, `repeat` and parameters. Missing parts are `null`, unless the name has a default, used when the part is missing or `null`:

```targascript
let {port = 8080, host = "localhost"} = {"host": "example.com"}
print(port, host)                          // → 8080 example.com
fn connect({url, retries = 3}) { "${url} x${retries}" }
print(connect({"url": "db"}))              // → db x3
let {name: who, "home town": town} = {"name": "Ana", "home town": "Rio"}
print(who, town)                           // → Ana Rio
```

In an object pattern, `key: name` takes the property `key` into a variable with another name.

## Null

```targascript
let config = {"theme": "dark"}
print(config.size ?? 14)                     // → 14
let user = {"address": null}
print(user.address?.city ?? "no city")       // → no city
config.size ??= 12                           // assigns only when null
```

`a?.b`, `a?.method()` and `a?.[i]` give `null` when `a` is null, skipping the rest of the chain.

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

```targascript
fn firstLine(lines) {
    print("opening")
    try {
        if lines.length == 0 { throw "empty" }
        return lines[0]
    } finally {
        print("closing")                     // runs on return and on errors
    }
}
print(firstLine(["a", "b"]))   // → opening, then closing, then a
```

`finally` runs after the `try` block and its `catch`, however they end: normally, with an error, or by `return`, `break` or `continue`. An error that no `catch` handled goes on after it. `catch` can be left out: `try { ... } finally { ... }`.

`throw` also works where a value goes, to fail instead of giving one: `let port = config.port ?? throw "no port set"`, or `_ => throw "unknown command"` in a `match`.

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

`await` waits for a promise and gives its result; on anything else it gives the value itself. Functions that await pause the whole program until the promise settles, so calling them needs no `await`. `sleep(ms)` and `fetch(url, options)` return promises; in the browser, so do many JavaScript functions. Builtins like `filter` and `sort` need the result of their function at once, so it can't await.

```
let response = await fetch("https://api.github.com/repos/migueltarga/TargaScript")
if !response.ok { throw "status ${response.status}" }
print(response.json().full_name)

let created = await fetch(url, {"method": "POST", "headers": {"X-Token": token}, "body": {"name": "x"}})
```

A response has `status`, `ok`, `url`, `headers`, `text` and `json()`. A `body` that isn't a string is sent as JSON.

`all(promises)` waits for several promises at once and gives their results in the same order, or fails as soon as one fails. Calling `fetch` without `await` starts the request and gives its promise, so requests can run side by side:

```
let repos = ["golang/go", "rust-lang/rust"]
let responses = await all(repos.map(r => fetch("https://api.github.com/repos/${r}")))
print(responses.map(r => r.json().stargazers_count))
```

A function given to `map` can await too. Then `map` gives its promise instead of waiting, so the calls run side by side:

```targascript
fn slowDouble(n) {
    await sleep(50)
    n * 2
}
print(await all([1, 2, 3].map(slowDouble)))   // → [2, 4, 6], in about 50 ms
```

## Loading files

```
load "lib/math.tg"            // its top-level names become available here
load "lib/text.tg" as text    // or are kept in an object: text.shout(...)
```

Paths are relative to the file with the `load`. A file runs once, however many times it's loaded. In web pages, `targascript.js` fetches the files from the same server.

From the command line, a file can also be loaded from a web address, so code can be shared with a link. The files it loads with relative paths come from the same place:

```
load "https://raw.githubusercontent.com/ana/tg-utils/main/text.tg" as text
```

It's downloaded each time the program runs. Load only code you trust: it runs with the same access to files as your program.

## Builtin functions

| Function | Does |
|---|---|
| `print(...values)` | prints values separated by spaces; it is a function like any other, so `x \|> print` works |
| `printf(format, ...values)` / `format(format, ...values)` | prints / returns values formatted with `%d`, `%s`, `%.2f`, `%5d`, `%-10s`, `%x`, `%v`...; `%,d` and `%,.2f` group the thousands |
| `type(value)` | `"integer"`, `"float"`, `"string"`, `"boolean"`, `"null"`, `"array"`, `"object"`, `"function"`, `"promise"` |
| `toString(v)`, `toInt(v)`, `toFloat(v)` | convert values |
| `abs`, `sqrt`, `pow`, `floor`, `ceil`, `trunc`, `sign`, `clamp(x, min, max)` | math |
| `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2(y, x)`, `hypot(x, y)`, `log`, `log10`, `log2`, `exp` | more math, with angles in radians; `PI` and `E` are constants |
| `round(x)`, `round(x, places)` | the nearest integer, or a float with that many decimal places: `round(3.14159, 2)` is `3.14` |
| `min(a, b, ...)`, `max(a, b, ...)`, `min(list)`, `max(list)` | the smallest or largest number |
| `random()`, `random(a, b)`, `random(list)` | a float from 0 to 1, an integer from `a` to `b`, or a random element |
| `range(start, end)`, `range(start, end, step)` | the integers from `start` to `end`, both included: `range(0, 10, 5)` is `[0, 5, 10]` |
| `now()` | the current Unix time in seconds |
| `timeFormat(seconds, layout)` | formats a time with `YYYY`, `YY`, `MMMM`, `MMM`, `MM`, `DD`, `dddd`, `ddd`, `HH`, `hh`, `mm`, `ss` and `A` (AM/PM), e.g. `timeFormat(now(), "DD/MM/YYYY HH:mm")` |
| `parseTime(text, layout)` | the opposite of `timeFormat`: `parseTime("16/10/2026", "DD/MM/YYYY")` is that day's Unix time |
| `clock()` | milliseconds since the program started, for timing code |
| `assert(condition, message)`, `assertEqual(actual, expected, message)` | fail with `assertion failed: ...` when the condition is falsy or the values differ; the message is optional |
| `escapeHTML(text)` | text safe to put in HTML: `<`, `>`, `&` and quotes become entities |
| `encodeURL(text)`, `decodeURL(text)` | text for a part of a URL, like a query value, and back |
| `bytes(size or text or list)` | binary data, see [Bytes](#bytes) |
| `parseJSON(text)`, `toJSON(value, indent)` | JSON, keeping the order of keys |
| `sleep(ms)`, `fetch(url, options)`, `all(promises)` | promises for `await` |
| `exists(path)`, `deleteFile(path)`, `makeDir(path)` | whether a file or directory exists; delete a file or an empty directory; make a directory and the ones it is in. Command line only |
| `exec(command)`, `exec(program, args)` | runs a command line in the shell, or a program with a list of arguments, and returns `{output, errors, code, ok}`: `exec("git", ["log", "-1"]).output`. Options come last: `{"dir": "src"}` runs it in another directory, `{"input": text}` gives it text to read, and `{"show": true}` shows its output as it comes instead of keeping it. A program that fails gives its exit code in `code`; one that can't start is an error. Command line only |
| `input`, `readFile`, `readBytes`, `writeFile`, `appendFile`, `listDir`, `env`, `exit`, `serve`, `response`, `staticFile` | [command line](#command-line) only |

## Testing

`test(name, fn)` runs a function as a test and prints whether it passed. A failing test shows the error and its line, and the rest of the file keeps running. The function can await, and the next test starts when it's done. `assert` and `assertEqual` are the usual way to fail.

```targascript
fn slugify(text) { text.trim().lower().replace(" ", "-") }

test("slugify joins words with dashes", fn() {
    assertEqual(slugify(" Hello World "), "hello-world")
})
// → ✓ slugify joins words with dashes
```

`describe(name, fn)` groups the tests that `fn` defines, and groups nest. The output is indented under the group's name, and `-run` matches the group's name too: `-run "parser numbers"`.

```targascript
describe("slugify", fn() {
    test("joins words", fn() { assertEqual(slugify("a b"), "a-b") })
    test("trims", fn() { assertEqual(slugify(" a "), "a") })
})
```

`targascript test` runs every file ending in `_test.tg` under the current directory, or the files and directories given, and prints a summary:

```text
math_test.tg
✓ adds
✗ divides
    assertion failed: expected 2, got 2.5 (line 9)

1 passed, 1 failed
```

It exits with 1 when a test fails, when a file has an error outside a test, or when there are no tests, so it can run in CI. `targascript test -run slug` runs only the tests whose name contains `slug`, ignoring case, and counts the others as skipped.

## Command line

```sh
targascript                       # the REPL: unfinished code continues on the next line; help, exit
targascript script.tg a b         # run a file; it sees ["a", "b"] in args
targascript --watch script.tg     # run it again whenever a .tg file next to it changes
targascript -e 'print(2 ** 10)'   # run code given on the command line
targascript test                  # run the *_test.tg files, see Testing
targascript test --watch          # and again whenever a .tg file changes
targascript check                 # find problems in the .tg files without running them
targascript build app.tg -o app   # make an executable that runs app.tg, see below
targascript lsp                   # a language server, for editors other than VS Code
targascript fmt                   # indent the .tg files; --check only lists the ones that need it
cat script.tg | targascript -     # run code from standard input
```

Scripts see their arguments in `args`, read what the user types with `input(prompt)`, and can use `readFile(path)`, `readBytes(path)`, `writeFile(path, text)`, `appendFile(path, text)`, `listDir(path)`, `env(name, default)` and `exit(code)`. Paths are relative to the working directory. `readBytes` gives a file as [bytes](#bytes), for files that aren't text, and `writeFile` and `appendFile` take bytes as well as text.

A script whose first line is `#!/usr/bin/env targascript` runs like any program on macOS and Linux, after `chmod +x script.tg`: `./script.tg a b`.

### Executables

`targascript build app.tg` makes an executable, `app`, with the program and every file it loads inside. It runs where TargaScript isn't installed, and every argument goes to the program's `args`:

```sh
targascript build tools/report.tg -o report
./report sales.csv
```

The executable is for the system it's built on. It's a copy of the `targascript` binary with the program added at its end.

### Web servers

`serve(port, handler)` starts an HTTP server and returns; the program then keeps running to answer requests. `handler` is called for each request and can `await`, and other requests are answered while it waits. The request is an object with `method`, `path`, `query` (an object), `headers` (with lowercase names) and `body` (a string). What the handler returns is the reply:

| Handler returns | Reply |
|---|---|
| a string | `200`, as text or HTML depending on what it looks like |
| an object, an array, a number... | `200`, as JSON |
| `null` | `204`, empty |
| `response(body, {status, headers})` | that body with that status and headers |
| an error | `500` with the message, which is also printed |

```text
let todos = []

serve(8080, fn(req) {
    match req.path {
        "/" => "<h1>Hello from TargaScript</h1>"
        "/todos" => if req.method == "POST" {
            todos.insert(parseJSON(req.body))
            response(todos.last(), {status: 201})
        } else {
            todos
        }
        _ => response("Not found", {status: 404})
    }
})
```

`staticFile(dir, path)` answers with a file from a directory, with its content type, or a 404. A directory gives its `index.html`. The path can't leave the directory, and hidden files like `.env` aren't served. A whole site with an API is a few lines:

```text
serve(8080, fn(req) {
    if req.path.startsWith("/api/") {
        return {time: timeFormat(now(), "HH:mm:ss")}
    }
    staticFile("public", req.path)
})
```

Text from a request that goes into HTML must pass through `escapeHTML`, so it can't add its own tags or scripts:

```text
serve(8080, fn(req) {
    let name = req.query.name ?? "world"
    "<h1>Hello, ${escapeHTML(name)}!</h1>"
})
```

A handler that awaits can call other services:

```text
serve(8080, fn(req) {
    let res = await fetch("https://api.github.com/repos/${req.query.repo}")
    let repo = res.json()
    return {name: repo.full_name, stars: repo.stargazers_count}
})
```

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
- **Conversions**: numbers, strings, booleans, `null` and arrays convert to TargaScript values, and `Uint8Array`, `Uint8ClampedArray` and `ArrayBuffer` to [bytes](#bytes), in one copy, as the pixels of `getImageData(...).data` do. Arrays and objects passed to JavaScript are copied, and bytes become a `Uint8Array`.
- **Functions**: TargaScript functions passed to JavaScript, like event handlers, can be called from JavaScript, and they return a promise if they await.
- **`print`**: it goes to the console.

Scripts can have a `src` and share their variables. In a worker created with `new Worker("targascript.js?src=worker.tg")`, `targascript.run(source, onOutput)` runs another program in a fresh environment and returns its error message, or null. `targascript.session()` makes an environment that keeps its variables between runs, as a REPL needs: its `run(source, onOutput)` works the same, and its `eval(source, onOutput)` returns the value of the code, or fails with the error message, so `try { show(session.eval(line, print)) } catch e { show(e.message) }` makes a REPL. JavaScript can call `targascriptRun(source, onOutput)` and `targascriptCheck(source)`.
