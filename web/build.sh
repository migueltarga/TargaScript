#!/bin/sh
# Builds targascript.wasm for the browser with TinyGo into the directory
# given (default: build), next to targascript.js.
#
# It targets WASI as a reactor with no scheduler: the module only needs a
# handful of WASI functions and the targascript imports from
# targascript.js. The 8MB stack fits the nesting limit set in cmd/wasm.
set -e
out=${1:-build}
mkdir -p "$out"
tinygo build -target wasip1 -buildmode=c-shared -scheduler=none -panic=trap \
  -opt=z -no-debug -ldflags="-extldflags=-zstack-size=8388608" \
  -o "$out/targascript.wasm" ./cmd/wasm
go run ./tools/wasmstrip "$out/targascript.wasm"
cp web/targascript.js "$out/"
