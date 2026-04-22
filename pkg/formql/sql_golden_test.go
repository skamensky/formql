package formql_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql"
)

func TestRentalAgencyGoldenSQL(t *testing.T) {
	workspaces := make(map[string]exampleWorkspace)
	for _, workspace := range loadExampleWorkspaces(t) {
		workspaces[workspace.Name] = workspace
	}

	testCases := []struct {
		name          string
		workspaceName string
		formulaFile   string
		expectedSQL   string
		expectedWarns []string
		expectedHints []string
	}{
		{
			name:          "contract customer priority",
			workspaceName: "offline-rental-contract",
			formulaFile:   "customer_priority.formql",
			expectedSQL:   "rental_contract.customer_priority.sql",
		},
		{
			name:          "contract route snapshot",
			workspaceName: "offline-rental-contract",
			formulaFile:   "route_snapshot.formql",
			expectedSQL:   "rental_contract.route_snapshot.sql",
		},
		{
			name:          "contract manager watch warning",
			workspaceName: "offline-rental-contract",
			formulaFile:   "manager_watch.formql",
			expectedSQL:   "rental_contract.manager_watch.sql",
			expectedWarns: []string{"non_indexed_join_source"},
			expectedHints: []string{"add an index on rep.manager_id"},
		},
		{
			name:          "contract vendor fallback warning",
			workspaceName: "offline-rental-contract",
			formulaFile:   "vendor_fallback.formql",
			expectedSQL:   "rental_contract.vendor_fallback.sql",
			expectedWarns: []string{"non_indexed_join_source"},
			expectedHints: []string{"add an index on fleet.vendor_id"},
		},
		{
			name:          "resale buyer channel",
			workspaceName: "offline-resale-sale",
			formulaFile:   "buyer_channel.formql",
			expectedSQL:   "resale_sale.buyer_channel.sql",
		},
		{
			name:          "resale margin band",
			workspaceName: "offline-resale-sale",
			formulaFile:   "margin_band.formql",
			expectedSQL:   "resale_sale.margin_band.sql",
		},
		{
			name:          "resale vehicle lineage",
			workspaceName: "offline-resale-sale",
			formulaFile:   "vehicle_lineage.formql",
			expectedSQL:   "resale_sale.vehicle_lineage.sql",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, ok := workspaces[tc.workspaceName]
			if !ok {
				t.Fatalf("missing workspace %s", tc.workspaceName)
			}

			catalog := loadWorkspaceCatalog(t, workspace)
			formulaPath := filepath.Join(workspace.Root, "formulas", tc.formulaFile)
			formulaText, err := os.ReadFile(formulaPath)
			if err != nil {
				t.Fatalf("read formula %s: %v", formulaPath, err)
			}

			compilation, err := formql.Compile(string(formulaText), catalog, "result")
			if err != nil {
				t.Fatalf("compile %s: %v", tc.formulaFile, err)
			}

			expectedSQLPath := filepath.Join("testdata", "rental_agency", "sql", tc.expectedSQL)
			expectedSQL, err := os.ReadFile(expectedSQLPath)
			if err != nil {
				t.Fatalf("read expected SQL %s: %v", expectedSQLPath, err)
			}
			if compilation.SQL.Query != strings.TrimSpace(string(expectedSQL)) {
				t.Fatalf("unexpected SQL for %s\nwant:\n%s\n\ngot:\n%s", tc.formulaFile, strings.TrimSpace(string(expectedSQL)), compilation.SQL.Query)
			}

			if len(compilation.HIR.Warnings) != len(tc.expectedWarns) {
				t.Fatalf("expected %d warnings, got %d", len(tc.expectedWarns), len(compilation.HIR.Warnings))
			}
			for index, wantCode := range tc.expectedWarns {
				if compilation.HIR.Warnings[index].Code != wantCode {
					t.Fatalf("warning %d code mismatch: want %s, got %s", index, wantCode, compilation.HIR.Warnings[index].Code)
				}
			}
			for index, wantHint := range tc.expectedHints {
				if !strings.Contains(compilation.HIR.Warnings[index].Hint, wantHint) {
					t.Fatalf("warning %d hint mismatch: want substring %q, got %q", index, wantHint, compilation.HIR.Warnings[index].Hint)
				}
			}
		})
	}
}
