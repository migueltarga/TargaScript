# Benchmarks

Each workload has equivalent TargaScript, Python, JavaScript and Lua programs. The runner checks that all available versions print the same values before timing them.

Run `./bench/run.sh` from the repository root. It builds TargaScript once, then reports the median of five runs for each language. Times include process startup. Python 3 is required for the runner; Node and Lua are optional. `./bench/run.sh --runs 1` checks outputs quickly.

| Workload | What it does |
|---|---|
| `callbacks` | Maps, filters and reduces 300,000 numbers five times |
| `concat` | Appends one character 200,000 times |
| `fib` | Computes recursive `fib(30)` |
| `loop` | Runs 3 million iterations with arithmetic |
| `methods` | Calls an object method one million times |
| `objects` | Creates and sorts 200,000 small objects |
| `properties` | Reads properties of 100,000 objects ten times |
| `strings` | Builds, joins and splits 200,000 strings |
| `words` | Counts words in an object 500,000 times |

Example result on an Apple M3 Max, October 2026, with Python 3.14.3, Node 22.23.1 and Lua 5.4.7 (seconds):

| Workload | TargaScript | Python | Node | Lua |
|---|---:|---:|---:|---:|
| callbacks | 0.112 | 0.141 | 0.063 | 0.065 |
| concat | 0.014 | 0.419 | 0.036 | 0.452 |
| fib | 0.058 | 0.109 | 0.033 | 0.040 |
| loop | 0.048 | 0.185 | 0.029 | 0.027 |
| methods | 0.039 | 0.093 | 0.032 | 0.026 |
| objects | 0.073 | 0.115 | 0.061 | 0.200 |
| properties | 0.052 | 0.136 | 0.041 | 0.063 |
| strings | 0.034 | 0.048 | 0.045 | 0.066 |
| words | 0.026 | 0.079 | 0.036 | 0.014 |

These are small programs, not a general ranking. `concat` particularly favors runtimes that optimize repeated string appends. The CLI build uses `cmd/targascript/default.pgo`, a profile trained on these workloads, so add representative applications before making broad performance claims.

## Profile-guided optimization

`cmd/targascript/default.pgo` is a CPU profile of these benchmarks, which Go uses to optimize the `targascript` binary. `./bench/pgo.sh` writes it again after VM changes. `TARGASCRIPT_CPUPROFILE=file targascript script.tg` profiles a program for `go tool pprof`.
