# Coming from JavaScript or Python

TargaScript will feel familiar. These tables show how to write what you already know. When you type the other language's way by habit, the error usually says what to write instead.

- [From JavaScript](#from-javascript)
- [From Python](#from-python)

## From JavaScript

| JavaScript | TargaScript |
|---|---|
| `let x = 1`, `const y = 2` | the same |
| `function add(a, b) { return a + b }` | `fn add(a, b) { a + b }`; the last value is returned |
| `(a, b) => a + b` | the same |
| `` `Hello ${name}` `` | `"Hello ${name}"`, or the same in backticks |
| `for (const x of list) { }` | `repeat x in list { }` |
| `for (let i = 0; i < 10; i++) { }` | `repeat i in 0..9 { }` |
| `while (cond) { }` | `repeat cond { }` |
| `list.forEach((x, i) => ...)` | `repeat i, x in list { }` |
| `===`, `!==` | `==`, `!=`, which never convert types |
| `a > 1 ? "big" : "small"` | `if a > 1 { "big" } else { "small" }` |
| `switch (x) { case 1: ... }` | `match x { 1 => ..., _ => ... }` |
| `list.push(x)` | `list.insert(x)` |
| `list.includes(x)`, `"k" in obj` | `x in list`, `"k" in obj` |
| `[...a, ...b]`, `{...a, b: 1}` | the same |
| `const {name, age = 0} = user` | `let {name, age = 0} = user` |
| `Object.keys(o)`, `Object.entries(o)` | `o.keys()`, `o.entries()` |
| `Object.fromEntries(pairs)` | `pairs.toObject()` |
| `JSON.parse(s)`, `JSON.stringify(v)` | `parseJSON(s)`, `toJSON(v)` |
| `console.log(a, b)` | `print(a, b)` |
| `x?.y ?? d` | the same, and `list?.[0]` |
| `await fetch(url)` | the same; no `async` needed |
| `Promise.all(list.map(f))` | `all(list.map(f))` |
| `class Point { ... }` | a function returning an object; inside methods, `this` works |
| `import { f } from "./lib.js"` | `load "lib.tg"`, or `load "lib.tg" as lib` |
| `try { } catch (e) { e.message }` | `try { } catch e { e.message }` |
| `Math.max(...list)` | `max(list)` |
| `x.toFixed(2)` | `"${x:.2f}"` |
| `typeof x` | `type(x)` |

TargaScript adds:

- the pipe `x |> f`;
- `match` with ranges and guards;
- `1_000_000`;
- `"-" * 20`;
- `list[-1]` for the last element;
- tests with `test(...)`;
- `serve(...)` for web servers.

## From Python

| Python | TargaScript |
|---|---|
| `x = 1` | `let x = 1`, or `const` |
| `def add(a, b): return a + b` | `fn add(a, b) { a + b }`; blocks are in braces |
| `lambda x: x * 2` | `x => x * 2` |
| `f"Hello {name}"`, `f"{x:.2f}"`, `f"{n:,}"` | `"Hello ${name}"`, `"${x:.2f}"`, `"${n:,}"` |
| `for x in items:` | `repeat x in items { }` |
| `for i in range(10):` | `repeat i in 0..9 { }`; ranges include their end |
| `for i, x in enumerate(items):` | `repeat i, x in items { }` |
| `for k, v in d.items():` | `repeat k, v in d { }` |
| `while cond:` | `repeat cond { }` |
| `"big" if a > 1 else "small"` | `if a > 1 { "big" } else { "small" }` |
| `None`, `True`, `False` | `null`, `true`, `false` |
| `len(x)` | `x.length` |
| `items.append(x)` | `items.insert(x)` |
| `[f(x) for x in xs if p(x)]` | `[f(x) repeat x in xs if p(x)]`, or `xs.filter(p).map(f)` |
| `", ".join(words)` | `words.join(", ")` |
| `sorted(xs, key=lambda p: p.age)` | `xs.sortBy(p => p.age)` |
| `x in items`, `"k" in d` | the same |
| `d.get("k", 0)` | `d.k ?? 0`, or `d.get("k", 0)` |
| `{**a, "b": 1}` | `{...a, b: 1}` |
| `a, *rest = items` | `let [a, ...rest] = items` |
| `match x: case 1: ...` | `match x { 1 => ... }` |
| `try: ... except Exception as e:` | `try { ... } catch e { e.message }` |
| `raise ValueError("bad")` | `throw "bad"` |
| `import json; json.loads(s)` | `parseJSON(s)` |
| `open(path).read()` | `readFile(path)` |
| `input("Name? ")` | the same |
| `assert x == 1` | `assert(x == 1)`, or `assertEqual(x, 1)` in tests |
| `import mod` | `load "mod.tg" as mod` |
| `asyncio.run(main())` | not needed: `await` works at the top level |

TargaScript adds:

- the pipe `x |> f`;
- `?.` and `??` for values that may be null;
- `serve(...)` for web servers;
- the same code running in web pages.
