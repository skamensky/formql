package semantics

import (
	"fmt"
	"strings"

	"github.com/skamensky/formql/pkg/formql/ast"
	"github.com/skamensky/formql/pkg/formql/builtin"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/ir"
	"github.com/skamensky/formql/pkg/formql/schema"
)

const DefaultMaxRelationshipDepth = 30

// Options configures semantic lowering behavior.
type Options struct {
	MaxRelationshipDepth int
}

// Lower converts AST into typed semantic IR.
func Lower(node ast.Expr, catalog schema.Resolver) (*ir.Plan, error) {
	return LowerWithOptions(node, catalog, Options{})
}

// LowerWithOptions converts AST into typed semantic IR with explicit options.
func LowerWithOptions(node ast.Expr, catalog schema.Resolver, options Options) (*ir.Plan, error) {
	l, err := newLowerer(catalog)
	if err != nil {
		return nil, err
	}
	l.maxRelationshipDepth = normalizedMaxRelationshipDepth(options.MaxRelationshipDepth)

	expr, err := l.lowerExpr(node)
	if err != nil {
		return nil, err
	}

	return &ir.Plan{
		BaseTable:  l.baseTable,
		BaseSchema: l.baseSchema,
		Expr:       expr,
		Joins:      l.orderedJoins(),
		Warnings:   l.orderedWarnings(),
	}, nil
}

// LowerDocument converts a top-level document AST into typed semantic IR.
func LowerDocument(document *ast.Document, catalog schema.Resolver) (*ir.DocumentPlan, error) {
	return LowerDocumentWithOptions(document, catalog, Options{})
}

// LowerDocumentWithOptions converts a top-level document AST into typed semantic IR with explicit options.
func LowerDocumentWithOptions(document *ast.Document, catalog schema.Resolver, options Options) (*ir.DocumentPlan, error) {
	if document == nil {
		return nil, fmt.Errorf("document is required")
	}
	if len(document.Items) == 0 {
		return nil, diagnostic.New("semantics", "empty document", document.Pos)
	}

	l, err := newLowerer(catalog)
	if err != nil {
		return nil, err
	}
	l.maxRelationshipDepth = normalizedMaxRelationshipDepth(options.MaxRelationshipDepth)

	fields := make([]ir.SelectField, 0, len(document.Items))
	usedAliases := make(map[string]bool, len(document.Items))
	for _, item := range document.Items {
		expr, err := l.lowerExpr(item.Expr)
		if err != nil {
			return nil, err
		}

		alias, err := selectAlias(item, usedAliases)
		if err != nil {
			return nil, err
		}

		fields = append(fields, ir.SelectField{
			Alias:      alias,
			ResultType: expr.Type(),
			Expr:       expr,
		})
	}

	return &ir.DocumentPlan{
		BaseTable:  l.baseTable,
		BaseSchema: l.baseSchema,
		Fields:     fields,
		Joins:      l.orderedJoins(),
		Warnings:   l.orderedWarnings(),
	}, nil
}

