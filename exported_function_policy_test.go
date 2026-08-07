package verity_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPublicAPIDoesNotAliasInternalFunctions(t *testing.T) {
	t.Parallel()

	allowedVariables := map[string]bool{
		"LastResponseStatusQ": true,
		"LastResponseBodyQ":   true,
		"ResponseTimeQ":       true,
	}

	err := filepath.WalkDir(".", func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if filename != "." && (entry.Name() == "internal" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(filename, "_test.go") || filepath.Ext(filename) != ".go" {
			return nil
		}

		fileset := token.NewFileSet()
		file, err := parser.ParseFile(fileset, filename, nil, 0)
		if err != nil {
			return err
		}

		internalImports := map[string]bool{}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if !strings.Contains(importPath, "/internal/") && !strings.HasSuffix(importPath, "/internal") {
				continue
			}
			name := path.Base(importPath)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			internalImports[name] = true
		}

		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, rawSpec := range general.Specs {
				spec := rawSpec.(*ast.ValueSpec)
				for index, name := range spec.Names {
					allowedQuestion := filepath.ToSlash(filename) == "verity_abilities/api/api.go" && allowedVariables[name.Name]
					if !ast.IsExported(name.Name) || allowedQuestion || index >= len(spec.Values) {
						continue
					}
					selector, ok := spec.Values[index].(*ast.SelectorExpr)
					if !ok {
						continue
					}
					packageName, ok := selector.X.(*ast.Ident)
					if ok && internalImports[packageName.Name] {
						t.Errorf("%s:%d: exported var %s aliases internal callable %s.%s; use a typed wrapper function", filename, fileset.Position(name.Pos()).Line, name.Name, packageName.Name, selector.Sel.Name)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan public API: %v", err)
	}
}
