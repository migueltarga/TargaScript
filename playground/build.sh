#!/bin/sh
# Builds the playground into the directory given (default: site), ready to
# serve as static files.
set -e
out=${1:-site}
./web/build.sh "$out"
cp playground/index.html playground/playground.tg playground/runner.tg playground/welcome.tg "$out/"

# Examples that use load need other files, which the browser can't read
mkdir -p "$out/examples"
: > "$out/examples/index.txt"
for f in examples/*.tg; do
  grep -q '^load ' "$f" && continue
  cp "$f" "$out/examples/"
  basename "$f" >> "$out/examples/index.txt"
done
