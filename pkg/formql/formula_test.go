package formql_test

import (
	"strings"
	"testing"

	"github.com/skamensky/formql/pkg/formql"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/schema"
)

type mockCatalog struct {
	baseTable     string
	tables        map[string]schema.Table
	relationships map[string]schema.Relationship
}

func (m *mockCatalog) BaseTableName() string {
	return m.baseTable
}

func (m *mockCatalog) Validate() error {
	return nil
}

func (m *mockCatalog) Table(name string) (*schema.Table, bool) {
	table, ok := m.tables[strings.ToLower(name)]
	if !ok {
		return nil, false
	}
	copy := table
	return &copy, true
}

func (m *mockCatalog) ColumnType(tableName, columnName string) (schema.Type, bool) {
	table, ok := m.tables[strings.ToLower(tableName)]
	if !ok {
		return schema.TypeUnknown, false
	}
	for _, column := range table.Columns {
		if column.Name == strings.ToLower(columnName) {
			return column.Type, true
		}
	}
	return schema.TypeUnknown, false
}

func (m *mockCatalog) Relationship(fromTable, relationshipName string) (*schema.Relationship, bool) {
	rel, ok := m.relationships[strings.ToLower(fromTable)+":"+strings.ToLower(relationshipName)]
	if !ok {
		return nil, false
	}
	copy := rel
	return &copy, true
}

func (m *mockCatalog) ColumnsForTable(name string) []schema.Column {
	table, ok := m.Table(name)
	if !ok {
		return nil
	}
	columns := make([]schema.Column, len(table.Columns))
	copy(columns, table.Columns)
	return columns
}

func (m *mockCatalog) RelationshipsFrom(tableName string) []schema.Relationship {
	prefix := strings.ToLower(tableName) + ":"
	relationships := make([]schema.Relationship, 0)
	for key, relationship := range m.relationships {
		if strings.HasPrefix(key, prefix) {
			relationships = append(relationships, relationship)
		}
	}
	return relationships
}

func boolPtr(value bool) *bool {
	return &value
}

func TestCompileArithmeticQuery(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
				},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	compilation, err := formql.Compile("amount + 2 * 3", catalog, "result")
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	want := "SELECT (t0.\"amount\" + (2 * 3)) AS \"result\"\nFROM \"submission\" t0"
	if compilation.SQL.Query != want {
		t.Fatalf("unexpected query\nwant:\n%s\n\ngot:\n%s", want, compilation.SQL.Query)
	}
}

func TestMockCatalogCanEmitJoinWarningsAndDedupeJoins(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "merchant_id", Type: schema.TypeNumber},
				},
			},
			"merchant": {
				Name: "merchant",
				Columns: []schema.Column{
					{Name: "id", Type: schema.TypeNumber},
					{Name: "name", Type: schema.TypeString},
					{Name: "category", Type: schema.TypeString},
				},
			},
		},
		relationships: map[string]schema.Relationship{
			"submission:merchant": {
				Name:              "merchant",
				FromTable:         "submission",
				ToTable:           "merchant",
				JoinColumn:        "merchant_id",
				TargetColumn:      "id",
				JoinColumnIndexed: boolPtr(false),
			},
		},
	}

	compilation, err := formql.Compile(`merchant_rel.name & merchant_rel.category`, catalog, "result")
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if len(compilation.HIR.Joins) != 1 {
		t.Fatalf("expected one deduped join, got %d", len(compilation.HIR.Joins))
	}
	if len(compilation.HIR.Warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(compilation.HIR.Warnings))
	}
	if !strings.Contains(compilation.HIR.Warnings[0].Message, "non-indexed source column") {
		t.Fatalf("unexpected warning message: %s", compilation.HIR.Warnings[0].Message)
	}
	if strings.Count(compilation.SQL.Query, "LEFT JOIN") != 1 {
		t.Fatalf("expected one LEFT JOIN in query, got:\n%s", compilation.SQL.Query)
	}
}

func TestMultiLevelRelationshipAndIf(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "opportunity",
		tables: map[string]schema.Table{
			"opportunity": {
				Name: "opportunity",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
					{Name: "customer_id", Type: schema.TypeNumber},
				},
			},
			"customer": {
				Name: "customer",
				Columns: []schema.Column{
					{Name: "id", Type: schema.TypeNumber},
					{Name: "assigned_rep_id", Type: schema.TypeNumber},
				},
			},
			"rep": {
				Name: "rep",
				Columns: []schema.Column{
					{Name: "id", Type: schema.TypeNumber},
					{Name: "name", Type: schema.TypeString},
				},
			},
		},
		relationships: map[string]schema.Relationship{
			"opportunity:customer": {
				Name:                "customer",
				FromTable:           "opportunity",
				ToTable:             "customer",
				JoinColumn:          "customer_id",
				TargetColumn:        "id",
				JoinColumnIndexed:   boolPtr(true),
				TargetColumnIndexed: boolPtr(true),
			},
			"customer:assigned_rep": {
				Name:                "assigned_rep",
				FromTable:           "customer",
				ToTable:             "rep",
				JoinColumn:          "assigned_rep_id",
				TargetColumn:        "id",
				JoinColumnIndexed:   boolPtr(true),
				TargetColumnIndexed: boolPtr(true),
			},
		},
	}

	compilation, err := formql.Compile(`IF(amount > 100, customer_rel.assigned_rep_rel.name, "low")`, catalog, "result")
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if len(compilation.HIR.Joins) != 2 {
		t.Fatalf("expected two joins, got %d", len(compilation.HIR.Joins))
	}
	if len(compilation.HIR.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(compilation.HIR.Warnings))
	}
	if !strings.Contains(compilation.SQL.Query, "CASE WHEN") {
		t.Fatalf("expected CASE WHEN in query, got:\n%s", compilation.SQL.Query)
	}
	if !strings.Contains(compilation.SQL.Query, "rel_customer_assigned_rep") {
		t.Fatalf("expected nested relationship alias in query, got:\n%s", compilation.SQL.Query)
	}
}

