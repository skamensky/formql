package frontend

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/skamensky/formql/pkg/formql/ast"
	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/token"
)

// Parse parses a formula into an AST.
func Parse(input string) (ast.Expr, error) {
	l := newLexer(input)
	p := &parser{lexer: l}

	var err error
	p.current, err = p.lexer.nextToken()
	if err != nil {
		return nil, err
	}

	return p.parse()
}

type parser struct {
	lexer   *lexer
	current token.Token
}

func (p *parser) expectedTokenError(expected token.Type, hint string) error {
	return diagnostic.NewError(
		"parser",
		"unexpected_token",
		fmt.Sprintf("expected %s, got %s", expected.HumanLabel(), p.current.HumanLabel()),
		hint,
		p.current.Position,
	)
}

func (p *parser) missingOperatorError() error {
	hint := "insert an operator between the two expressions"
	if p.current.Type == token.STRING {
		hint = "use '&' for string concatenation"
	}
	return diagnostic.NewError(
		"parser",
		"missing_operator_between_expressions",
		fmt.Sprintf("missing operator between expressions before %s", p.current.HumanLabel()),
		hint,
		p.current.Position,
	)
}

func (p *parser) expectedFunctionCloseHint() string {
	switch {
	case p.current.Type == token.EOF:
		return "close the function call with ')'"
	case token.StartsExpression(p.current.Type):
		return "separate function arguments with commas"
	default:
		return "close the function call with ')'"
	}
}

func (p *parser) next() error {
	next, err := p.lexer.nextToken()
	if err != nil {
		return err
	}
	p.current = next
	return nil
}

func (p *parser) eat(expected token.Type) error {
	if p.current.Type != expected {
		return p.expectedTokenError(expected, "")
	}
	return p.next()
}

func (p *parser) parse() (ast.Expr, error) {
	if p.current.Type == token.EOF {
		return nil, diagnostic.New("parser", "empty formula", p.current.Position)
	}

	node, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	if p.current.Type != token.EOF {
		if token.StartsExpression(p.current.Type) {
			return nil, p.missingOperatorError()
		}
		return nil, diagnostic.NewError("parser", "unexpected_token_after_expression", fmt.Sprintf("unexpected %s after the end of the expression", p.current.HumanLabel()), "remove the extra token or add an operator before it", p.current.Position)
	}

	return node, nil
}

func (p *parser) parseComparison() (ast.Expr, error) {
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current.Type {
		case token.GT, token.GTE, token.LT, token.LTE, token.EQ, token.NEQ:
			op := p.current.Literal
			pos := p.current.Position
			if err := p.next(); err != nil {
				return nil, err
			}
			right, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			node = &ast.BinaryExpr{
				Kind:  "binary_expr",
				Op:    op,
				Left:  node,
				Right: right,
				Pos:   pos,
			}
		default:
			return node, nil
		}
	}
}

func (p *parser) parseExpression() (ast.Expr, error) {
	node, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current.Type {
		case token.PLUS, token.MINUS, token.AMPERSAND:
			op := p.current.Literal
			pos := p.current.Position
			if err := p.next(); err != nil {
				return nil, err
			}
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			node = &ast.BinaryExpr{
				Kind:  "binary_expr",
				Op:    op,
				Left:  node,
				Right: right,
				Pos:   pos,
			}
		default:
			return node, nil
		}
	}
}

func (p *parser) parseTerm() (ast.Expr, error) {
	node, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current.Type {
		case token.MULTIPLY, token.DIVIDE:
			op := p.current.Literal
			pos := p.current.Position
			if err := p.next(); err != nil {
				return nil, err
			}
			right, err := p.parseFactor()
			if err != nil {
				return nil, err
			}
			node = &ast.BinaryExpr{
				Kind:  "binary_expr",
				Op:    op,
				Left:  node,
				Right: right,
				Pos:   pos,
			}
		default:
			return node, nil
		}
	}
}

