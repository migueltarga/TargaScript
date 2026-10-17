// Package wasmcheck checks the Go code of the WebAssembly build for what
// makes TinyGo pull reflection into it, which adds kilobytes:
//
//   - == and != between interfaces, and switch on an interface, which TinyGo
//     compiles to a comparison with reflection
//   - map keys other than strings that aren't plain numbers, bools or typed
//     pointers, such as interfaces, structs with strings or unsafe.Pointer,
//     which TinyGo hashes with reflection
//   - imports of reflect, fmt and sort, which use it
//
// Compare objects with object.Same and key maps by object.IdentityOf or
// ast.IdentityOf instead.
package wasmcheck

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/migueltarga/TargaScript"

// forbidden are the packages the WebAssembly build doesn't import
var forbidden = map[string]bool{"reflect": true, "fmt": true, "sort": true}

// checker type-checks the packages of the module as the WebAssembly build
// sees them, with the tinygo tag on wasip1.
type checker struct {
	t        *testing.T
	root     string
	fset     *token.FileSet
	context  build.Context
	std      types.Importer
	packages map[string]*types.Package
	found    []string
}

func (c *checker) Import(path string) (*types.Package, error) {
	if !strings.HasPrefix(path, module+"/") {
		return c.std.Import(path)
	}
	if p, ok := c.packages[path]; ok {
		return p, nil
	}
	return c.check(path), nil
}

func (c *checker) report(pos token.Pos, what string) {
	p := c.fset.Position(pos)
	rel, _ := filepath.Rel(c.root, p.Filename)
	c.found = append(c.found, rel+":"+itoa(p.Line)+": "+what)
}

func (c *checker) check(path string) *types.Package {
	dir := filepath.Join(c.root, strings.TrimPrefix(path, module+"/"))
	pkg, err := c.context.ImportDir(dir, 0)
	if err != nil {
		c.t.Fatal(err)
	}
	files := []*ast.File{}
	for _, name := range pkg.GoFiles {
		f, err := parser.ParseFile(c.fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			c.t.Fatal(err)
		}
		files = append(files, f)
		for _, spec := range f.Imports {
			if imported := strings.Trim(spec.Path.Value, `"`); forbidden[imported] {
				c.report(spec.Pos(), "imports "+imported)
			}
		}
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
	conf := types.Config{Importer: c}
	checked, err := conf.Check(path, c.fset, files, info)
	if err != nil {
		c.t.Fatalf("%s: %v", path, err)
	}
	c.packages[path] = checked

	isInterface := func(e ast.Expr) bool {
		tv := info.Types[e]
		return tv.Type != nil && !tv.IsNil() && types.IsInterface(tv.Type)
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BinaryExpr:
				if (n.Op == token.EQL || n.Op == token.NEQ) && !info.Types[n.X].IsNil() && !info.Types[n.Y].IsNil() &&
					(isInterface(n.X) || isInterface(n.Y)) {
					c.report(n.Pos(), "compares interfaces: "+types.ExprString(n))
				}
			case *ast.SwitchStmt:
				if n.Tag != nil && isInterface(n.Tag) {
					c.report(n.Pos(), "switches on an interface: "+types.ExprString(n.Tag))
				}
			case *ast.MapType:
				if key := info.Types[n.Key].Type; key != nil && !plainKey(key, true) {
					c.report(n.Pos(), "map key hashed with reflection: "+types.ExprString(n.Key))
				}
			}
			return true
		})
	}
	return checked
}

// plainKey reports whether TinyGo hashes a map key of type t as bytes: a
// string at the top, or numbers, bools and typed pointers, alone or in
// arrays and structs.
func plainKey(t types.Type, top bool) bool {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		return top && info&types.IsString != 0 || info&(types.IsInteger|types.IsBoolean) != 0
	case *types.Pointer, *types.Chan:
		return true
	case *types.Array:
		return plainKey(u.Elem(), false)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if !plainKey(u.Field(i).Type(), false) {
				return false
			}
		}
		return true
	}
	return false
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func TestNoReflection(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the standard library from source")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	context := build.Default
	context.GOOS, context.GOARCH = "wasip1", "wasm"
	context.BuildTags = []string{"tinygo"}
	context.CgoEnabled = false
	fset := token.NewFileSet()
	c := &checker{t: t, root: root, fset: fset, context: context,
		std: importer.ForCompiler(fset, "source", nil), packages: map[string]*types.Package{}}
	c.check(module + "/cmd/wasm")
	if len(c.packages) < 10 {
		t.Fatalf("checked only %d packages", len(c.packages))
	}
	for _, f := range c.found {
		t.Error(f)
	}
}
