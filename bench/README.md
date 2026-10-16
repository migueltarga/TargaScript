# Benchmarks

Each benchmark is a small program written twice, in TargaScript (`.tg`) and in Python (`.py`). Both versions print the same result.

| Benchmark | What it does |
|---|---|
| `concat` | builds a 200,000-character string with `s += "x"` |
| `fib` | `fib(30)` the slow recursive way: 1.6 million function calls |
| `loop` | 3 million loop iterations with arithmetic |
| `objects` | creates 200,000 small objects and sorts them by a field |
| `strings` | builds 200,000 strings with interpolation, joins and splits them |
| `words` | counts words in an object, 500,000 times |

`./run.sh` builds TargaScript, checks that both versions agree, and prints the best of three runs of each, in seconds, including startup. On an Apple M-series laptop, with Python 3.12:

```
benchmark   TargaScript       Python
concat             0.02         0.54
fib                0.14         0.13
loop               0.18         0.21
objects            0.15         0.13
strings            0.06         0.06
words              0.07         0.09
```

TargaScript runs about as fast as CPython. That's enough for scripts, servers and web pages, but not for heavy number crunching.
