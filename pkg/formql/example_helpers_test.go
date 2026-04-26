package formql_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql/filemeta"
	"github.com/skamensky/formql/pkg/formql/schema"
)

type exampleWorkspace struct {
	Name       string
	Root       string
	SchemaPath string
}

type workspaceSettings struct {
	SchemaPath string `json:"formql.schemaPath"`
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
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate schema for %s: %v", workspace.Name, err)
	}

	return &catalog
}

func loadWorkspaceCatalogForFile(t *testing.T, workspace exampleWorkspace, sourcePath string, sourceText []byte) *schema.Catalog {
	t.Helper()

	catalog := loadWorkspaceCatalog(t, workspace)
	baseTable := resolveExampleFileBaseTable(t, sourcePath, string(sourceText))
	catalog.BaseTable = baseTable
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate schema for %s with base table %s: %v", workspace.Name, baseTable, err)
	}
	return catalog
}

func resolveExampleFileBaseTable(t *testing.T, sourcePath, sourceText string) string {
	t.Helper()

	if metadata, ok, err := filemeta.ParseSource(sourceText); err != nil {
		t.Fatalf("parse metadata for %s: %v", sourcePath, err)
	} else if ok && metadata.BaseTable() != "" {
		return metadata.BaseTable()
	}

	if metadata, _, ok, err := filemeta.LoadSidecar(sourcePath); err != nil {
		t.Fatalf("load metadata sidecar for %s: %v", sourcePath, err)
	} else if ok && metadata.BaseTable() != "" {
		return metadata.BaseTable()
	}

	t.Fatalf("missing FormQL table metadata for %s", sourcePath)
	return ""
}