func TestUnknownColumnReturnsError(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name:    "submission",
				Columns: []schema.Column{{Name: "amount", Type: schema.TypeNumber}},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	_, err := formql.Compile("missing_column", catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}
	if !strings.Contains(err.Error(), "unknown column") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMissingOperatorDiagnosticIncludesHint(t *testing.T) {
	_, err := formql.Parse(`IF(customer_rel.email = NULL, "missing-email", customer_rel.first_name " " & customer_rel.last_name)`)
	if err == nil {
		t.Fatal("expected parse error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "missing_operator_between_expressions" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "use '&'") {
		t.Fatalf("unexpected hint: %s", typed.Hint)
	}
}

func TestMissingFunctionCloseDiagnosticIncludesClosingParenHint(t *testing.T) {
	_, err := formql.Parse(`IF(customer_rel.email = NULL, "missing-email", customer_rel.first_name & " " & customer_rel.last_name`)
	if err == nil {
		t.Fatal("expected parse error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "unexpected_token" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Message, "expected ')'") {
		t.Fatalf("unexpected message: %s", typed.Message)
	}
	if !strings.Contains(typed.Hint, "close the function call with ')'") {
		t.Fatalf("unexpected hint: %s", typed.Hint)
	}
}

func TestUnknownColumnDiagnosticSuggestsClosestMatch(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
					{Name: "stage", Type: schema.TypeString},
				},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	_, err := formql.Compile("amunt", catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "unknown_column" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "amount") {
		t.Fatalf("expected closest-match hint, got %q", typed.Hint)
	}
}

func TestUnknownRelationshipDiagnosticSuggestsRelName(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "opportunity",
		tables: map[string]schema.Table{
			"opportunity": {
				Name: "opportunity",
				Columns: []schema.Column{
					{Name: "customer_id", Type: schema.TypeNumber},
				},
			},
			"customer": {
				Name: "customer",
				Columns: []schema.Column{
					{Name: "email", Type: schema.TypeString},
				},
			},
		},
		relationships: map[string]schema.Relationship{
			"opportunity:customer": {
				Name:         "customer",
				FromTable:    "opportunity",
				ToTable:      "customer",
				JoinColumn:   "customer_id",
				TargetColumn: "id",
			},
		},
	}

	_, err := formql.Compile(`custmer_rel.email`, catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "unknown_relationship" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "customer_rel") {
		t.Fatalf("expected relationship suggestion, got %q", typed.Hint)
	}
}

func TestUnknownFunctionDiagnosticSuggestsBuiltinSignature(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
				},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	_, err := formql.Compile(`IFF(amount > 0, "yes", "no")`, catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "unknown_function" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "IF(condition, whenTrue, whenFalse)") {
		t.Fatalf("expected builtin suggestion, got %q", typed.Hint)
	}
}

func TestStringConcatTypeDiagnosticSuggestsSTRINGCast(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
				},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	_, err := formql.Compile(`amount & " usd"`, catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "invalid_concat_operands" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "STRING(...)") {
		t.Fatalf("expected cast hint, got %q", typed.Hint)
	}
}

func TestFunctionArityDiagnosticIncludesSignatureHint(t *testing.T) {
	catalog := &mockCatalog{
		baseTable: "submission",
		tables: map[string]schema.Table{
			"submission": {
				Name: "submission",
				Columns: []schema.Column{
					{Name: "amount", Type: schema.TypeNumber},
				},
			},
		},
		relationships: map[string]schema.Relationship{},
	}

	_, err := formql.Compile(`IF(amount > 0, "yes")`, catalog, "result")
	if err == nil {
		t.Fatal("expected compile error")
	}

	typed, ok := diagnostic.AsError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if typed.Code != "invalid_function_arity" {
		t.Fatalf("unexpected code: %s", typed.Code)
	}
	if !strings.Contains(typed.Hint, "IF(condition, whenTrue, whenFalse)") {
		t.Fatalf("expected signature hint, got %q", typed.Hint)
	}
}
