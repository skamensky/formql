package api

import (
	"context"
	"testing"

	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/verify"
)

const testCatalogJSON = `{
  "base_table": "orders",
  "tables": [
    {
      "name": "orders",
      "columns": [
        {"name": "id", "type": "number"},
        {"name": "amount", "type": "number"},
        {"name": "customer_id", "type": "number"}
      ]
    },
    {
      "name": "customers",
      "columns": [
        {"name": "id", "type": "number"},
        {"name": "email", "type": "string"}
      ]
    }
  ],
  "relationships": [
    {
      "name": "customer_id__rel",
      "from_table": "orders",
      "to_table": "customers",
      "join_column": "customer_id",
      "target_column": "id"
    }
  ]
}`

const testIntrospectionJSON = `{
  "namespace": "public",
  "base_table": "orders",
  "columns": [
    {"table_name":"orders","column_name":"id","data_type":"bigint","udt_name":"int8"},
    {"table_name":"orders","column_name":"amount","data_type":"numeric","udt_name":"numeric"},
    {"table_name":"orders","column_name":"customer_id","data_type":"bigint","udt_name":"int8"},
    {"table_name":"customers","column_name":"id","data_type":"bigint","udt_name":"int8"},
    {"table_name":"customers","column_name":"email","data_type":"text","udt_name":"text"}
  ],
  "relationships": [
    {
      "source_table":"orders",
      "source_column":"customer_id",
      "target_table":"customers",
      "target_column":"id",
      "join_column_indexed": true,
      "target_column_indexed": true
    }
  ]
}`

func TestLoadCatalogJSONValidates(t *testing.T) {
	catalog, err := LoadCatalogJSON([]byte(testCatalogJSON))
	if err != nil {
		t.Fatalf("LoadCatalogJSON returned error: %v", err)
	}
	if catalog.BaseTable != "orders" {
		t.Fatalf("unexpected base table %q", catalog.BaseTable)
	}
}

func TestVerifySQLUsesDefaultVerifier(t *testing.T) {
	result, err := VerifySQL(context.Background(), "SELECT 1", verify.ModeSyntax)
	if err != nil {
		t.Fatalf("VerifySQL returned error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK result, got %#v", result)
	}
}

func TestCompileCatalogJSONCompilesFormula(t *testing.T) {
	compilation, err := CompileCatalogJSON([]byte(testCatalogJSON), `customer_id__rel.email & " / " & STRING(amount)`, "result")
	if err != nil {
		t.Fatalf("CompileCatalogJSON returned error: %v", err)
	}
	if compilation.SQL.Query == "" {
		t.Fatal("expected non-empty SQL query")
	}
}

func TestCompileDocumentCatalogJSONCompilesDocument(t *testing.T) {
	compilation, err := CompileDocumentCatalogJSON([]byte(testCatalogJSON), `amount, customer_id__rel.email AS customer_email`)
	if err != nil {
		t.Fatalf("CompileDocumentCatalogJSON returned error: %v", err)
	}
	if compilation.SQL.Query == "" {
		t.Fatal("expected non-empty SQL query")
	}
	if len(compilation.HIR.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(compilation.HIR.Fields))
	}
}

func TestCompileAndVerifyCatalogJSON(t *testing.T) {
	compilation, verification, err := CompileAndVerifyCatalogJSON(context.Background(), []byte(testCatalogJSON), "amount + 1", "result", verify.ModeSyntax)
	if err != nil {
		t.Fatalf("CompileAndVerifyCatalogJSON returned error: %v", err)
	}
	if compilation == nil {
		t.Fatal("expected compilation")
	}
	if !verification.OK {
		t.Fatalf("expected verification OK, got %#v", verification)
	}
}

func TestCompileAndVerifyDocumentCatalogJSON(t *testing.T) {
	compilation, verification, err := CompileAndVerifyDocumentCatalogJSON(context.Background(), []byte(testCatalogJSON), `amount, customer_id__rel.email`, verify.ModeSyntax)
	if err != nil {
		t.Fatalf("CompileAndVerifyDocumentCatalogJSON returned error: %v", err)
	}
	if compilation == nil {
		t.Fatal("expected compilation")
	}
	if !verification.OK {
		t.Fatalf("expected verification OK, got %#v", verification)
	}
}

func TestLoadSchemaInfoJSON(t *testing.T) {
	info, err := LoadSchemaInfoJSON([]byte(testCatalogJSON))
	if err != nil {
		t.Fatalf("LoadSchemaInfoJSON returned error: %v", err)
	}
	if info.BaseTable != "orders" {
		t.Fatalf("expected base table orders, got %q", info.BaseTable)
	}
	if len(info.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(info.Relationships))
	}
}

func TestCompileAndVerifyWithProvider(t *testing.T) {
	catalogValue, err := LoadCatalogJSON([]byte(testCatalogJSON))
	if err != nil {
		t.Fatalf("LoadCatalogJSON returned error: %v", err)
	}

	provider := catalog.StaticProvider{
		Snapshot: &catalog.Snapshot{Catalog: catalogValue},
	}

	compilation, verification, err := CompileAndVerify(
		context.Background(),
		provider,
		catalog.Ref{BaseTable: "orders"},
		`customer_id__rel.email & " / " & STRING(amount)`,
		"result",
		verify.ModeSyntax,
	)
	if err != nil {
		t.Fatalf("CompileAndVerify returned error: %v", err)
	}
	if compilation == nil || compilation.SQL.Query == "" {
		t.Fatal("expected compiled SQL query")
	}
	if !verification.OK {
		t.Fatalf("expected verification OK, got %#v", verification)
	}
}

func TestCompileAndVerifyCatalogIntrospectionJSON(t *testing.T) {
	compilation, verification, err := CompileAndVerifyCatalogIntrospectionJSON(
		context.Background(),
		[]byte(testIntrospectionJSON),
		`customer_id__rel.email & " / " & STRING(amount)`,
		"result",
		verify.ModeSyntax,
	)
	if err != nil {
		t.Fatalf("CompileAndVerifyCatalogIntrospectionJSON returned error: %v", err)
	}
	if compilation == nil || compilation.SQL.Query == "" {
		t.Fatal("expected compiled SQL query")
	}
	if !verification.OK {
		t.Fatalf("expected verification OK, got %#v", verification)
	}
}
