package betrecords

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListEndpointsDoNotMaterializeAllBetRows(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	path := filepath.Join(filepath.Dir(currentFile), "service_db.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse service_db.go: %v", err)
	}

	assertMethodDoesNotCall(t, parsed, "GroupsWithFilter", "loadRowsFiltered")
	assertMethodDoesNotCall(t, parsed, "Detail", "loadRows")
}

func assertMethodDoesNotCall(t *testing.T, file *ast.File, methodName, forbiddenCall string) {
	t.Helper()
	foundMethod := false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != methodName {
			continue
		}
		foundMethod = true
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == forbiddenCall {
				t.Errorf("%s must page and aggregate in PostgreSQL; it still calls %s", methodName, forbiddenCall)
			}
			return true
		})
	}
	if !foundMethod {
		t.Fatalf("method %s not found", methodName)
	}
}
