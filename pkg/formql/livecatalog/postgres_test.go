package livecatalog

import "testing"

func TestSnapshotFromIntrospection(t *testing.T) {
	snapshot, err := SnapshotFromIntrospection(IntrospectionSnapshot{
		Namespace: "public",
		BaseTable: "orders",
		Columns: []ColumnRecord{
			{TableName: "orders", ColumnName: "id", DataType: "bigint", UDTName: "int8"},
			{TableName: "orders", ColumnName: "amount", DataType: "numeric", UDTName: "numeric"},
			{TableName: "orders", ColumnName: "customer_id", DataType: "bigint", UDTName: "int8"},
			{TableName: "customers", ColumnName: "id", DataType: "bigint", UDTName: "int8"},
			{TableName: "customers", ColumnName: "email", DataType: "text", UDTName: "text"},
		},
		Relationships: []RelationshipRecord{
			{
				SourceTable:         "orders",
				SourceColumn:        "customer_id",
				TargetTable:         "customers",
				TargetColumn:        "id",
				JoinColumnIndexed:   true,
				TargetColumnIndexed: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("SnapshotFromIntrospection returned error: %v", err)
	}
	if snapshot.Catalog.BaseTable != "orders" {
		t.Fatalf("expected base table orders, got %q", snapshot.Catalog.BaseTable)
	}
	if len(snapshot.Catalog.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(snapshot.Catalog.Relationships))
	}
	if got := snapshot.Catalog.Relationships[0].Name; got != "customer_id__rel" {
		t.Fatalf("expected inferred relationship name customer_id__rel, got %q", got)
	}
}
