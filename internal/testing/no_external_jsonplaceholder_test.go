package testing

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNoJSONPlaceholderDomainInGoSources(t *testing.T) {
	t.Parallel()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	needle := "jsonplaceholder" + ".typicode.com"

	var hits []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), needle) {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				relPath = path
			}
			hits = append(hits, relPath)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}

	if len(hits) > 0 {
		t.Fatalf("found external jsonplaceholder domain in Go files: %v", hits)
	}
}
