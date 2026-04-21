package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/skamensky/formql/pkg/formql/ir"
	"github.com/skamensky/formql/pkg/formql/schema"
)

// Artifact is the SQL result of lowering a formula IR plan to PostgreSQL.
type Artifact struct {
	Expression  string   `json:"expression"`
	Query       string   `json:"query"`
	JoinClauses []string `json:"joins"`
}

// Renderer lowers semantic IR to PostgreSQL SQL.
type Renderer struct{}

// Render renders a semantic plan into a scalar SQL expression and a full SELECT query.
func (r Renderer) Render(plan *ir.Plan, fieldAlias string) (Artifact, error) {
	if plan == nil {
		return Artifact{}, fmt.Errorf("plan is required")
	}
	if fieldAlias == "" {
		fieldAlias = "result"
	}

	expression, err := r.renderExpr(plan.Expr)
	if err != nil {
		return Artifact{}, err
	}

	fromLine := fmt.Sprintf("FROM %s t0", quoteIdent(plan.BaseTable))
	joinClauses := make([]string, 0, len(plan.Joins))

	for _, join := range plan.Joins {
		parentAlias := "t0"
		if len(join.Path) > 1 {
			parentAlias = aliasForPath(join.Path[:len(join.Path)-1])
		}
		alias := aliasForPath(join.Path)
		joinClauses = append(joinClauses,
			fmt.Sprintf(
				"LEFT JOIN %s %s ON %s.%s = %s.%s",
				quoteIdent(join.ToTable),
				alias,
				parentAlias,
				quoteIdent(join.JoinColumn),
				alias,
				quoteIdent(join.TargetColumn),
			),
		)
	}

	lines := []string{
		fmt.Sprintf("SELECT %s AS %s", expression, quoteIdent(fieldAlias)),
		fromLine,
	}
	lines = append(lines, joinClauses...)

	return Artifact{
		Expression:  expression,
		Query:       strings.Join(lines, "\n"),
		JoinClauses: joinClauses,
	}, nil
}

func (Renderer) renderExpr(node ir.Expr) (string, error) {
	switch n := node.(type) {
	case *ir.Literal:
		return renderLiteral(n)
	case *ir.FieldRef:
		alias := "t0"
		if len(n.Path) > 0 {
			alias = aliasForPath(n.Path)
		}
		return fmt.Sprintf("%s.%s", alias, quoteIdent(n.Column)), nil
	case *ir.UnaryExpr:
		operand, err := (Renderer{}).renderExpr(n.Operand)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(-%s)", operand), nil
	case *ir.BinaryExpr:
		return renderBinary(n)
	case *ir.CallExpr:
		return renderCall(n)
	default:
		return "", fmt.Errorf("unsupported IR node %T", node)
	}
}

func renderLiteral(literal *ir.Literal) (string, error) {
	switch literal.ResultType {
	case schema.TypeNumber:
		value, ok := literal.Value.(float64)
		if !ok {
			return "", fmt.Errorf("numeric literal has unexpected type %T", literal.Value)
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case schema.TypeString:
		value, ok := literal.Value.(string)
		if !ok {
			return "", fmt.Errorf("string literal has unexpected type %T", literal.Value)
		}
		return "'" + strings.ReplaceAll(value, "'", "''") + "'", nil
	case schema.TypeBoolean:
		value, ok := literal.Value.(bool)
		if !ok {
			return "", fmt.Errorf("boolean literal has unexpected type %T", literal.Value)
		}
		if value {
			return "TRUE", nil
		}
		return "FALSE", nil
	case schema.TypeNull:
		return "NULL", nil
	default:
		return "", fmt.Errorf("unsupported literal type %s", literal.ResultType)
	}
}

func renderBinary(node *ir.BinaryExpr) (string, error) {
	left, err := (Renderer{}).renderExpr(node.Left)
	if err != nil {
		return "", err
	}
	right, err := (Renderer{}).renderExpr(node.Right)
	if err != nil {
		return "", err
	}

	switch node.Op {
	case "=":
		return fmt.Sprintf("(%s IS NOT DISTINCT FROM %s)", left, right), nil
	case "!=", "<>":
		return fmt.Sprintf("(%s IS DISTINCT FROM %s)", left, right), nil
	case "&":
		return fmt.Sprintf("(%s || %s)", left, right), nil
	default:
		return fmt.Sprintf("(%s %s %s)", left, node.Op, right), nil
	}
}

func renderCall(node *ir.CallExpr) (string, error) {
	args, err := renderArgs(node.Args)
	if err != nil {
		return "", err
	}

	switch node.Name {
	case "IF":
		return fmt.Sprintf("CASE WHEN %s THEN %s ELSE %s END", args[0], args[1], args[2]), nil
	case "AND":
		return "(" + strings.Join(args, " AND ") + ")", nil
	case "OR":
		return "(" + strings.Join(args, " OR ") + ")", nil
	case "NOT":
		return "(NOT " + args[0] + ")", nil
	case "STRING":
		return "CAST(" + args[0] + " AS TEXT)", nil
	case "DATE":
		return "CAST(" + args[0] + " AS DATE)", nil
	case "COALESCE", "NULLVALUE":
		return "COALESCE(" + strings.Join(args, ", ") + ")", nil
	case "ISNULL":
		return "(" + args[0] + " IS NULL)", nil
	case "ISBLANK":
		if len(node.Args) == 1 && node.Args[0].Type() != schema.TypeString && node.Args[0].Type() != schema.TypeNull {
			return "(" + args[0] + " IS NULL)", nil
		}
		return "((" + args[0] + " IS NULL) OR (" + args[0] + " = ''))", nil
	case "TODAY":
		return "CURRENT_DATE", nil
	case "ABS":
		return "ABS(" + args[0] + ")", nil
	case "ROUND":
		return "ROUND(" + strings.Join(args, ", ") + ")", nil
	case "LEN":
		return "CHAR_LENGTH(" + args[0] + ")", nil
	case "UPPER", "LOWER", "TRIM":
		return node.Name + "(" + args[0] + ")", nil
	default:
		return "", fmt.Errorf("unsupported Postgres function %q", node.Name)
	}
}

func renderArgs(args []ir.Expr) ([]string, error) {
	rendered := make([]string, 0, len(args))
	for _, arg := range args {
		sql, err := (Renderer{}).renderExpr(arg)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, sql)
	}
	return rendered, nil
}

func aliasForPath(path []string) string {
	return "rel_" + strings.Join(path, "_")
}

func quoteIdent(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
