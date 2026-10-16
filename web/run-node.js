// Runs a TargaScript file with targascript.wasm in Node and prints its
// output, like the CLI does. CI uses it to check that the WebAssembly build
// behaves the same.
//
// usage: node web/run-node.js path/to/targascript.wasm file.tg
const fs = require("fs");
const path = require("path");
const load = require(path.join(__dirname, "targascript.js"));

const [wasmPath, file] = process.argv.slice(2);
load(fs.readFileSync(wasmPath)).then((run) => {
  let output = "";
  const error = run(fs.readFileSync(file, "utf8"), (text) => { output += text; });
  process.stdout.write(output);
  if (error) {
    process.stderr.write(`${error}\n`);
    process.exitCode = 1;
  }
});
