# TargaScript Cookbook

Short, complete programs for everyday jobs. Each one runs as written. The examples marked with their output are run by the tests, so they stay correct. For the details of the language, see the [reference](REFERENCE.md).

- [Read a CSV file](#read-a-csv-file)
- [Count the words in a text](#count-the-words-in-a-text)
- [Reshape JSON](#reshape-json)
- [Do several things at once](#do-several-things-at-once)
- [Retry what can fail](#retry-what-can-fail)
- [Objects with methods](#objects-with-methods)
- [Test your code](#test-your-code)
- [Call a web API](#call-a-web-api)
- [Write a command-line tool](#write-a-command-line-tool)
- [Serve a website and an API](#serve-a-website-and-an-api)

## Read a CSV file

The first line has the names of the columns. `zip` pairs each value with its name, and `toObject` makes the pairs an object:

```targascript
writeFile("sales.csv", `region,product,units
North,Widget,10
South,Gadget,3
North,Gadget,7`)

fn readCSV(path) {
    let [header, ...rows] = readFile(path).trim().lines()
    let names = header.split(",")
    rows.map(row => names.zip(row.split(",")).toObject())
}

let sales = readCSV("sales.csv")
repeat region, rows in sales.groupBy(s => s.region) {
    print(region, rows.map(r => toInt(r.units)).sum())
}
```

```output
North 17
South 3
```

## Count the words in a text

```targascript
let text = "the cat and the hat and the bat"

let counts = text.lower().splitPattern("\W+")
    .groupBy(word => word)
    .entries()
    .map(([word, list]) => [word, list.length])
    .sortBy(([word, n]) => [-n, word])

repeat [word, n] in counts.slice(0, 3) {
    print(word.padEnd(5), "*" * n)
}
```

```output
the   ***
and   **
bat   *
```

## Reshape JSON

Patterns take what you need out of each item, with defaults for what's missing:

```targascript
let users = parseJSON(`[
  {"name": "Ana", "email": "ana@example.com", "role": "admin"},
  {"name": "Bia", "email": "bia@example.com"}
]`)

let summary = users.map(({name, email, role = "user"}) => {
    name,
    role,
    domain: email.split("@")[1],
})
print(toJSON(summary, 2))
```

```output
[
  {
    "name": "Ana",
    "role": "admin",
    "domain": "example.com"
  },
  {
    "name": "Bia",
    "role": "user",
    "domain": "example.com"
  }
]
```

## Do several things at once

A function given to `map` that awaits gives a promise, and `all` waits for all of them, so the waits overlap:

```targascript
fn slowSquare(n) {
    await sleep(100)
    n * n
}

let started = clock()
let squares = await all([1, 2, 3, 4].map(slowSquare))
print(squares, clock() - started < 300)
```

```output
[1, 4, 9, 16] true
```

## Retry what can fail

```targascript
let tries = 0
fn unreliable() {
    tries++
    if tries < 3 { throw "busy" }
    "done after ${tries} tries"
}

fn retry(task, times = 5) {
    repeat i in 1..times {
        let result = try { task() } catch e { null }
        if result != null { return result }
        await sleep(10 * i)
    }
    throw "gave up after ${times} tries"
}

print(retry(unreliable))
```

```output
done after 3 tries
```

## Objects with methods

Inside a method, `this` is the object. A function that returns an object works like a class:

```targascript
fn account(owner, balance = 0) {
    {
        owner,
        balance,
        deposit: fn(amount) {
            this.balance += amount
            this
        },
        withdraw: fn(amount) {
            if amount > this.balance { throw "not enough money" }
            this.balance -= amount
            this
        },
    }
}

let ana = account("Ana").deposit(100).withdraw(30)
print(ana.owner, ana.balance)
print(try { ana.withdraw(500) } catch e { e.message })
```

```output
Ana 70
not enough money
```

## Test your code

Put tests in files ending in `_test.tg` and run `targascript test`:

```targascript
fn slugify(text) {
    text.trim().lower().replacePattern("[^a-z0-9]+", "-")
}

test("joins words with dashes", fn() {
    assertEqual(slugify("Hello World"), "hello-world")
})

test("drops punctuation", fn() {
    assertEqual(slugify("Hi, there!"), "hi-there-")
})
```

```output
✓ joins words with dashes
✓ drops punctuation
```

## Call a web API

`fetch` returns a promise, and `json()` reads the body:

```
let response = await fetch("https://api.github.com/repos/golang/go")
if !response.ok { throw "GitHub answered ${response.status}" }

let {full_name, stargazers_count: stars} = response.json()
print("${full_name} has ${stars} stars")
```

To send data, give `fetch` a method and a body; an object is sent as JSON:

```
let created = await fetch("https://example.com/api/items", {
    method: "POST",
    headers: {Authorization: "Bearer ${env("TOKEN")}"},
    body: {name: "new item"},
})
```

## Write a command-line tool

`args` has the arguments, `input` asks for what's missing, and `exit` sets the exit code:

```
// grep.tg: print the lines of a file that match a pattern
if args.length < 2 {
    print("usage: grep.tg PATTERN FILE")
    exit(1)
}
let [pattern, file] = args
repeat i, line in readFile(file).lines() {
    if line.matches(pattern) { print("${i + 1}: ${line}") }
}
```

Run it with `targascript grep.tg "TODO" main.tg`, or make it a program anyone can run with `targascript build grep.tg`.

## Serve a website and an API

Files come from `public/`, and the API answers JSON:

```
let notes = []

serve(8080, fn(req) {
    match req.path {
        "/api/notes" if req.method == "POST" => {
            notes.insert(parseJSON(req.body))
            response(notes.last(), {status: 201})
        }
        "/api/notes" => notes
        _ => staticFile("public", req.path)
    }
})
```

`targascript server.tg` prints the address, and the program keeps running until you stop it.
