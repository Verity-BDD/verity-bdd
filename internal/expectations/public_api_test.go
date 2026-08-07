package expectations_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestNewContainsSubstringIsNotExposed(t *testing.T) {
	t.Parallel()

	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, 0)
	if err != nil {
		t.Fatalf("parse expectations package: %v", err)
	}

	for _, file := range packages["expectations"].Files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.Name == "NewContainsSubstring" {
				t.Fatal("NewContainsSubstring must not be exposed; use ContainsSubstring as the sole factory")
			}
		}
	}
}
