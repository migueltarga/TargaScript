package compiler

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/code"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
	"github.com/migueltarga/TargaScript/token"
)

// moduleName is a top-level name defined by a loaded file.
type moduleName struct {
	name    string
	isConst bool
}

// ResolveLoadPath returns the absolute path of a file loaded from code in
// dir, or the working directory when dir is empty.
func ResolveLoadPath(dir, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	return filepath.Abs(path)
}

// ReadModule reads and parses a loaded file, returning the error message the
// load fails with when it can't.
func ReadModule(path, shown string) (*ast.Program, string) {
	source, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "cannot load " + quote(shown) + ": file not found"
		}
		return nil, "cannot load " + quote(shown) + ": " + err.Error()
	}

	p := parser.New(lexer.New(string(source)))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, "cannot load " + quote(shown) + ": " + strings.Join(p.Errors(), "; ")
	}
	return program, ""
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// moduleNames returns the names a file defines at its top level, including
// those it loads without an alias, sorted. seen guards against circular
// loads, which fail when the files run.
func moduleNames(program *ast.Program, dir string, seen map[string]bool) []moduleName {
	byName := map[string]bool{}
	for _, stmt := range program.Statements {
		switch stmt := stmt.(type) {
		case *ast.LetStatement:
			byName[stmt.Name.Value] = false
		case *ast.ConstStatement:
			byName[stmt.Name.Value] = true
		case *ast.FunctionStatement:
			byName[stmt.Name.Value] = false
		case *ast.LoadStatement:
			if stmt.Alias != nil {
				byName[stmt.Alias.Value] = false
				continue
			}
			path, err := ResolveLoadPath(dir, stmt.Path.Value)
			if err != nil || seen[path] {
				continue
			}
			seen[path] = true
			loaded, message := ReadModule(path, stmt.Path.Value)
			if message != "" {
				continue
			}
			for _, n := range moduleNames(loaded, filepath.Dir(path), seen) {
				byName[n.name] = n.isConst
			}
		}
	}

	names := make([]moduleName, 0, len(byName))
	for name, isConst := range byName {
		names = append(names, moduleName{name, isConst})
	}
	sort.Slice(names, func(i, j int) bool { return names[i].name < names[j].name })
	return names
}

// loadedNames returns the names a load statement defines, or the error
// message it fails with.
func (c *Compiler) loadedNames(stmt *ast.LoadStatement) (string, []moduleName, string) {
	shown := stmt.Path.Value
	path, err := ResolveLoadPath(c.Dir, shown)
	if err != nil {
		return "", nil, "cannot load " + quote(shown) + ": " + err.Error()
	}
	program, message := ReadModule(path, shown)
	if message != "" {
		return path, nil, message
	}
	return path, moduleNames(program, filepath.Dir(path), map[string]bool{path: true}), ""
}

func (c *Compiler) compileLoad(stmt *ast.LoadStatement) error {
	path, names, message := c.loadedNames(stmt)
	if message != "" {
		c.emitError(stmt.Token, "%s", message)
		return nil
	}

	c.emit(stmt.Token, code.OpLoad,
		c.addConstant(&object.String{Value: path}),
		c.addConstant(&object.String{Value: stmt.Path.Value}))

	if stmt.Alias != nil {
		name := stmt.Alias.Value
		if existing, ok := c.lookupCurrentScope(name); ok && existing.Const {
			c.emitError(stmt.Token, "cannot reassign constant: %s", name)
			return nil
		}
		c.emitDefine(c.define(name, false))
		c.emit(token.Token{}, code.OpPop)
		c.emit(token.Token{}, code.OpNull)
		return nil
	}

	for _, n := range names {
		if existing, ok := c.lookupCurrentScope(n.name); ok && existing.Const {
			c.emitError(stmt.Token, "cannot load %s: it defines %s, which is a constant here", quote(stmt.Path.Value), n.name)
			return nil
		}
	}
	for _, n := range names {
		c.emit(token.Token{}, code.OpDup)
		c.emit(token.Token{}, code.OpGetProperty, c.addConstant(&object.String{Value: n.name}))
		c.emitDefine(c.define(n.name, n.isConst))
		c.emit(token.Token{}, code.OpPop)
	}
	c.emit(token.Token{}, code.OpPop)
	c.emit(token.Token{}, code.OpNull)
	return nil
}