func newLowerer(catalog schema.Resolver) (*lowerer, error) {
	if catalog == nil {
		return nil, fmt.Errorf("schema catalog is required")
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	baseTable := strings.ToLower(catalog.BaseTableName())
	baseSchema := ""
	if sp, ok := catalog.(schemaProvider); ok {
		baseSchema = sp.SchemaFor(baseTable)
	}

	l := &lowerer{
		catalog:    catalog,
		baseTable:  baseTable,
		baseSchema: baseSchema,
		joins:      make(map[string]ir.Join),
		order:      make([]string, 0, 4),
		warnings:   make(map[string]diagnostic.Warning),
		warnOrder:  make([]string, 0, 2),
	}

	return l, nil
}

func (l *lowerer) orderedJoins() []ir.Join {
	joins := make([]ir.Join, 0, len(l.order))
	for _, key := range l.order {
		joins = append(joins, l.joins[key])
	}
	return joins
}

func (l *lowerer) orderedWarnings() []diagnostic.Warning {
	warnings := make([]diagnostic.Warning, 0, len(l.warnOrder))
	for _, key := range l.warnOrder {
		warnings = append(warnings, l.warnings[key])
	}
	return warnings
}

// schemaProvider is satisfied by *schema.Catalog so the lowerer can resolve table schemas
// without changing the Resolver interface.
type schemaProvider interface {
	SchemaFor(tableName string) string
}

type lowerer struct {
	catalog              schema.Resolver
	baseTable            string
	baseSchema           string
	maxRelationshipDepth int
	joins                map[string]ir.Join
	order                []string
	warnings             map[string]diagnostic.Warning
	warnOrder            []string
}

func selectAlias(item ast.SelectItem, used map[string]bool) (string, error) {
	explicit := strings.TrimSpace(item.Alias) != ""
	alias := strings.ToLower(strings.TrimSpace(item.Alias))
	if alias == "" {
		alias = defaultAlias(item.Expr)
	}
	if alias == "" {
		alias = "result"
	}

	if explicit {
		if used[alias] {
			position := item.AliasPos
			if position == 0 {
				position = item.Pos
			}
			return "", diagnostic.NewError("semantics", "duplicate_output_alias", fmt.Sprintf("duplicate output alias %q", alias), "choose a unique alias for each selected field", position)
		}
		used[alias] = true
		return alias, nil
	}

	alias = uniqueAlias(alias, used)
	used[alias] = true
	return alias, nil
}

func defaultAlias(node ast.Expr) string {
	switch n := node.(type) {
	case *ast.Identifier:
		return n.Name
	case *ast.RelationshipRef:
		parts := make([]string, 0, len(n.Chain)+1)
		parts = append(parts, n.Chain...)
		parts = append(parts, n.Field)
		return strings.Join(parts, "_")
	default:
		return "result"
	}
}

func uniqueAlias(base string, used map[string]bool) string {
	if !used[base] {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s_%d", base, suffix)
		if !used[candidate] {
			return candidate
		}
	}
}

func (l *lowerer) lowerExpr(node ast.Expr) (ir.Expr, error) {
	switch n := node.(type) {
	case *ast.NumberLiteral:
		return &ir.Literal{Kind: "literal", ResultType: schema.TypeNumber, Value: n.Value}, nil
	case *ast.StringLiteral:
		return &ir.Literal{Kind: "literal", ResultType: schema.TypeString, Value: n.Value}, nil
	case *ast.BooleanLiteral:
		return &ir.Literal{Kind: "literal", ResultType: schema.TypeBoolean, Value: n.Value}, nil
	case *ast.NullLiteral:
		return &ir.Literal{Kind: "literal", ResultType: schema.TypeNull, Value: nil}, nil
	case *ast.Identifier:
		columnType, ok := l.catalog.ColumnType(l.baseTable, n.Name)
		if !ok {
			hint := "check the base table schema for the correct column name"
			if suggestion, ok := l.suggestColumn(l.baseTable, n.Name); ok {
				hint = fmt.Sprintf("did you mean %q?", suggestion)
			}
			return nil, diagnostic.NewError("semantics", "unknown_column", fmt.Sprintf("unknown column %q on table %q", n.Name, l.baseTable), hint, n.Pos)
		}
		return &ir.FieldRef{
			Kind:       "field_ref",
			ResultType: columnType,
			Table:      l.baseTable,
			Column:     n.Name,
		}, nil
	case *ast.RelationshipRef:
		return l.lowerRelationshipRef(n)
	case *ast.UnaryExpr:
		operand, err := l.lowerExpr(n.Operand)
		if err != nil {
			return nil, err
		}
		if n.Op != "-" {
			return nil, diagnostic.NewError("semantics", "unsupported_unary_operator", "unsupported unary operator "+n.Op, "use '-' for numeric negation", n.Pos)
		}
		if !allowsType(operand.Type(), schema.TypeNumber) {
			return nil, diagnostic.NewError("semantics", "invalid_unary_operand", fmt.Sprintf("unary '-' requires a number, got %s", operand.Type()), "convert the value to a number before negating it", n.Pos)
		}
		return &ir.UnaryExpr{
			Kind:       "unary_expr",
			ResultType: schema.TypeNumber,
			Op:         n.Op,
			Operand:    operand,
		}, nil
	case *ast.BinaryExpr:
		left, err := l.lowerExpr(n.Left)
		if err != nil {
			return nil, err
		}
		right, err := l.lowerExpr(n.Right)
		if err != nil {
			return nil, err
		}
		resultType, err := binaryResultType(n.Op, left.Type(), right.Type())
		if err != nil {
			return nil, semanticBinaryError(n.Op, left.Type(), right.Type(), n.Pos)
		}
		return &ir.BinaryExpr{
			Kind:       "binary_expr",
			ResultType: resultType,
			Op:         n.Op,
			Left:       left,
			Right:      right,
		}, nil
	case *ast.CallExpr:
		return l.lowerCall(n)
	default:
		return nil, diagnostic.NewError("semantics", "unknown_ast_node", "unknown AST node", "", node.Position())
	}
}

func (l *lowerer) lowerRelationshipRef(node *ast.RelationshipRef) (ir.Expr, error) {
	if len(node.Chain) > l.maxRelationshipDepth {
		return nil, diagnostic.NewError("semantics", "relationship_depth_exceeded", fmt.Sprintf("relationship path depth %d exceeds max depth %d", len(node.Chain), l.maxRelationshipDepth), "shorten the path or raise max_relationship_depth in the host configuration", node.Pos)
	}

	currentTable := l.baseTable
	path := make([]string, 0, len(node.Chain))

	for _, relationshipName := range node.Chain {
		relationship, ok := l.catalog.Relationship(currentTable, relationshipName)
		if !ok {
			hint := "check the catalog relationship names for this table"
			if suggestion, ok := l.suggestRelationship(currentTable, relationshipName); ok {
				hint = fmt.Sprintf("did you mean %q?", suggestion)
			}
			return nil, diagnostic.NewError("semantics", "unknown_relationship", fmt.Sprintf("unknown relationship %q from table %q", relationshipName, currentTable), hint, node.Pos)
		}
		path = append(path, strings.ToLower(relationshipName))
		l.addJoin(path, relationship, node.Pos)
		currentTable = strings.ToLower(relationship.ToTable)
	}

	fieldType, ok := l.catalog.ColumnType(currentTable, node.Field)
	if !ok {
		hint := "check the related table schema for the correct field name"
		if suggestion, ok := l.suggestColumn(currentTable, node.Field); ok {
			hint = fmt.Sprintf("did you mean %q?", suggestion)
		}
		return nil, diagnostic.NewError("semantics", "unknown_related_field", fmt.Sprintf("unknown field %q on related table %q", node.Field, currentTable), hint, node.Pos)
	}

	return &ir.FieldRef{
		Kind:       "field_ref",
		ResultType: fieldType,
		Path:       append([]string(nil), path...),
		Table:      currentTable,
		Column:     node.Field,
	}, nil
}

func normalizedMaxRelationshipDepth(value int) int {
	if value <= 0 {
		return DefaultMaxRelationshipDepth
	}
	return value
}

func (l *lowerer) addJoin(path []string, relationship *schema.Relationship, position int) {
	key := strings.Join(path, ".")
	if _, ok := l.joins[key]; ok {
		return
	}
	toTable := strings.ToLower(relationship.ToTable)
	toSchema := ""
	if sp, ok := l.catalog.(schemaProvider); ok {
		toSchema = sp.SchemaFor(toTable)
	}
	l.joins[key] = ir.Join{
		Key:          key,
		Path:         append([]string(nil), path...),
		FromTable:    strings.ToLower(relationship.FromTable),
		ToTable:      toTable,
		ToSchema:     toSchema,
		JoinColumn:   strings.ToLower(relationship.JoinColumn),
		TargetColumn: strings.ToLower(relationship.ResolvedTargetColumn()),
	}
	l.order = append(l.order, key)

	if relationship.JoinColumnIndexed != nil && !*relationship.JoinColumnIndexed {
		l.addWarning(key+":join-column", diagnostic.NewWarning("semantics", "non_indexed_join_source", fmt.Sprintf("join path %q uses non-indexed source column %q on table %q", key, relationship.JoinColumn, relationship.FromTable), fmt.Sprintf("add an index on %s.%s if this path is performance-sensitive", relationship.FromTable, relationship.JoinColumn), position))
	}
	if relationship.TargetColumnIndexed != nil && !*relationship.TargetColumnIndexed {
		l.addWarning(key+":target-column", diagnostic.NewWarning("semantics", "non_indexed_join_target", fmt.Sprintf("join path %q uses non-indexed target column %q on table %q", key, relationship.ResolvedTargetColumn(), relationship.ToTable), fmt.Sprintf("add an index on %s.%s if this path is performance-sensitive", relationship.ToTable, relationship.ResolvedTargetColumn()), position))
	}
}

func (l *lowerer) addWarning(key string, warning diagnostic.Warning) {
	if _, ok := l.warnings[key]; ok {
		return
	}
	l.warnings[key] = warning
	l.warnOrder = append(l.warnOrder, key)
}

func (l *lowerer) lowerCall(node *ast.CallExpr) (ir.Expr, error) {
	args := make([]ir.Expr, 0, len(node.Args))
	for _, arg := range node.Args {
		lowered, err := l.lowerExpr(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, lowered)
	}

	name := strings.ToUpper(node.Name)
	resultType, err := functionResultType(name, args)
	if err != nil {
		return nil, semanticFunctionError(name, args, err, node.Pos)
	}

	return &ir.CallExpr{
		Kind:       "call_expr",
		ResultType: resultType,
		Name:       name,
		Args:       args,
	}, nil
}

func (l *lowerer) suggestColumn(tableName, columnName string) (string, bool) {
	explorer, ok := l.catalog.(schema.Explorer)
	if !ok {
		return "", false
	}
	names := make([]string, 0, len(explorer.ColumnsForTable(tableName)))
	for _, column := range explorer.ColumnsForTable(tableName) {
		names = append(names, column.Name)
	}
	return diagnostic.ClosestSuggestion(columnName, names)
}

func (l *lowerer) suggestRelationship(tableName, relationshipName string) (string, bool) {
	explorer, ok := l.catalog.(schema.Explorer)
	if !ok {
		return "", false
	}
	names := make([]string, 0, len(explorer.RelationshipsFrom(tableName)))
	for _, relationship := range explorer.RelationshipsFrom(tableName) {
		names = append(names, relationship.Name)
	}
	return diagnostic.ClosestSuggestion(relationshipName, names)
}

func allowsType(actual schema.Type, allowed ...schema.Type) bool {
	if actual == schema.TypeNull || actual == schema.TypeUnknown {
		return true
	}
	for _, candidate := range allowed {
		if actual == candidate {
			return true
		}
	}
	return false
}

func binaryResultType(op string, left, right schema.Type) (schema.Type, error) {
	switch op {
	case "+":
		switch {
		case allowsType(left, schema.TypeNumber) && allowsType(right, schema.TypeNumber):
			return schema.TypeNumber, nil
		case allowsType(left, schema.TypeDate) && allowsType(right, schema.TypeNumber):
			return schema.TypeDate, nil
		case allowsType(left, schema.TypeNumber) && allowsType(right, schema.TypeDate):
			return schema.TypeDate, nil
		default:
			return schema.TypeUnknown, fmt.Errorf("operator '+' requires number+number or date+number, got %s and %s", left, right)
		}
	case "-":
		switch {
		case allowsType(left, schema.TypeNumber) && allowsType(right, schema.TypeNumber):
			return schema.TypeNumber, nil
		case allowsType(left, schema.TypeDate) && allowsType(right, schema.TypeNumber):
			return schema.TypeDate, nil
		case allowsType(left, schema.TypeDate) && allowsType(right, schema.TypeDate):
			return schema.TypeNumber, nil
		default:
			return schema.TypeUnknown, fmt.Errorf("operator '-' requires number-number, date-number, or date-date, got %s and %s", left, right)
		}
	case "*", "/":
		if allowsType(left, schema.TypeNumber) && allowsType(right, schema.TypeNumber) {
			return schema.TypeNumber, nil
		}
		return schema.TypeUnknown, fmt.Errorf("operator %q requires numeric operands, got %s and %s", op, left, right)
	case "&":
		if allowsType(left, schema.TypeString) && allowsType(right, schema.TypeString) {
			return schema.TypeString, nil
		}
		return schema.TypeUnknown, fmt.Errorf("operator '&' requires string operands, got %s and %s", left, right)
	case "=", "!=", "<>", ">", ">=", "<", "<=":
		if schema.Compatible(left, right) {
			return schema.TypeBoolean, nil
		}
		return schema.TypeUnknown, fmt.Errorf("cannot compare %s and %s", left, right)
	default:
		return schema.TypeUnknown, fmt.Errorf("unknown operator %q", op)
	}
}

func semanticBinaryError(op string, left, right schema.Type, position int) error {
	switch op {
	case "&":
		return diagnostic.NewError("semantics", "invalid_concat_operands", fmt.Sprintf("operator '&' requires string operands, got %s and %s", left, right), "wrap non-string values with STRING(...) before concatenating", position)
	case "*", "/":
		return diagnostic.NewError("semantics", "invalid_numeric_operands", fmt.Sprintf("operator %q requires numeric operands, got %s and %s", op, left, right), "convert both operands to numbers before using arithmetic operators", position)
	case "+", "-":
		return diagnostic.NewError("semantics", "invalid_operator_operands", binaryOperatorMessage(op, left, right), binaryOperatorHint(op), position)
	case "=", "!=", "<>", ">", ">=", "<", "<=":
		return diagnostic.NewError("semantics", "invalid_comparison_operands", fmt.Sprintf("cannot compare %s and %s", left, right), "compare values of the same type or cast one side first", position)
	default:
		return diagnostic.NewError("semantics", "unknown_operator", fmt.Sprintf("unknown operator %q", op), "", position)
	}
}

func binaryOperatorMessage(op string, left, right schema.Type) string {
	switch op {
	case "+":
		return fmt.Sprintf("operator '+' requires number+number or date+number, got %s and %s", left, right)
	case "-":
		return fmt.Sprintf("operator '-' requires number-number, date-number, or date-date, got %s and %s", left, right)
	default:
		return fmt.Sprintf("invalid operands for operator %q: %s and %s", op, left, right)
	}
}

func binaryOperatorHint(op string) string {
	switch op {
	case "+":
		return "use '+' for numeric math or date arithmetic only"
	case "-":
		return "use '-' for numeric subtraction or date arithmetic only"
	default:
		return ""
	}
}

func semanticFunctionError(name string, args []ir.Expr, cause error, position int) error {
	spec, hasSpec := builtin.Lookup(name)
	signatureHint := ""
	if hasSpec {
		signatureHint = "expected signature: " + spec.Signature
	}

	switch name {
	case "IF":
		if len(args) != 3 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "IF expects 3 arguments", signatureHint, position)
		}
		if !allowsType(args[0].Type(), schema.TypeBoolean) {
			return diagnostic.NewError("semantics", "invalid_if_condition", fmt.Sprintf("IF condition must be boolean, got %s", args[0].Type()), "make the first argument a comparison or boolean expression", position)
		}
		return diagnostic.NewError("semantics", "incompatible_if_branches", cause.Error(), "make both IF branches return the same type, or cast one branch", position)
	case "AND", "OR":
		if len(args) < 2 {
			return diagnostic.NewError("semantics", "invalid_function_arity", fmt.Sprintf("%s expects at least 2 arguments", name), signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_boolean_arguments", fmt.Sprintf("%s expects boolean arguments", name), "use comparisons like amount > 0 inside boolean functions", position)
	case "NOT":
		if len(args) != 1 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "NOT expects 1 argument", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_boolean_argument", fmt.Sprintf("NOT expects a boolean argument, got %s", args[0].Type()), "use a comparison or boolean expression", position)
	case "STRING":
		return diagnostic.NewError("semantics", "invalid_function_arity", "STRING expects 1 argument", signatureHint, position)
	case "DATE":
		if len(args) != 1 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "DATE expects 1 argument", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_date_argument", fmt.Sprintf("DATE expects a string or date argument, got %s", args[0].Type()), "pass a date column or a string that can be parsed as a date", position)
	case "COALESCE":
		if len(args) < 1 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "COALESCE expects at least 1 argument", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "incompatible_coalesce_arguments", cause.Error(), "make COALESCE arguments compatible types", position)
	case "NULLVALUE":
		if len(args) != 2 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "NULLVALUE expects 2 arguments", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "incompatible_nullvalue_arguments", cause.Error(), "make NULLVALUE arguments compatible types", position)
	case "ISNULL", "ISBLANK":
		return diagnostic.NewError("semantics", "invalid_function_arity", fmt.Sprintf("%s expects 1 argument", name), signatureHint, position)
	case "TODAY":
		return diagnostic.NewError("semantics", "invalid_function_arity", "TODAY expects 0 arguments", signatureHint, position)
	case "ABS":
		if len(args) != 1 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "ABS expects 1 argument", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_numeric_argument", fmt.Sprintf("ABS expects a numeric argument, got %s", args[0].Type()), "pass a numeric value", position)
	case "ROUND":
		if len(args) != 1 && len(args) != 2 {
			return diagnostic.NewError("semantics", "invalid_function_arity", "ROUND expects 1 or 2 arguments", signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_numeric_argument", cause.Error(), "ROUND expects numeric arguments", position)
	case "LEN", "UPPER", "LOWER", "TRIM":
		if len(args) != 1 {
			return diagnostic.NewError("semantics", "invalid_function_arity", fmt.Sprintf("%s expects 1 argument", name), signatureHint, position)
		}
		return diagnostic.NewError("semantics", "invalid_string_argument", cause.Error(), "wrap the value with STRING(...) if you need string behavior", position)
	default:
		hint := "check the builtin function name"
		if suggestion, ok := builtin.Suggest(name); ok {
			hint = fmt.Sprintf("did you mean %s?", suggestion.Signature)
		}
		return diagnostic.NewError("semantics", "unknown_function", fmt.Sprintf("unknown function %q", name), hint, position)
	}
}

func functionResultType(name string, args []ir.Expr) (schema.Type, error) {
	switch name {
	case "IF":
		if len(args) != 3 {
			return schema.TypeUnknown, fmt.Errorf("IF expects 3 arguments")
		}
		if !allowsType(args[0].Type(), schema.TypeBoolean) {
			return schema.TypeUnknown, fmt.Errorf("IF condition must be boolean, got %s", args[0].Type())
		}
		resultType, err := schema.Unify(args[1].Type(), args[2].Type())
		if err != nil {
			return schema.TypeUnknown, fmt.Errorf("IF branch types are incompatible: %w", err)
		}
		return resultType, nil
	case "AND", "OR":
		if len(args) < 2 {
			return schema.TypeUnknown, fmt.Errorf("%s expects at least 2 arguments", name)
		}
		for _, arg := range args {
			if !allowsType(arg.Type(), schema.TypeBoolean) {
				return schema.TypeUnknown, fmt.Errorf("%s expects boolean arguments, got %s", name, arg.Type())
			}
		}
		return schema.TypeBoolean, nil
	case "NOT":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("NOT expects 1 argument")
		}
		if !allowsType(args[0].Type(), schema.TypeBoolean) {
			return schema.TypeUnknown, fmt.Errorf("NOT expects a boolean argument, got %s", args[0].Type())
		}
		return schema.TypeBoolean, nil
	case "STRING":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("STRING expects 1 argument")
		}
		return schema.TypeString, nil
	case "DATE":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("DATE expects 1 argument")
		}
		if !allowsType(args[0].Type(), schema.TypeString, schema.TypeDate) {
			return schema.TypeUnknown, fmt.Errorf("DATE expects a string or date argument, got %s", args[0].Type())
		}
		return schema.TypeDate, nil
	case "COALESCE":
		if len(args) < 1 {
			return schema.TypeUnknown, fmt.Errorf("COALESCE expects at least 1 argument")
		}
		types := make([]schema.Type, 0, len(args))
		for _, arg := range args {
			types = append(types, arg.Type())
		}
		resultType, err := schema.Unify(types...)
		if err != nil {
			return schema.TypeUnknown, fmt.Errorf("COALESCE arguments are incompatible: %w", err)
		}
		return resultType, nil
	case "NULLVALUE":
		if len(args) != 2 {
			return schema.TypeUnknown, fmt.Errorf("NULLVALUE expects 2 arguments")
		}
		resultType, err := schema.Unify(args[0].Type(), args[1].Type())
		if err != nil {
			return schema.TypeUnknown, fmt.Errorf("NULLVALUE arguments are incompatible: %w", err)
		}
		return resultType, nil
	case "ISNULL", "ISBLANK":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("%s expects 1 argument", name)
		}
		return schema.TypeBoolean, nil
	case "TODAY":
		if len(args) != 0 {
			return schema.TypeUnknown, fmt.Errorf("TODAY expects 0 arguments")
		}
		return schema.TypeDate, nil
	case "ABS":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("ABS expects 1 argument")
		}
		if !allowsType(args[0].Type(), schema.TypeNumber) {
			return schema.TypeUnknown, fmt.Errorf("ABS expects a numeric argument, got %s", args[0].Type())
		}
		return schema.TypeNumber, nil
	case "ROUND":
		if len(args) != 1 && len(args) != 2 {
			return schema.TypeUnknown, fmt.Errorf("ROUND expects 1 or 2 arguments")
		}
		if !allowsType(args[0].Type(), schema.TypeNumber) {
			return schema.TypeUnknown, fmt.Errorf("ROUND expects a numeric first argument, got %s", args[0].Type())
		}
		if len(args) == 2 && !allowsType(args[1].Type(), schema.TypeNumber) {
			return schema.TypeUnknown, fmt.Errorf("ROUND expects a numeric precision argument, got %s", args[1].Type())
		}
		return schema.TypeNumber, nil
	case "LEN":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("LEN expects 1 argument")
		}
		if !allowsType(args[0].Type(), schema.TypeString) {
			return schema.TypeUnknown, fmt.Errorf("LEN expects a string argument, got %s", args[0].Type())
		}
		return schema.TypeNumber, nil
	case "UPPER", "LOWER", "TRIM":
		if len(args) != 1 {
			return schema.TypeUnknown, fmt.Errorf("%s expects 1 argument", name)
		}
		if !allowsType(args[0].Type(), schema.TypeString) {
			return schema.TypeUnknown, fmt.Errorf("%s expects a string argument, got %s", name, args[0].Type())
		}
		return schema.TypeString, nil
	default:
		return schema.TypeUnknown, fmt.Errorf("unknown function %q", name)
	}
}
