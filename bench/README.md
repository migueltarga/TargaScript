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
| callbacks | 0.157 | 0.142 | 0.061 | 0.064 |
| concat | 0.017 | 0.424 | 0.036 | 0.461 |
| fib | 0.076 | 0.112 | 0.032 | 0.040 |
| loop | 0.089 | 0.193 | 0.029 | 0.028 |
| methods | 0.053 | 0.093 | 0.029 | 0.025 |
| objects | 0.097 | 0.120 | 0.064 | 0.218 |
| properties | 0.072 | 0.137 | 0.043 | 0.064 |
| strings | 0.057 | 0.054 | 0.054 | 0.076 |
| words | 0.044 | 0.084 | 0.036 | 0.016 |

These are small programs, not a general ranking. `concat` particularly favors runtimes that optimize repeated string appends. The CLI build uses `cmd/targascript/default.pgo`, a profile trained on these workloads, so add representative applications before making broad performance claims.

## Profile-guided optimization

`cmd/targascript/default.pgo` is a CPU profile of these benchmarks, which Go uses to optimize the `targascript` binary. `./bench/pgo.sh` writes it again after VM changes. `TARGASCRIPT_CPUPROFILE=file targascript script.tg` profiles a program for `go tool pprof`.
