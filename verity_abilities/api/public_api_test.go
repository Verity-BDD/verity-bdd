package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
	"testing"
)

func TestExportedFunctionSignaturesDoNotExposeInternalPackages(t *testing.T) {
	t.Parallel()

	fileset := token.NewFileSet()
	file, err := parser.ParseFile(fileset, "api.go", nil, 0)
	if err != nil {
		t.Fatalf("parse public API: %v", err)
	}

	imports := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("unquote import path %s: %v", spec.Path.Value, err)
		}

		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = importPath
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || !ast.IsExported(function.Name.Name) {
			continue
		}

		ast.Inspect(function.Type, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			packageName, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			importPath, ok := imports[packageName.Name]
			if ok && strings.Contains(importPath, "/internal/") {
				t.Errorf("%s exposes internal type %s.%s", function.Name.Name, packageName.Name, selector.Sel.Name)
			}
			return true
		})
	}
}
