# Change Log

All notable changes to the "TargaScript" extension will be documented in this file.

## [0.5.0] - 2026-10-16

- Show errors while editing: syntax errors, names that aren't defined (with suggestions for typos), constants assigned again and more, checked by the TargaScript runtime itself

## [0.4.0] - 2026-10-16

- Highlight the `await`, `try`, `catch` and `throw` keywords
- Highlight the `??` and `?.` operators

## [0.3.0] - 2026-10-15

- Highlight `/* */` block comments and toggle them with the block comment command
- Highlight numbers with an exponent, like `1e3` and `2.5E-4`

## [0.2.0] - 2026-10-15

- Highlight `${...}` interpolation inside strings
- Highlight `++`, `--` and `..` operators
- Fix `<=`, `>=` and `!=` being highlighted as two separate operators
- Fix floats like `3.14` being highlighted as two integers
- Remove the `/* */` block comment setting and `'` auto-closing, which the language doesn't support

## [0.1.0] - 2025-01-04

- Initial release
- Basic syntax highlighting support for TargaScript files
- Recognition of TargaScript keywords, operators, functions, and literals 