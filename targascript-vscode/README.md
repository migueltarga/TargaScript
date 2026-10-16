# TargaScript Language Support for VS Code

This extension provides syntax highlighting and error checking for TargaScript (.tg) files in Visual Studio Code.

## Features

- Errors while you edit: syntax errors, names that aren't defined (with "Did you mean...?" for typos), constants assigned again, `break` outside a loop. They're found by the TargaScript runtime itself, compiled to WebAssembly and included in the extension, so nothing else needs to be installed
- Syntax highlighting for TargaScript files
- Recognition of language keywords, operators, and built-in functions
- Support for TargaScript's specific syntax constructs

## TargaScript

TargaScript is a programming language designed by Miguel Targa. For more information about the language, visit the [TargaScript GitHub repository](https://github.com/migueltarga/TargaScript).

You can also try it without installing anything in the [online playground](https://migueltarga.github.io/TargaScript/).

## Example

Here's how TargaScript code looks with this syntax highlighting:

```targascript
// Fibonacci function in TargaScript
fn fibonacci(n) {
  if (n <= 1) {
    return n
  }
  return fibonacci(n-1) + fibonacci(n-2)
}

// Print first 10 Fibonacci numbers
let i = 0
repeat i < 10 {
  print(fibonacci(i))
  i = i + 1
}
```

## Installation

1. Launch VS Code
2. Go to Extensions view (`Ctrl+Shift+X` or `Cmd+Shift+X`)
3. Search for "TargaScript"
4. Click Install

## Manual Installation

If you prefer to install the extension manually:

1. Download the latest release from the GitHub repository
2. Unzip the downloaded file
3. Copy the folder to your VS Code extensions folder:
   - Windows: `%USERPROFILE%\.vscode\extensions`
   - macOS/Linux: `~/.vscode/extensions`
4. Restart VS Code

## Issues and Contributions

For issues, feature requests, or contributions, please visit the [extension's GitHub repository](https://github.com/migueltarga/TargaScript).

## License

This extension is licensed under the [MIT License](LICENSE). 
## Building

The error checking needs the TargaScript runtime in `runtime/`. `npm run vscode:prepublish` (or `sh build.sh`) builds it with [TinyGo](https://tinygo.org), and `vsce package` runs it automatically.
