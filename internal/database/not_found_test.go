package database

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// A lookup that returns nil, nil hands the caller a nil pointer with no error,
// and every caller that forgot the nil check panicked (1917291, d7a7608,
// fa373b7, ed09f4f). Lookups report a missing row as ErrNotFound instead.
func TestLookupsNeverReturnNilNil(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 2 || !isNil(ret.Results[0]) || !isNil(ret.Results[1]) {
				return true
			}
			t.Errorf("%s: return nil, nil; return fmt.Errorf(\"...: %%w\", ErrNotFound) for a missing row, or a non-nil value", fset.Position(ret.Pos()))
			return true
		})
	}
}

func isNil(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}
