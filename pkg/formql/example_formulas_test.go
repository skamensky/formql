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

				catalog := loadWorkspaceCatalogForFile(t, workspace, path, formulaText)
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

func TestOfflineWorkspaceDocumentExamplesCompile(t *testing.T) {
	for _, workspace := range loadExampleWorkspaces(t) {
		t.Run(workspace.Name, func(t *testing.T) {
			documentDir := filepath.Join(workspace.Root, "documents")
			entries, err := os.ReadDir(documentDir)
			if os.IsNotExist(err) {
				return
			}
			if err != nil {
				t.Fatalf("read example document directory: %v", err)
			}

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Name() < entries[j].Name()
			})

			documentCount := 0
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".formql") {
					continue
				}
				documentCount++

				path := filepath.Join(documentDir, entry.Name())
				documentText, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read document %s: %v", entry.Name(), err)
				}

				catalog := loadWorkspaceCatalogForFile(t, workspace, path, documentText)
				compilation, err := formql.CompileDocument(string(documentText), catalog)
				if err != nil {
					t.Fatalf("compile document %s: %v", entry.Name(), err)
				}
				if compilation.HIR == nil {
					t.Fatalf("compile document %s: missing HIR", entry.Name())
				}
				if len(compilation.HIR.Fields) < 2 {
					t.Fatalf("compile document %s: expected multiple fields", entry.Name())
				}
				if !strings.Contains(compilation.SQL.Query, "SELECT\n") {
					t.Fatalf("compile document %s: expected multi-line SELECT query, got %q", entry.Name(), compilation.SQL.Query)
				}
			}

			if documentCount == 0 {
				t.Fatalf("expected at least one document example in %s", documentDir)
			}
		})
	}
}
