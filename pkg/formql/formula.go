package formql

import (
	"github.com/skamensky/formql/pkg/formql/ast"
	"github.com/skamensky/formql/pkg/formql/backend/postgres"
	"github.com/skamensky/formql/pkg/formql/frontend"
	"github.com/skamensky/formql/pkg/formql/ir"
	"github.com/skamensky/formql/pkg/formql/schema"
	"github.com/skamensky/formql/pkg/formql/semantics"
)

// Compilation is the end-to-end compiler output for a formula.
type Compilation struct {
	AST ast.Expr          `json:"ast"`
	HIR *ir.Plan          `json:"hir"`
	SQL postgres.Artifact `json:"sql"`
}

// Parse parses a formula into an AST.
func Parse(input string) (ast.Expr, error) {
	return frontend.Parse(input)
}

// Lower parses and typechecks a formula into semantic IR.
func Lower(input string, catalog schema.Resolver) (*ir.Plan, error) {
	parsed, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return semantics.Lower(parsed, catalog)
}

// RenderSQL renders semantic IR into PostgreSQL SQL.
func RenderSQL(plan *ir.Plan, fieldAlias string) (postgres.Artifact, error) {
	return postgres.Renderer{}.Render(plan, fieldAlias)
}

// Compile performs parse, typecheck, IR generation, and PostgreSQL rendering.
func Compile(input string, catalog schema.Resolver, fieldAlias string) (*Compilation, error) {
	parsed, err := Parse(input)
	if err != nil {
		return nil, err
	}

	plan, err := semantics.Lower(parsed, catalog)
	if err != nil {
		return nil, err
	}

	sqlArtifact, err := RenderSQL(plan, fieldAlias)
	if err != nil {
		return nil, err
	}

	return &Compilation{
		AST: parsed,
		HIR: plan,
		SQL: sqlArtifact,
	}, nil
}
