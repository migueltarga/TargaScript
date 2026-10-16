// Shows the errors in TargaScript files while they are edited. The checks
// run in the TargaScript runtime compiled to WebAssembly, in runtime/, so
// nothing else needs to be installed.
const vscode = require("vscode");
const fs = require("fs");
const path = require("path");

const DELAY = 300; // milliseconds to wait after a change before checking

async function activate(context) {
  const runtime = path.join(__dirname, "runtime");
  const load = require(path.join(runtime, "targascript.js"));
  const run = await load(fs.readFileSync(path.join(runtime, "targascript.wasm")));

  const diagnostics = vscode.languages.createDiagnosticCollection("targascript");
  context.subscriptions.push(diagnostics);

  const check = (document) => {
    if (document.languageId !== "targascript") return;
    const problems = run.check(document.getText());
    diagnostics.set(document.uri, problems.map((problem) => {
      const position = new vscode.Position(Math.max(problem.line - 1, 0), Math.max(problem.column - 1, 0));
      const range = document.getWordRangeAtPosition(position) || new vscode.Range(position, position.translate(0, 1));
      const diagnostic = new vscode.Diagnostic(range, problem.message, vscode.DiagnosticSeverity.Error);
      diagnostic.source = "TargaScript";
      return diagnostic;
    }));
  };

  const timers = new Map();
  const checkLater = (document) => {
    clearTimeout(timers.get(document.uri.toString()));
    timers.set(document.uri.toString(), setTimeout(() => check(document), DELAY));
  };

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument(check),
    vscode.workspace.onDidChangeTextDocument((event) => checkLater(event.document)),
    vscode.workspace.onDidCloseTextDocument((document) => diagnostics.delete(document.uri)),
  );
  vscode.workspace.textDocuments.forEach(check);
}

function deactivate() {}

module.exports = { activate, deactivate };
