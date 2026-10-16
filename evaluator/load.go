package evaluator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/migueltarga/TargaScript/ast"
	"github.com/migueltarga/TargaScript/lexer"
	"github.com/migueltarga/TargaScript/object"
	"github.com/migueltarga/TargaScript/parser"
)

var (
	// Top-level environments of files already loaded, by absolute path
	loadedModules = map[string]*object.Environment{}
	// Files currently being loaded, to detect circular loads
	loadingModules = map[string]bool{}
	// Directory used to resolve loads from code with no file of its own,
	// such as the REPL. Empty means the working directory.
	mainDir string
)

// SetMainDir sets the directory used to resolve `load` paths in the main
// program, normally the directory of the script being run.
func SetMainDir(dir string) {
	mainDir = dir
}

// MainDir returns the directory set by SetMainDir.
func MainDir() string {
	return mainDir
}

// ResetModules forgets the files loaded so far, so they run again when
// loaded.
func ResetModules() {
	loadedModules = map[string]*object.Environment{}
	loadingModules = map[string]bool{}
}

// evalLoadStatement runs a file once and makes its top-level names
// available: in the current scope, or in a hash named by the alias.
func evalLoadStatement(node *ast.LoadStatement, env *object.Environment) object.Object {
	line, column := node.Token.Line, node.Token.Column

	path := node.Path.Value
	if !filepath.IsAbs(path) {
		dir := env.ModuleDir()
		if dir == "" {
			dir = mainDir
		}
		path = filepath.Join(dir, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return newError(line, column, "cannot load %q: %s", node.Path.Value, err)
	}

	module, errObj := loadModule(path, node.Path.Value, line, column)
	if errObj != nil {
		return errObj
	}

	if node.Alias != nil {
		hash := object.NewHash()
		for _, name := range module.LocalNames() {
			value, _ := module.Get(name)
			hash.Set(&object.String{Value: name}, value)
		}
		if env.IsLocalConstant(node.Alias.Value) {
			return newError(line, column, "cannot reassign constant: %s", node.Alias.Value)
		}
		env.SetLocal(node.Alias.Value, hash)
		return NULL
	}

	for _, name := range module.LocalNames() {
		if env.IsLocalConstant(name) {
			return newError(line, column, "cannot load %q: it defines %s, which is a constant here", node.Path.Value, name)
		}
	}
	for _, name := range module.LocalNames() {
		value, _ := module.Get(name)
		if module.IsLocalConstant(name) {
			env.SetConst(name, value)
		} else {
			env.SetLocal(name, value)
		}
	}
	return NULL
}

func loadModule(path, shownPath string, line, column int) (*object.Environment, *object.Error) {
	if module, ok := loadedModules[path]; ok {
		return module, nil
	}
	if loadingModules[path] {
		return nil, newError(line, column, "circular load of %q", shownPath)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, newError(line, column, "cannot load %q: %s", shownPath, readError(err))
	}

	p := parser.New(lexer.New(string(source)))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, newError(line, column, "cannot load %q: %s", shownPath, strings.Join(p.Errors(), "; "))
	}

	module := object.NewEnvironment()
	module.Dir = filepath.Dir(path)

	loadingModules[path] = true
	result := Eval(program, module)
	delete(loadingModules, path)

	if errObj, ok := result.(*object.Error); ok {
		return nil, newError(line, column, "%s:%d:%d: %s", shownPath, errObj.Line, errObj.Column, errObj.Message)
	}

	loadedModules[path] = module
	return module, nil
}

func readError(err error) string {
	if os.IsNotExist(err) {
		return "file not found"
	}
	return err.Error()
}
