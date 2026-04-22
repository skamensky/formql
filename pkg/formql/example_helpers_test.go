package formql_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql/schema"
)

type exampleWorkspace struct {
	Name       string
	Root       string
	SchemaPath string
	BaseTable  string
}

type workspaceSettings struct {
	SchemaPath string `json:"formql.schemaPath"`
	BaseTable  string `json:"formql.baseTable"`
}

func loadExampleWorkspaces(t *testing.T) []exampleWorkspace {
	t.Helper()

	workspacesRoot := filepath.Join("..", "..", "examples", "workspaces")
	entries, err := os.ReadDir(workspacesRoot)
	if err != nil {
		t.Fatalf("read example workspace root: %v", err)
	}

	workspaces := make([]exampleWorkspace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(workspacesRoot, entry.Name())
		settingsPath := filepath.Join(root, ".vscode", "settings.json")
		settingsFile, err := os.ReadFile(settingsPath)
		if err != nil {
			continue
		}

		var settings workspaceSettings
		if err := json.Unmarshal(settingsFile, &settings); err != nil {
			t.Fatalf("decode %s: %v", settingsPath, err)
		}

		schemaPath := strings.ReplaceAll(settings.SchemaPath, "${workspaceFolder}", root)
		workspaces = append(workspaces, exampleWorkspace{
			Name:       entry.Name(),
			Root:       root,
			SchemaPath: filepath.Clean(schemaPath),
			BaseTable:  strings.ToLower(strings.TrimSpace(settings.BaseTable)),
		})
	}

	if len(workspaces) == 0 {
		t.Fatal("expected at least one example workspace")
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Name < workspaces[j].Name
	})

	return workspaces
}

func loadWorkspaceCatalog(t *testing.T, workspace exampleWorkspace) *schema.Catalog {
	t.Helper()

	catalogFile, err := os.ReadFile(workspace.SchemaPath)
	if err != nil {
		t.Fatalf("read schema for %s: %v", workspace.Name, err)
	}

	var catalog schema.Catalog
	if err := json.Unmarshal(catalogFile, &catalog); err != nil {
		t.Fatalf("decode schema for %s: %v", workspace.Name, err)
	}
	if workspace.BaseTable != "" {
		catalog.BaseTable = workspace.BaseTable
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate schema for %s: %v", workspace.Name, err)
	}

	return &catalog
}
