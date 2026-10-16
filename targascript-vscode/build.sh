#!/bin/sh
# Copies the TargaScript runtime into runtime/, where extension.js loads it,
# and the docs of the builtins, which the language server shares. Run it
# before packaging the extension; it needs TinyGo, like web/build.sh.
set -e
cd "$(dirname "$0")/.."
./web/build.sh targascript-vscode/runtime
cp lsp/builtins.json targascript-vscode/builtins.json
