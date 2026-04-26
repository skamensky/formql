package livecatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/schema"

	_ "github.com/lib/pq"
)

// Source provides the raw catalog material needed to build a compiler catalog.
//
// External processes can implement this with database/sql over a connection
// string. In-server environments such as a PostgreSQL extension can implement
// the same contract through SPI or direct catalog access without any transport
// URL at all.
type Source interface {
	Tables(ctx context.Context, namespace string) ([]schema.Table, error)
	Relationships(ctx context.Context, namespace string) ([]schema.Relationship, error)
}

// ColumnRecord is raw column metadata captured from PostgreSQL introspection.
type ColumnRecord struct {
	TableSchema string `json:"table_schema,omitempty"`
	TableName   string `json:"table_name"`
	ColumnName  string `json:"column_name"`
	DataType    string `json:"data_type"`
	UDTName     string `json:"udt_name,omitempty"`
}

// RelationshipRecord is raw foreign-key metadata captured from PostgreSQL introspection.
type RelationshipRecord struct {
	SourceTable         string `json:"source_table"`
	SourceColumn        string `json:"source_column"`
	TargetSchema        string `json:"target_schema,omitempty"`
	TargetTable         string `json:"target_table"`
	TargetColumn        string `json:"target_column"`
	JoinColumnIndexed   bool   `json:"join_column_indexed"`
	TargetColumnIndexed bool   `json:"target_column_indexed"`
}

// IntrospectionSnapshot is a host-facing raw metadata payload.
type IntrospectionSnapshot struct {
	Namespace     string               `json:"namespace,omitempty"`
	BaseTable     string               `json:"base_table"`
	Columns       []ColumnRecord       `json:"columns"`
	Relationships []RelationshipRecord `json:"relationships"`
}

// crossSchemaSource is implemented by sources that can detect FK targets in other schemas.
type crossSchemaSource interface {
	CrossSchemaTargets(ctx context.Context, namespace string) ([]string, error)
}

// BuildSnapshot materializes a validated compiler catalog snapshot from a live source.
func BuildSnapshot(ctx context.Context, source Source, ref catalog.Ref) (*catalog.Snapshot, error) {
	namespace, baseTable := resolveRef(ref, "public")

	tables, err := source.Tables(ctx, namespace)
	if err != nil {
		return nil, err
	}

	relationships, err := source.Relationships(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Pull in tables from schemas referenced by cross-schema foreign keys.
	if css, ok := source.(crossSchemaSource); ok {
		foreignSchemaList, err := css.CrossSchemaTargets(ctx, namespace)
		if err != nil {
			return nil, err
		}
		for _, foreignSchema := range foreignSchemaList {
			foreignTables, err := source.Tables(ctx, foreignSchema)
			if err != nil {
				return nil, err
			}
			tables = append(tables, foreignTables...)
		}
	}

	compilationCatalog := &schema.Catalog{
		BaseTable:     strings.ToLower(strings.TrimSpace(baseTable)),
		BaseSchema:    namespace,
		Tables:        tables,
		Relationships: relationships,
	}

	if err := compilationCatalog.Validate(); err != nil {
		return nil, err
	}

	return &catalog.Snapshot{
		Catalog:  compilationCatalog,
		Revision: namespace,
	}, nil
}

// SnapshotFromIntrospection converts raw host metadata into a compiler snapshot.
func SnapshotFromIntrospection(payload IntrospectionSnapshot) (*catalog.Snapshot, error) {
	namespace := strings.ToLower(strings.TrimSpace(payload.Namespace))
	compilationCatalog := &schema.Catalog{
		BaseTable:     strings.ToLower(strings.TrimSpace(payload.BaseTable)),
		BaseSchema:    namespace,
		Tables:        tablesFromColumnRecords(payload.Columns),
		Relationships: relationshipsFromRecords(payload.Relationships),
	}
	if err := compilationCatalog.Validate(); err != nil {
		return nil, err
	}
	return &catalog.Snapshot{
		Catalog:  compilationCatalog,
		Revision: namespace,
	}, nil
}

// SnapshotFromIntrospectionJSON decodes raw host metadata and builds a compiler snapshot.
func SnapshotFromIntrospectionJSON(data []byte) (*catalog.Snapshot, error) {
	var payload IntrospectionSnapshot
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode introspection json: %w", err)
	}
	return SnapshotFromIntrospection(payload)
}

