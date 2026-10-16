# Benchmarks

Each benchmark is a small program written twice, in TargaScript (`.tg`) and in Python (`.py`). Both versions print the same result.

| Benchmark | What it does |
|---|---|
| `callbacks` | `map`, `filter` and `reduce` with arrow functions over 300,000 numbers, 5 times |
| `concat` | builds a 200,000-character string with `s += "x"` |
| `fib` | `fib(30)` the slow recursive way: 1.6 million function calls |
| `loop` | 3 million loop iterations with arithmetic |
| `methods` | a million calls to a method that updates its object through `this` |
| `objects` | creates 200,000 small objects and sorts them by a field |
| `properties` | reads two properties of 100,000 objects, 10 times |
| `strings` | builds 200,000 strings with interpolation, joins and splits them |
| `words` | counts words in an object, 500,000 times |

`./run.sh` builds TargaScript, checks that both versions agree, and prints the best of three runs of each, in seconds, including startup. On an Apple M-series laptop, with Python 3.12:

```
benchmark   TargaScript       Python
callbacks          0.18         0.24
concat             0.02         0.56
fib                0.10         0.14
loop               0.11         0.21
methods            0.09         0.12
objects            0.10         0.14
properties         0.09         0.14
strings            0.06         0.07
words              0.05         0.09
```

TargaScript is faster than CPython on all of them, from 1.2 times on `strings` to 2 times on `loop` (and 28 times on `concat`, which CPython makes a new string for every time). That's plenty for scripts, servers and web pages, though not for heavy number crunching.
