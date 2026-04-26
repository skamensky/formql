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

// DocumentCompilation is the end-to-end compiler output for a multi-field document.
type DocumentCompilation struct {
	AST *ast.Document             `json:"ast"`
	HIR *ir.DocumentPlan          `json:"hir"`
	SQL postgres.DocumentArtifact `json:"sql"`
}

// Parse parses a formula into an AST.
func Parse(input string) (ast.Expr, error) {
	return frontend.Parse(input)
}

// ParseDocument parses a multi-field document into an AST.
func ParseDocument(input string) (*ast.Document, error) {
	return frontend.ParseDocument(input)
}

// Lower parses and typechecks a formula into semantic IR.
func Lower(input string, catalog schema.Resolver) (*ir.Plan, error) {
	return LowerWithOptions(input, catalog, Options{})
}

// LowerWithOptions parses and typechecks a formula into semantic IR with explicit options.
func LowerWithOptions(input string, catalog schema.Resolver, options Options) (*ir.Plan, error) {
	parsed, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return semantics.LowerWithOptions(parsed, catalog, semantics.Options{
		MaxRelationshipDepth: options.Normalized().MaxRelationshipDepth,
	})
}

// LowerDocument parses and typechecks a multi-field document into semantic IR.
func LowerDocument(input string, catalog schema.Resolver) (*ir.DocumentPlan, error) {
	return LowerDocumentWithOptions(input, catalog, Options{})
}

// LowerDocumentWithOptions parses and typechecks a multi-field document into semantic IR with explicit options.
func LowerDocumentWithOptions(input string, catalog schema.Resolver, options Options) (*ir.DocumentPlan, error) {
	parsed, err := ParseDocument(input)
	if err != nil {
		return nil, err
	}
	return semantics.LowerDocumentWithOptions(parsed, catalog, semantics.Options{
		MaxRelationshipDepth: options.Normalized().MaxRelationshipDepth,
	})
}

// RenderSQL renders semantic IR into PostgreSQL SQL.
func RenderSQL(plan *ir.Plan, fieldAlias string) (postgres.Artifact, error) {
	return postgres.Renderer{}.Render(plan, fieldAlias)
}

// RenderDocumentSQL renders document semantic IR into PostgreSQL SQL.
func RenderDocumentSQL(plan *ir.DocumentPlan) (postgres.DocumentArtifact, error) {
	return postgres.Renderer{}.RenderDocument(plan)
}

// Compile performs parse, typecheck, IR generation, and PostgreSQL rendering.
func Compile(input string, catalog schema.Resolver, fieldAlias string) (*Compilation, error) {
	return CompileWithOptions(input, catalog, fieldAlias, Options{})
}

// CompileWithOptions performs parse, typecheck, IR generation, and PostgreSQL rendering with explicit options.
func CompileWithOptions(input string, catalog schema.Resolver, fieldAlias string, options Options) (*Compilation, error) {
	parsed, err := Parse(input)
	if err != nil {
		return nil, err
	}

	plan, err := semantics.LowerWithOptions(parsed, catalog, semantics.Options{
		MaxRelationshipDepth: options.Normalized().MaxRelationshipDepth,
	})
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

// CompileDocument performs parse, typecheck, IR generation, and PostgreSQL rendering for a multi-field document.
func CompileDocument(input string, catalog schema.Resolver) (*DocumentCompilation, error) {
	return CompileDocumentWithOptions(input, catalog, Options{})
}

// CompileDocumentWithOptions performs parse, typecheck, IR generation, and PostgreSQL rendering for a multi-field document.
func CompileDocumentWithOptions(input string, catalog schema.Resolver, options Options) (*DocumentCompilation, error) {
	parsed, err := ParseDocument(input)
	if err != nil {
		return nil, err
	}

	plan, err := semantics.LowerDocumentWithOptions(parsed, catalog, semantics.Options{
		MaxRelationshipDepth: options.Normalized().MaxRelationshipDepth,
	})
	if err != nil {
		return nil, err
	}

	sqlArtifact, err := RenderDocumentSQL(plan)
	if err != nil {
		return nil, err
	}

	return &DocumentCompilation{
		AST: parsed,
		HIR: plan,
		SQL: sqlArtifact,
	}, nil
}
