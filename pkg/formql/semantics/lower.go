package semantics

import (
	"fmt"
	"strings"

	"github.com/skamensky/formql/pkg/formql/ast"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/ir"
	"github.com/skamensky/formql/pkg/formql/schema"
)

// Lower converts AST into typed semantic IR.
func Lower(node ast.Expr, catalog schema.Resolver) (*ir.Plan, error) {
	if catalog == nil {
		return nil, fmt.Errorf("schema catalog is required")
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	l := &lowerer{
		catalog:   catalog,
		baseTable: strings.ToLower(catalog.BaseTableName()),
		joins:     make(map[string]ir.Join),
		order:     make([]string, 0, 4),
		warnings:  make(map[string]diagnostic.Warning),
		warnOrder: make([]string, 0, 2),
	}

	expr, err := l.lowerExpr(node)
	if err != nil {
		return nil, err
	}

	joins := make([]ir.Join, 0, len(l.order))
	for _, key := range l.order {
		joins = append(joins, l.joins[key])
	}

	warnings := make([]diagnostic.Warning, 0, len(l.warnOrder))
	for _, key := range l.warnOrder {
		warnings = append(warnings, l.warnings[key])
	}

	return &ir.Plan{
		BaseTable: l.baseTable,
		Expr:      expr,
		Joins:     joins,
		Warnings:  warnings,
	}, nil
}

type lowerer struct {
	catalog   schema.Resolver
	baseTable string
	joins     map[string]ir.Join
	order     []string
	warnings  map[string]diagnostic.Warning
	warnOrder []string
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
			return nil, diagnostic.New("semantics", fmt.Sprintf("unknown column %q on table %q", n.Name, l.baseTable), n.Pos)
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
			return nil, diagnostic.New("semantics", "unsupported unary operator "+n.Op, n.Pos)
		}
		if !allowsType(operand.Type(), schema.TypeNumber) {
			return nil, diagnostic.New("semantics", fmt.Sprintf("unary '-' requires a number, got %s", operand.Type()), n.Pos)
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
			return nil, diagnostic.New("semantics", err.Error(), n.Pos)
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
		return nil, diagnostic.New("semantics", "unknown AST node", node.Position())
	}
}

func (l *lowerer) lowerRelationshipRef(node *ast.RelationshipRef) (ir.Expr, error) {
	currentTable := l.baseTable
	path := make([]string, 0, len(node.Chain))

	for _, relationshipName := range node.Chain {
		relationship, ok := l.catalog.Relationship(currentTable, relationshipName)
		if !ok {
			return nil, diagnostic.New("semantics", fmt.Sprintf("unknown relationship %q from table %q", relationshipName, currentTable), node.Pos)
		}
		path = append(path, strings.ToLower(relationshipName))
		l.addJoin(path, relationship, node.Pos)
		currentTable = strings.ToLower(relationship.ToTable)
	}

	fieldType, ok := l.catalog.ColumnType(currentTable, node.Field)
	if !ok {
		return nil, diagnostic.New("semantics", fmt.Sprintf("unknown field %q on related table %q", node.Field, currentTable), node.Pos)
	}

	return &ir.FieldRef{
		Kind:       "field_ref",
		ResultType: fieldType,
		Path:       append([]string(nil), path...),
		Table:      currentTable,
		Column:     node.Field,
	}, nil
}

func (l *lowerer) addJoin(path []string, relationship *schema.Relationship, position int) {
	key := strings.Join(path, ".")
	if _, ok := l.joins[key]; ok {
		return
	}
	l.joins[key] = ir.Join{
		Key:          key,
		Path:         append([]string(nil), path...),
		FromTable:    strings.ToLower(relationship.FromTable),
		ToTable:      strings.ToLower(relationship.ToTable),
		JoinColumn:   strings.ToLower(relationship.JoinColumn),
		TargetColumn: strings.ToLower(relationship.ResolvedTargetColumn()),
	}
	l.order = append(l.order, key)

	if relationship.JoinColumnIndexed != nil && !*relationship.JoinColumnIndexed {
		l.addWarning(key+":join-column", diagnostic.Warning{
			Stage:    "semantics",
			Message:  fmt.Sprintf("join path %q uses non-indexed source column %q on table %q", key, relationship.JoinColumn, relationship.FromTable),
			Position: position,
		})
	}
	if relationship.TargetColumnIndexed != nil && !*relationship.TargetColumnIndexed {
		l.addWarning(key+":target-column", diagnostic.Warning{
			Stage:    "semantics",
			Message:  fmt.Sprintf("join path %q uses non-indexed target column %q on table %q", key, relationship.ResolvedTargetColumn(), relationship.ToTable),
			Position: position,
		})
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
		return nil, diagnostic.New("semantics", err.Error(), node.Pos)
	}

	return &ir.CallExpr{
		Kind:       "call_expr",
		ResultType: resultType,
		Name:       name,
		Args:       args,
	}, nil
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
