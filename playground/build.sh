#!/bin/sh
# Builds the playground into the directory given (default: site), ready to
# serve as static files.
set -e
out=${1:-site}
./web/build.sh "$out"
cp playground/index.html playground/playground.tg playground/runner.tg playground/welcome.tg playground/social.png playground/CNAME "$out/"

# Demos: web apps written in TargaScript
mkdir -p "$out/demos"
cp -R playground/demos/. "$out/demos/"

# The tour, listed as file|title, the title coming from each lesson's first
# line: "// Lesson 1: Hello" becomes "Hello"
mkdir -p "$out/tour"
: > "$out/tour/index.txt"
for f in playground/tour/*.tg; do
  cp "$f" "$out/tour/"
  title=$(head -1 "$f" | sed 's/^\/\/ Lesson [0-9]*: //')
  echo "$(basename "$f")|$title" >> "$out/tour/index.txt"
done

# Examples that use load need other files, which the browser can't read
mkdir -p "$out/examples"
: > "$out/examples/index.txt"
for f in examples/*.tg; do
  grep -q '^load ' "$f" && continue
  cp "$f" "$out/examples/"
  basename "$f" >> "$out/examples/index.txt"
done

# A version for the files, from the content of the WebAssembly: pages load
# targascript.js?v=..., which fetches everything else with it, so a browser
# never mixes cached files from an older deploy with newer ones
version=$(shasum "$out/targascript.wasm" | cut -c1-12)
for page in "$out"/index.html "$out"/demos/*/index.html; do
  sed -i.bak "s|targascript\.js\"|targascript.js?v=$version\"|" "$page" && rm "$page.bak"
done
sed -i.bak "s|targascript\.js?src=|targascript.js?v=$version\&src=|" "$out/playground.tg" && rm "$out/playground.tg.bak"
