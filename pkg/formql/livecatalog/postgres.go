package livecatalog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skamensky/formql/pkg/formql/schema"

	_ "github.com/lib/pq"
)

// Provider loads formula catalogs from a live database.
type Provider interface {
	LoadCatalog(ctx context.Context, baseTable string) (*schema.Catalog, error)
	Close()
}

// PostgresProvider introspects a PostgreSQL schema into a formula catalog.
type PostgresProvider struct {
	db         *sql.DB
	schemaName string
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
		db:         db,
		schemaName: "public",
	}, nil
}

// Close closes the underlying connection pool.
func (p *PostgresProvider) Close() {
	if p == nil || p.db == nil {
		return
	}
	_ = p.db.Close()
}

// LoadCatalog introspects a live schema for compiler use.
func (p *PostgresProvider) LoadCatalog(ctx context.Context, baseTable string) (*schema.Catalog, error) {
	tables, err := p.loadTables(ctx)
	if err != nil {
		return nil, err
	}

	relationships, err := p.loadRelationships(ctx)
	if err != nil {
		return nil, err
	}

	catalog := &schema.Catalog{
		BaseTable:     strings.ToLower(strings.TrimSpace(baseTable)),
		Tables:        tables,
		Relationships: relationships,
	}

	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	return catalog, nil
}

func (p *PostgresProvider) loadTables(ctx context.Context) ([]schema.Table, error) {
	const query = `
		SELECT table_name, column_name, data_type, udt_name
		FROM information_schema.columns
		WHERE table_schema = $1
		ORDER BY table_name, ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, p.schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orderedNames := make([]string, 0)
	tableColumns := make(map[string][]schema.Column)

	for rows.Next() {
		var tableName string
		var columnName string
		var dataType string
		var udtName string
		if err := rows.Scan(&tableName, &columnName, &dataType, &udtName); err != nil {
			return nil, err
		}

		normalizedTable := strings.ToLower(tableName)
		if _, ok := tableColumns[normalizedTable]; !ok {
			orderedNames = append(orderedNames, normalizedTable)
		}

		tableColumns[normalizedTable] = append(tableColumns[normalizedTable], schema.Column{
			Name: strings.ToLower(columnName),
			Type: mapPostgresType(dataType, udtName),
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	tables := make([]schema.Table, 0, len(orderedNames))
	for _, tableName := range orderedNames {
		tables = append(tables, schema.Table{
			Name:    tableName,
			Columns: tableColumns[tableName],
		})
	}

	return tables, nil
}

func (p *PostgresProvider) loadRelationships(ctx context.Context) ([]schema.Relationship, error) {
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
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
		ORDER BY source_table, source_column
	`

	rows, err := p.db.QueryContext(ctx, query, p.schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relationships := make([]schema.Relationship, 0)
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

		relationships = append(relationships, schema.Relationship{
			Name:                relationshipNameForColumn(sourceColumn),
			FromTable:           strings.ToLower(sourceTable),
			ToTable:             strings.ToLower(targetTable),
			JoinColumn:          strings.ToLower(sourceColumn),
			TargetColumn:        strings.ToLower(targetColumn),
			JoinColumnIndexed:   boolPtr(sourceIndexed),
			TargetColumnIndexed: boolPtr(targetIndexed),
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return relationships, nil
}

func relationshipNameForColumn(columnName string) string {
	name := strings.ToLower(columnName)
	name = strings.TrimSuffix(name, "_id")
	name = strings.TrimSuffix(name, "_fk")
	return name
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

// LoadCatalog is a helper for one-shot introspection without keeping a provider around.
func LoadCatalog(ctx context.Context, databaseURL, baseTable string) (*schema.Catalog, error) {
	provider, err := NewPostgresProvider(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	defer provider.Close()

	catalog, err := provider.LoadCatalog(ctx, baseTable)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}
	return catalog, nil
}
