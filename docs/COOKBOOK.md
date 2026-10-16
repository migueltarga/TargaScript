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
- [Run other programs](#run-other-programs)
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

[▶ Try it in the playground](https://script.targa.me/#code=TI4xSwQxEIX7_IpHquRcgyCWa3Fgb2ehW8Td4TYQJyGZ4zzE_35kCcdWeZn3vZkXSSD0KxihZSXMXuB5QdPrTn970Uo1ek5nlopxi7mYLlSMdTXHIO9ehAob_fXxoK0CAHcq6ZyPV3NJZcH4ivZ2i1hKoGr698dnYz6bPyCGKpNt_G7gIvFJ1qnzNRU5Xu8R7vwjD9uRySpVKJMX3AkE7v1djWEm8zTg2eJvW5hLYNl6uuyXN17Mix2gDxoHsFX_6hYAAP__)

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

[▶ Try it in the playground](https://script.targa.me/#code=VI-xasQwEET7_YphKwmEISkPHC4pUyRFyuPgNjkVAq0sJBsSjP89SL7G3Qi9pxlFP2OpvlSMyFKqf__6_DC3CwErJ1HPJ_BrEnZgrxJiO0uSs_8VzdEPP5O2uzLFjspdQ-LNHfy3cPS_w9Hf6HqzRG1LXVSl_GHcVw0q2Zi1PeTQfYdWhRHcAN4sxhesBACd6qkhe7pPKiGddneoOYbZ8Jnt5enqaLOUS0izmaf-7Ue5w7O19B8AAP__)

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

[▶ Try it in the playground](https://script.targa.me/#code=RI4xDsIwEAR7v2LLMzIoIZTkFZSI4hSMFHFcjH0oBeLviBBIO9LO7EVRZBgP9wfnSOrxdADAI_eGIjEmqqvKT1CxgrqXcxINxThbPKNFJ0N3Jf-lk6egnQ0sQsc6YBvQBOxOmxsnWoLeu5R7NZp34SfD-h_Yo_kceAcAAP__)

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

[▶ Try it in the playground](https://script.targa.me/#code=TJDdagMhEIXvfYrTpRfalJBQetd9GJPONhJrlnFsWGTfvfgTNl6J5xvmO3oSCDuKGHFQU0AKTN7ZkydtkBWAlu929e6mjn_hAxly4dsdwynFZcBaieH7Fgh2EmK85gqvbWZQqyormIQXLTZe3yHut-7-fCxjmskKHFzAcb9veYvKKb5MMXnBCOGlONh41QYrzlbOFxAyQvK-63TpPvMytiwXicTh8b6x9m6dIHqiWR8PeIMzNWtE7_tj_whp3loWy-eWM7sguhXdftQY9R8AAP__)

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

[▶ Try it in the playground](https://script.targa.me/#code=jFDRSgQxDHzvVwz7tEUtFfFFOMFPib3utXCbHt0cy7H036Ved08RxTyUJpnMJDMwyLl0ZunTzD7f452OxM5jB6uxKADtrXHFbGnD3gp7f0pTlBcM3NNYafWX6RoS4mRWjbsdrqgfkK1QbuRzlLDPNP_BHodGiNfvSgsk5DSj4yTwnM6HgDGxv3Qov-_38O_9iipKHb2AmLDbPO3emDptmi39o7XarGf0T1arU44sPTGZZn_9Nvm1K_mC5bOxjT5bq1HgSFxAvc6b0U8THTyKVh8BAAD__w)

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

[▶ Try it in the playground](https://script.targa.me/#code=jI69TgUhEEZ7nmKcCiJsbH0AE0s7C6MJucwKBmGdGYI_ue9urpvt7b7TnPOtDaSO17J-WaVPdfBjAAAue1Eu79YttU9i6xamrcYTPURV4mbx6SWG75tw-3yNHjCgM2djlEQtvvXSBGbnJDCLZkhRMgl6WJs9GlGEWO8-Rqz2-ID3VGuHx841ofOA-cJh7mzO7igk7pvANtpJR9TS23_cxYNmYrra1SX8UdjFvwEAAP__)

## Call a web API

`fetch` returns a promise, and `json()` reads the body:

```
let response = await fetch("https://api.github.com/repos/golang/go")
if !response.ok { throw "GitHub answered ${response.status}" }

let {full_name, stargazers_count: stars} = response.json()
print("${full_name} has ${stars} stars")
```

[▶ Try it in the playground](https://script.targa.me/#code=RI7BasMwEETv-oqt8MGGYt0DObd_ETbuWlKraIV2haFC_14a2uQ0MMwbXiKFSlI4C8EZ8MCosJNuYbZBtcjJOSxx9VFDu64b31ylwuI8J8zeebaLiTu8_J-s_AUdNFQ-wL5FfW9XwCwHVfqAqT9moqhNhoVhzK9E31tKl4w3egVRrB6_qcpl45b1dG9kwPnhun4K53kxpcass52e-ICAAlP_Q-5hF_MTAAD__w)

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

## Run other programs

`exec` runs a command line in the shell, or a program with a list of arguments, and returns its `output`, `errors`, exit `code` and whether it went `ok`:

```targascript
// Runs a command line and gives its output, or fails with its errors
fn run(command) {
    let result = exec(command)
    if !result.ok { throw "${command} failed: ${result.errors.trim()}" }
    result.output.trim()
}

writeFile("notes.txt", "milk\neggs\nbread\n")
print(run("sort notes.txt").lines())
print(run("wc -l < notes.txt"))
print(try { run("cat shopping.txt") } catch e { e.message })

// A list of arguments goes to the program as is, spaces and all
writeFile("my notes.txt", "hi")
print(exec("cat", ["my notes.txt"]).output)
```

```output
["bread", "eggs", "milk"]
3
cat shopping.txt failed: cat: shopping.txt: No such file or directory
hi
```

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
