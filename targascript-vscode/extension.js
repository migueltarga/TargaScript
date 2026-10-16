// Shows the errors in TargaScript files while they are edited, and the docs
// of builtins on hover and in completions. The checks run in the
// TargaScript runtime compiled to WebAssembly, in runtime/, so nothing else
// needs to be installed.
const vscode = require("vscode");
const fs = require("fs");
const path = require("path");

const DELAY = 300; // milliseconds to wait after a change before checking

// Signature and description of each builtin, copied from lsp/ by build.sh;
// a Go test keeps it complete
const BUILTINS = require("./builtins.json");

const KEYWORDS = ["let", "const", "fn", "if", "else", "return", "repeat", "in", "break",
  "continue", "load", "as", "await", "try", "catch", "throw", "match", "true", "false", "null", "this"];

const builtinDocs = (name) => {
  const [signature, description] = BUILTINS[name];
  return new vscode.MarkdownString().appendCodeblock(signature, "targascript").appendMarkdown(description);
};

const hover = {
  provideHover(document, position) {
    const range = document.getWordRangeAtPosition(position);
    const word = range && document.getText(range);
    if (!word || !Object.hasOwn(BUILTINS, word)) return null;
    // A method or property with the same name isn't the builtin
    if (range.start.character > 0 && document.lineAt(position.line).text[range.start.character - 1] === ".") return null;
    return new vscode.Hover(builtinDocs(word), range);
  },
};

const completions = {
  provideCompletionItems() {
    const items = Object.keys(BUILTINS).map((name) => {
      const item = new vscode.CompletionItem(name, vscode.CompletionItemKind.Function);
      item.detail = BUILTINS[name][0];
      item.documentation = builtinDocs(name);
      item.insertText = new vscode.SnippetString(`${name}($0)`);
      return item;
    });
    for (const keyword of KEYWORDS) {
      items.push(new vscode.CompletionItem(keyword, vscode.CompletionItemKind.Keyword));
    }
    return items;
  },
};

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

  // Runs the file, or its tests, with the targascript command in a terminal
  const runIn = (command) => async () => {
    const editor = vscode.window.activeTextEditor;
    if (!editor || editor.document.languageId !== "targascript") return;
    await editor.document.save();
    const program = vscode.workspace.getConfiguration("targascript").get("path") || "targascript";
    const terminal = vscode.window.terminals.find((t) => t.name === "TargaScript") ||
      vscode.window.createTerminal("TargaScript");
    terminal.show(true);
    const file = JSON.stringify(editor.document.fileName);
    terminal.sendText(command ? `${program} ${command} ${file}` : `${program} ${file}`);
  };

  context.subscriptions.push(
    vscode.commands.registerCommand("targascript.runFile", runIn("")),
    vscode.commands.registerCommand("targascript.testFile", runIn("test")),
    vscode.languages.registerHoverProvider("targascript", hover),
    vscode.languages.registerCompletionItemProvider("targascript", completions),
    vscode.workspace.onDidOpenTextDocument(check),
    vscode.workspace.onDidChangeTextDocument((event) => checkLater(event.document)),
    vscode.workspace.onDidCloseTextDocument((document) => diagnostics.delete(document.uri)),
  );
  vscode.workspace.textDocuments.forEach(check);
}

function deactivate() {}

module.exports = { activate, deactivate };
