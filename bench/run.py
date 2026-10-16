#!/usr/bin/env python3
"""Compare equivalent CLI programs, including process startup."""

import argparse
import os
from pathlib import Path
import platform
import shutil
import statistics
import subprocess
import tempfile
import time


HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
RUNTIMES = {"Python": ("python3", ".py"), "Node": ("node", ".js"), "Lua": ("lua", ".lua")}


def run(command):
    result = subprocess.run(command, cwd=HERE, capture_output=True, text=True, check=True)
    return result.stdout.split()


def measure(command, repetitions):
    elapsed = []
    for _ in range(repetitions):
        start = time.perf_counter()
        run(command)
        elapsed.append(time.perf_counter() - start)
    return statistics.median(elapsed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--runs", type=int, default=5, help="runs per program (default: 5)")
    args = parser.parse_args()
    if args.runs < 1:
        parser.error("--runs must be positive")

    with tempfile.TemporaryDirectory(prefix="targascript-bench-") as temp:
        binary = Path(temp) / ("targascript.exe" if os.name == "nt" else "targascript")
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/targascript"], cwd=ROOT, check=True)
        available = {name: shutil.which(exe) for name, (exe, _) in RUNTIMES.items()}
        print(f"{platform.platform()} | {args.runs} runs | median wall time, startup included")
        print("Runtimes: TargaScript (built from this checkout)", end="")
        for name, path in available.items():
            if path:
                version = subprocess.run([path, "--version" if name != "Lua" else "-v"], capture_output=True, text=True)
                print(f", {name} {(version.stdout or version.stderr).strip().splitlines()[0]}", end="")
        print("\n")
        print(f"{'benchmark':<12} {'TargaScript':>12} {'Python':>12} {'Node':>12} {'Lua':>12}")
        for source in sorted(HERE.glob("*.tg")):
            commands = {"TargaScript": [str(binary), str(source)]}
            for name, (_, extension) in RUNTIMES.items():
                counterpart = source.with_suffix(extension)
                if available[name] and counterpart.is_file():
                    commands[name] = [available[name], str(counterpart)]
            expected = run(commands["TargaScript"])
            for name, command in commands.items():
                actual = run(command)
                if actual != expected:
                    raise SystemExit(f"{source.name}: {name} printed {actual!r}, expected {expected!r}")
            times = {name: measure(command, args.runs) for name, command in commands.items()}
            print(f"{source.stem:<12} " + " ".join(f"{times[name]:>11.3f}s" if name in times else f"{'-':>12}" for name in ("TargaScript", "Python", "Node", "Lua")), flush=True)


if __name__ == "__main__":
    main()
