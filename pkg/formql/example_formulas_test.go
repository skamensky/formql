package formql_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql"
)

func TestOfflineWorkspaceExamplesCompile(t *testing.T) {
	for _, workspace := range loadExampleWorkspaces(t) {
		t.Run(workspace.Name, func(t *testing.T) {
			catalog := loadWorkspaceCatalog(t, workspace)
			formulaDir := filepath.Join(workspace.Root, "formulas")
			entries, err := os.ReadDir(formulaDir)
			if err != nil {
				t.Fatalf("read example formula directory: %v", err)
			}

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Name() < entries[j].Name()
			})

			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".formql") {
					continue
				}

				path := filepath.Join(formulaDir, entry.Name())
				formulaText, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read formula %s: %v", entry.Name(), err)
				}

				compilation, err := formql.Compile(string(formulaText), catalog, "result")
				if err != nil {
					t.Fatalf("compile %s: %v", entry.Name(), err)
				}
				if compilation.HIR == nil {
					t.Fatalf("compile %s: missing HIR", entry.Name())
				}
				if !strings.Contains(compilation.SQL.Query, "SELECT ") {
					t.Fatalf("compile %s: expected SELECT query, got %q", entry.Name(), compilation.SQL.Query)
				}
			}
		})
	}
}