// PostgresSource introspects a PostgreSQL schema into raw catalog material.
type PostgresSource struct {
	db         *sql.DB
	schemaName string
}

// PostgresProvider loads a compiler catalog from a Postgres-backed source.
type PostgresProvider struct {
	source *PostgresSource
	db     *sql.DB
}

// NewPostgresProvider creates a Postgres-backed catalog provider.
func NewPostgresProvider(ctx context.Context, databaseURL string) (*PostgresProvider, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &PostgresProvider{
		source: &PostgresSource{
			db:         db,
			schemaName: "public",
		},
		db: db,
	}, nil
}

// Close closes the underlying connection pool.
func (p *PostgresProvider) Close() {
	if p == nil || p.db == nil {
		return
	}
	_ = p.db.Close()
}

// Load introspects a live schema snapshot for compiler or tooling use.
func (p *PostgresProvider) Load(ctx context.Context, ref catalog.Ref) (*catalog.Snapshot, error) {
	return BuildSnapshot(ctx, p.source, ref)
}

// Tables loads table and column metadata from PostgreSQL.
func (s *PostgresSource) Tables(ctx context.Context, namespace string) ([]schema.Table, error) {
	const query = `
		SELECT table_schema, table_name, column_name, data_type, udt_name
		FROM information_schema.columns
		WHERE table_schema = $1
		ORDER BY table_name, ordinal_position
	`

	rows, err := s.db.QueryContext(ctx, query, schemaName(namespace, s.schemaName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnRecords := make([]ColumnRecord, 0)

	for rows.Next() {
		var tableSchema string
		var tableName string
		var columnName string
		var dataType string
		var udtName string
		if err := rows.Scan(&tableSchema, &tableName, &columnName, &dataType, &udtName); err != nil {
			return nil, err
		}

		columnRecords = append(columnRecords, ColumnRecord{
			TableSchema: strings.ToLower(tableSchema),
			TableName:   strings.ToLower(tableName),
			ColumnName:  strings.ToLower(columnName),
			DataType:    strings.ToLower(dataType),
			UDTName:     strings.ToLower(udtName),
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return tablesFromColumnRecords(columnRecords), nil
}

// CrossSchemaTargets returns distinct schemas that are FK targets from the given namespace.
func (s *PostgresSource) CrossSchemaTargets(ctx context.Context, namespace string) ([]string, error) {
	const query = `
		SELECT DISTINCT ccu.table_schema
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_schema = tc.constraint_schema
			AND ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND ccu.table_schema != $1
	`

	rows, err := s.db.QueryContext(ctx, query, schemaName(namespace, s.schemaName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make([]string, 0)
	for rows.Next() {
		var targetSchema string
		if err := rows.Scan(&targetSchema); err != nil {
			return nil, err
		}
		targets = append(targets, strings.ToLower(targetSchema))
	}

	return targets, rows.Err()
}

// Relationships loads relationship and index metadata from PostgreSQL.
func (s *PostgresSource) Relationships(ctx context.Context, namespace string) ([]schema.Relationship, error) {
	const query = `
		SELECT
			tc.table_name AS source_table,
			kcu.column_name AS source_column,
			ccu.table_name AS target_table,
			ccu.column_name AS target_column,
			EXISTS (
				SELECT 1
				FROM pg_index idx
				JOIN pg_class cls ON cls.oid = idx.indrelid
				JOIN pg_namespace ns ON ns.oid = cls.relnamespace
				JOIN pg_attribute attr ON attr.attrelid = cls.oid
				WHERE ns.nspname = tc.table_schema
					AND cls.relname = tc.table_name
					AND attr.attname = kcu.column_name
					AND attr.attnum = ANY(idx.indkey)
			) AS source_indexed,
			EXISTS (
				SELECT 1
				FROM pg_index idx
				JOIN pg_class cls ON cls.oid = idx.indrelid
				JOIN pg_namespace ns ON ns.oid = cls.relnamespace
				JOIN pg_attribute attr ON attr.attrelid = cls.oid
				WHERE ns.nspname = tc.table_schema
					AND cls.relname = ccu.table_name
					AND attr.attname = ccu.column_name
					AND attr.attnum = ANY(idx.indkey)
			) AS target_indexed
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_schema = tc.constraint_schema
			AND ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
		ORDER BY source_table, source_column
	`

	rows, err := s.db.QueryContext(ctx, query, schemaName(namespace, s.schemaName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]RelationshipRecord, 0)
	for rows.Next() {
		var sourceTable string
		var sourceColumn string
		var targetTable string
		var targetColumn string
		var sourceIndexed bool
		var targetIndexed bool
		if err := rows.Scan(
			&sourceTable,
			&sourceColumn,
			&targetTable,
			&targetColumn,
			&sourceIndexed,
			&targetIndexed,
		); err != nil {
			return nil, err
		}

		records = append(records, RelationshipRecord{
			SourceTable:         strings.ToLower(sourceTable),
			SourceColumn:        strings.ToLower(sourceColumn),
			TargetTable:         strings.ToLower(targetTable),
			TargetColumn:        strings.ToLower(targetColumn),
			JoinColumnIndexed:   sourceIndexed,
			TargetColumnIndexed: targetIndexed,
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return relationshipsFromRecords(records), nil
}

func relationshipNameForRecord(record RelationshipRecord) string {
	sourceColumn := strings.ToLower(strings.TrimSpace(record.SourceColumn))
	if sourceColumn == "" {
		return ""
	}
	return sourceColumn + "__rel"
}

func mapPostgresType(dataType, udtName string) schema.Type {
	switch strings.ToLower(dataType) {
	case "smallint", "integer", "bigint", "numeric", "decimal", "real", "double precision":
		return schema.TypeNumber
	case "date", "timestamp without time zone", "timestamp with time zone":
		return schema.TypeDate
	case "boolean":
		return schema.TypeBoolean
	case "array":
		return schema.TypeString
	default:
		_ = udtName
		return schema.TypeString
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func tablesFromColumnRecords(records []ColumnRecord) []schema.Table {
	orderedNames := make([]string, 0)
	tableColumns := make(map[string][]schema.Column)
	tableSchemas := make(map[string]string)
	for _, record := range records {
		tableName := strings.ToLower(strings.TrimSpace(record.TableName))
		if _, ok := tableColumns[tableName]; !ok {
			orderedNames = append(orderedNames, tableName)
			tableSchemas[tableName] = strings.ToLower(strings.TrimSpace(record.TableSchema))
		}
		tableColumns[tableName] = append(tableColumns[tableName], schema.Column{
			Name: strings.ToLower(strings.TrimSpace(record.ColumnName)),
			Type: mapPostgresType(record.DataType, record.UDTName),
		})
	}

	tables := make([]schema.Table, 0, len(orderedNames))
	for _, tableName := range orderedNames {
		tables = append(tables, schema.Table{
			Name:    tableName,
			Schema:  tableSchemas[tableName],
			Columns: tableColumns[tableName],
		})
	}
	return tables
}

func relationshipsFromRecords(records []RelationshipRecord) []schema.Relationship {
	relationships := make([]schema.Relationship, 0, len(records))
	for _, record := range records {
		name := relationshipNameForRecord(record)

		relationships = append(relationships, schema.Relationship{
			Name:                name,
			FromTable:           strings.ToLower(strings.TrimSpace(record.SourceTable)),
			ToTable:             strings.ToLower(strings.TrimSpace(record.TargetTable)),
			JoinColumn:          strings.ToLower(strings.TrimSpace(record.SourceColumn)),
			TargetColumn:        strings.ToLower(strings.TrimSpace(record.TargetColumn)),
			JoinColumnIndexed:   boolPtr(record.JoinColumnIndexed),
			TargetColumnIndexed: boolPtr(record.TargetColumnIndexed),
		})
	}
	return relationships
}

// LoadCatalog is a helper for one-shot introspection without keeping a provider around.
func LoadCatalog(ctx context.Context, databaseURL, baseTable string) (*schema.Catalog, error) {
	provider, err := NewPostgresProvider(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	defer provider.Close()

	snapshot, err := provider.Load(ctx, catalog.Ref{BaseTable: baseTable})
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}
	return snapshot.Catalog, nil
}

func resolveRef(ref catalog.Ref, fallbackNamespace string) (string, string) {
	namespace := strings.ToLower(strings.TrimSpace(ref.Namespace))
	baseTable := strings.ToLower(strings.TrimSpace(ref.BaseTable))
	if strings.Contains(baseTable, ".") {
		parts := strings.SplitN(baseTable, ".", 2)
		if namespace == "" {
			namespace = strings.TrimSpace(parts[0])
		}
		baseTable = strings.TrimSpace(parts[1])
	}
	if namespace == "" {
		namespace = fallbackNamespace
	}
	return namespace, baseTable
}

func schemaName(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
