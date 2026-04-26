package filemeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceDirective(t *testing.T) {
	metadata, ok, err := ParseSource(`// formql: table=rental_contract mode="document"
actual_total`)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata directive")
	}
	if metadata.BaseTable() != "rental_contract" {
		t.Fatalf("unexpected base table %q", metadata.BaseTable())
	}
	if metadata.Params["mode"] != "document" {
		t.Fatalf("unexpected mode metadata %q", metadata.Params["mode"])
	}
}

func TestParseSourceMergesLeadingDirectives(t *testing.T) {
	metadata, ok, err := ParseSource(`// formql: table=rental_contract
// @formql mode=document owner='ops'
actual_total`)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata directive")
	}
	if metadata.BaseTable() != "rental_contract" {
		t.Fatalf("unexpected base table %q", metadata.BaseTable())
	}
	if metadata.Params["mode"] != "document" || metadata.Params["owner"] != "ops" {
		t.Fatalf("unexpected merged metadata %#v", metadata.Params)
	}
}

func TestParseSourceRejectsConflictingDuplicateParams(t *testing.T) {
	_, _, err := ParseSource(`// formql: table=rental_contract
// formql: table=resale_sale
actual_total`)
	if err == nil {
		t.Fatal("expected duplicate metadata error")
	}
}

func TestParseSourceDirectiveAfterLeadingBlockComment(t *testing.T) {
	metadata, ok, err := ParseSource(`/* ordinary header */
// @formql base-table=resale_sale
sale_price`)
	if err != nil {
		t.Fatalf("ParseSource returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected metadata directive")
	}
	if metadata.BaseTable() != "resale_sale" {
		t.Fatalf("unexpected base table %q", metadata.BaseTable())
	}
}

func TestLoadSidecar(t *testing.T) {
	dir := t.TempDir()
	formulaPath := filepath.Join(dir, "contract_overview.formql")
	sidecarPath := filepath.Join(dir, "contract_overview.meta.json")
	if err := os.WriteFile(sidecarPath, []byte(`{"base_table":"rental_contract","owner":"ops"}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	metadata, path, ok, err := LoadSidecar(formulaPath)
	if err != nil {
		t.Fatalf("LoadSidecar returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected sidecar metadata")
	}
	if path != sidecarPath {
		t.Fatalf("unexpected sidecar path %q", path)
	}
	if metadata.BaseTable() != "rental_contract" {
		t.Fatalf("unexpected base table %q", metadata.BaseTable())
	}
	if metadata.Params["owner"] != "ops" {
		t.Fatalf("unexpected owner metadata %q", metadata.Params["owner"])
	}
}
