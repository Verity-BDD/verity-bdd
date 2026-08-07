package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFindInternalTypeLeaks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		files     map[string]string
		wantLeaks int
	}{
		{
			name: "checks every production Go file",
			files: map[string]string{
				"api.go": "package api\n",
				"questions.go": `package api

import internalapi "example.com/project/internal/api"

func LeakedQuestion() internalapi.Question { return internalapi.Question{} }
`,
			},
			wantLeaks: 1,
		},
		{
			name: "recognizes internal package root",
			files: map[string]string{
				"api.go": `package api

import internalapi "example.com/project/internal"

func LeakedQuestion() internalapi.Question { return internalapi.Question{} }
`,
			},
			wantLeaks: 1,
		},
		{
			name: "ignores test files",
			files: map[string]string{
				"api.go": "package api\n",
				"questions_test.go": `package api_test

import internalapi "example.com/project/internal/api"

func LeakedTestHelper() internalapi.Question { return internalapi.Question{} }
`,
			},
			wantLeaks: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			for name, source := range test.files {
				writeGoFile(t, directory, name, source)
			}

			leaks, err := findInternalTypeLeaks(directory)
			if err != nil {
				t.Fatalf("find internal type leaks: %v", err)
			}
			if len(leaks) != test.wantLeaks {
				t.Fatalf("expected %d internal type leaks, got %v", test.wantLeaks, leaks)
			}
		})
	}
}

func writeGoFile(t *testing.T, directory, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func findInternalTypeLeaks(directory string) ([]string, error) {
	fileset := token.NewFileSet()
	packages, err := parser.ParseDir(fileset, directory, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}

	var leaks []string
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			imports := make(map[string]string, len(file.Imports))
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return nil, err
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
					if ok && (strings.Contains(importPath, "/internal/") || strings.HasSuffix(importPath, "/internal")) {
						leaks = append(leaks, function.Name.Name+" exposes internal type "+packageName.Name+"."+selector.Sel.Name)
					}
					return true
				})
			}
		}
	}

	return leaks, nil
}

func TestExportedFunctionSignaturesDoNotExposeInternalPackages(t *testing.T) {
	t.Parallel()

	leaks, err := findInternalTypeLeaks(".")
	if err != nil {
		t.Fatalf("find internal type leaks: %v", err)
	}
	for _, leak := range leaks {
		t.Error(leak)
	}
}
