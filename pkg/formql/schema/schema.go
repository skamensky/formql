package schema

import (
	"fmt"
	"strings"
)

// Type is a semantic type in the formula language.
type Type string

const (
	TypeUnknown Type = "unknown"
	TypeNumber  Type = "number"
	TypeString  Type = "string"
	TypeBoolean Type = "boolean"
	TypeDate    Type = "date"
	TypeNull    Type = "null"
)

func (t Type) String() string {
	return string(t)
}

// ParseType normalizes a string to a formula type.
func ParseType(raw string) Type {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "number", "numeric", "float", "int", "integer", "decimal":
		return TypeNumber
	case "string", "text", "varchar":
		return TypeString
	case "boolean", "bool":
		return TypeBoolean
	case "date", "timestamp", "timestamptz":
		return TypeDate
	case "null":
		return TypeNull
	default:
		return TypeUnknown
	}
}

// Column describes a table column.
type Column struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// Table describes a schema table.
type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Relationship describes a direct foreign-key style edge.
type Relationship struct {
	Name                string `json:"name"`
	FromTable           string `json:"from_table"`
	ToTable             string `json:"to_table"`
	JoinColumn          string `json:"join_column"`
	TargetColumn        string `json:"target_column,omitempty"`
	JoinColumnIndexed   *bool  `json:"join_column_indexed,omitempty"`
	TargetColumnIndexed *bool  `json:"target_column_indexed,omitempty"`
}

// Resolver is the compiler-facing schema contract so catalogs can be mocked.
type Resolver interface {
	BaseTableName() string
	Validate() error
	Table(name string) (*Table, bool)
	ColumnType(tableName, columnName string) (Type, bool)
	Relationship(fromTable, relationshipName string) (*Relationship, bool)
}

// Explorer extends Resolver with listing methods useful for tooling.
type Explorer interface {
	Resolver
	ColumnsForTable(name string) []Column
	RelationshipsFrom(tableName string) []Relationship
}

// Catalog is the frontend/backend shared schema model.
type Catalog struct {
	BaseTable     string         `json:"base_table"`
	Tables        []Table        `json:"tables"`
	Relationships []Relationship `json:"relationships"`

	tableIndex map[string]*Table
	colIndex   map[string]map[string]Type
	relIndex   map[string]*Relationship
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func relationshipKey(fromTable, name string) string {
	return normalize(fromTable) + ":" + normalize(name)
}

func (c *Catalog) buildIndexes() {
	if c.tableIndex != nil {
		return
	}

	c.tableIndex = make(map[string]*Table, len(c.Tables))
	c.colIndex = make(map[string]map[string]Type, len(c.Tables))
	c.relIndex = make(map[string]*Relationship, len(c.Relationships))

	for i := range c.Tables {
		table := &c.Tables[i]
		tableKey := normalize(table.Name)
		c.tableIndex[tableKey] = table

		columnIndex := make(map[string]Type, len(table.Columns))
		for _, column := range table.Columns {
			columnIndex[normalize(column.Name)] = ParseType(column.Type.String())
		}
		c.colIndex[tableKey] = columnIndex
	}

	for i := range c.Relationships {
		relationship := &c.Relationships[i]
		c.relIndex[relationshipKey(relationship.FromTable, relationship.Name)] = relationship
	}
}

// Validate checks that the catalog is internally consistent.
func (c *Catalog) Validate() error {
	c.buildIndexes()

	if c.BaseTable == "" {
		return fmt.Errorf("schema catalog is missing base_table")
	}

	base := normalize(c.BaseTable)
	if _, ok := c.tableIndex[base]; !ok {
		return fmt.Errorf("base table %q is not declared in schema", c.BaseTable)
	}

	for _, rel := range c.Relationships {
		fromKey := normalize(rel.FromTable)
		toKey := normalize(rel.ToTable)
		targetColumn := rel.ResolvedTargetColumn()

		if _, ok := c.tableIndex[fromKey]; !ok {
			return fmt.Errorf("relationship %q references unknown from_table %q", rel.Name, rel.FromTable)
		}
		if _, ok := c.tableIndex[toKey]; !ok {
			return fmt.Errorf("relationship %q references unknown to_table %q", rel.Name, rel.ToTable)
		}
		if _, ok := c.colIndex[fromKey][normalize(rel.JoinColumn)]; !ok {
			return fmt.Errorf("relationship %q references missing join column %q on table %q", rel.Name, rel.JoinColumn, rel.FromTable)
		}
		if _, ok := c.colIndex[toKey][normalize(targetColumn)]; !ok {
			return fmt.Errorf("relationship %q references missing target column %q on table %q", rel.Name, targetColumn, rel.ToTable)
		}
	}

	return nil
}

// BaseTableName returns the normalized base table.
func (c *Catalog) BaseTableName() string {
	return normalize(c.BaseTable)
}

// Table returns a table by name.
func (c *Catalog) Table(name string) (*Table, bool) {
	c.buildIndexes()
	table, ok := c.tableIndex[normalize(name)]
	return table, ok
}

// ColumnType looks up a column type by table and column name.
func (c *Catalog) ColumnType(tableName, columnName string) (Type, bool) {
	c.buildIndexes()
	columns, ok := c.colIndex[normalize(tableName)]
	if !ok {
		return TypeUnknown, false
	}
	columnType, ok := columns[normalize(columnName)]
	return columnType, ok
}

// Relationship returns a relationship by source table and relationship name.
func (c *Catalog) Relationship(fromTable, relationshipName string) (*Relationship, bool) {
	c.buildIndexes()
	relationship, ok := c.relIndex[relationshipKey(fromTable, relationshipName)]
	return relationship, ok
}

// ColumnsForTable lists columns for a table. It returns nil when the table is unknown.
func (c *Catalog) ColumnsForTable(name string) []Column {
	c.buildIndexes()
	table, ok := c.tableIndex[normalize(name)]
	if !ok {
		return nil
	}
	columns := make([]Column, len(table.Columns))
	copy(columns, table.Columns)
	return columns
}

// RelationshipsFrom lists direct relationships for a source table.
func (c *Catalog) RelationshipsFrom(tableName string) []Relationship {
	c.buildIndexes()
	normalized := normalize(tableName)
	relationships := make([]Relationship, 0)
	for _, relationship := range c.Relationships {
		if normalize(relationship.FromTable) == normalized {
			relationships = append(relationships, relationship)
		}
	}
	return relationships
}

// ResolvedTargetColumn returns the relationship target column or the default "id".
func (r Relationship) ResolvedTargetColumn() string {
	if strings.TrimSpace(r.TargetColumn) == "" {
		return "id"
	}
	return r.TargetColumn
}

// Compatible returns whether two types can coexist in a single semantic slot.
func Compatible(left, right Type) bool {
	return left == right || left == TypeNull || right == TypeNull
}

// Unify collapses a list of types into a single semantic type.
func Unify(types ...Type) (Type, error) {
	result := TypeNull
	for _, current := range types {
		if current == TypeUnknown {
			return TypeUnknown, nil
		}
		if result == TypeNull {
			result = current
			continue
		}
		if current == TypeNull {
			continue
		}
		if result == TypeNull {
			result = current
			continue
		}
		if result != current {
			return TypeUnknown, fmt.Errorf("cannot unify types %s and %s", result, current)
		}
	}

	if result == TypeNull {
		return TypeNull, nil
	}
	return result, nil
}
