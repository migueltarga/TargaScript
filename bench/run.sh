#!/usr/bin/env bash
# Runs each benchmark with TargaScript and, when it's installed, Python, and
# prints the best of three runs in seconds. Both versions print the same
# result, which is checked.
set -e
cd "$(dirname "$0")"
go build -o /tmp/targascript-bench ../cmd/targascript

best() {
  local fastest=""
  for _ in 1 2 3; do
    local start end elapsed
    start=$(date +%s.%N)
    "$@" > /dev/null
    end=$(date +%s.%N)
    elapsed=$(echo "$end - $start" | bc)
    if [ -z "$fastest" ] || [ "$(echo "$elapsed < $fastest" | bc)" = 1 ]; then
      fastest=$elapsed
    fi
  done
  printf "%.2f" "$fastest"
}

printf "%-10s %12s %12s\n" "benchmark" "TargaScript" "Python"
for script in *.tg; do
  name=${script%.tg}
  python="-"
  if command -v python3 > /dev/null; then
    if [ "$(/tmp/targascript-bench "$script")" != "$(python3 "$name.py")" ]; then
      echo "$name: TargaScript and Python print different results" >&2
      exit 1
    fi
    python=$(best python3 "$name.py")
  fi
  printf "%-10s %12s %12s\n" "$name" "$(best /tmp/targascript-bench "$script")" "$python"
done