func (p *parser) parseFactor() (ast.Expr, error) {
	switch p.current.Type {
	case token.PLUS:
		return nil, diagnostic.NewError("parser", "unsupported_unary_plus", "unary plus is not supported", "remove the leading '+'", p.current.Position)
	case token.MINUS:
		pos := p.current.Position
		if err := p.next(); err != nil {
			return nil, err
		}
		operand, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{
			Kind:    "unary_expr",
			Op:      "-",
			Operand: operand,
			Pos:     pos,
		}, nil
	case token.NUMBER:
		pos := p.current.Position
		value, err := strconv.ParseFloat(p.current.Literal, 64)
		if err != nil {
			return nil, diagnostic.NewError("parser", "invalid_numeric_literal", "invalid numeric literal", "check the number format", pos)
		}
		if err := p.next(); err != nil {
			return nil, err
		}
		return &ast.NumberLiteral{Kind: "number_literal", Value: value, Pos: pos}, nil
	case token.STRING:
		pos := p.current.Position
		value := p.current.Literal
		if err := p.next(); err != nil {
			return nil, err
		}
		return &ast.StringLiteral{Kind: "string_literal", Value: value, Pos: pos}, nil
	case token.IDENT:
		return p.parseIdentifierLike()
	case token.LPAREN:
		if err := p.eat(token.LPAREN); err != nil {
			return nil, err
		}
		node, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		if err := p.eat(token.RPAREN); err != nil {
			if token.StartsExpression(p.current.Type) {
				return nil, p.missingOperatorError()
			}
			return nil, err
		}
		return node, nil
	case token.EOF:
		return nil, diagnostic.NewError("parser", "unexpected_end_of_formula", "unexpected end of formula", "finish the expression or close any open function calls and parentheses", p.current.Position)
	default:
		return nil, diagnostic.NewError("parser", "unexpected_token", fmt.Sprintf("unexpected %s", p.current.HumanLabel()), "check for a missing operator or an extra token", p.current.Position)
	}
}

func (p *parser) parseIdentifierLike() (ast.Expr, error) {
	literal := p.current.Literal
	pos := p.current.Position
	lower := strings.ToLower(literal)
	upper := strings.ToUpper(literal)

	if err := p.next(); err != nil {
		return nil, err
	}

	if p.current.Type == token.LPAREN {
		if err := p.eat(token.LPAREN); err != nil {
			return nil, err
		}
		args := make([]ast.Expr, 0, 4)
		if p.current.Type != token.RPAREN {
			for {
				arg, err := p.parseComparison()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.current.Type != token.COMMA {
					break
				}
				if err := p.eat(token.COMMA); err != nil {
					return nil, err
				}
			}
		}
		if token.StartsExpression(p.current.Type) {
			return nil, p.missingOperatorError()
		}
		if err := p.eat(token.RPAREN); err != nil {
			return nil, p.expectedTokenError(token.RPAREN, p.expectedFunctionCloseHint())
		}
		return &ast.CallExpr{
			Kind: "call_expr",
			Name: upper,
			Args: args,
			Pos:  pos,
		}, nil
	}

	switch upper {
	case "TRUE":
		return &ast.BooleanLiteral{Kind: "boolean_literal", Value: true, Pos: pos}, nil
	case "FALSE":
		return &ast.BooleanLiteral{Kind: "boolean_literal", Value: false, Pos: pos}, nil
	case "NULL":
		return &ast.NullLiteral{Kind: "null_literal", Pos: pos}, nil
	}

	if strings.HasSuffix(lower, "_rel") && p.current.Type == token.DOT {
		return p.parseRelationship(lower, pos)
	}

	return &ast.Identifier{
		Kind: "identifier",
		Name: lower,
		Pos:  pos,
	}, nil
}

func (p *parser) parseRelationship(first string, pos int) (ast.Expr, error) {
	chain := []string{strings.TrimSuffix(first, "_rel")}

	for {
		if err := p.eat(token.DOT); err != nil {
			return nil, err
		}
		if p.current.Type != token.IDENT {
			return nil, diagnostic.NewError("parser", "invalid_relationship_path", fmt.Sprintf("relationship path must end in a field name, got %s", p.current.HumanLabel()), "relationship paths look like customer_rel.email", p.current.Position)
		}

		segment := strings.ToLower(p.current.Literal)
		if err := p.eat(token.IDENT); err != nil {
			return nil, err
		}

		if strings.HasSuffix(segment, "_rel") {
			if p.current.Type != token.DOT {
				return nil, diagnostic.NewError("parser", "invalid_relationship_path", "relationship path must end in a field, not another relationship", "add a field name after the relationship path, for example customer_rel.email", pos)
			}
			chain = append(chain, strings.TrimSuffix(segment, "_rel"))
			continue
		}

		return &ast.RelationshipRef{
			Kind:  "relationship_ref",
			Chain: chain,
			Field: segment,
			Pos:   pos,
		}, nil
	}
}
