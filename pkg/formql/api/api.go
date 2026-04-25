package api

import (
	"context"
	"strings"

	"github.com/skamensky/formql/pkg/formql"
	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/livecatalog"
	"github.com/skamensky/formql/pkg/formql/schema"
	"github.com/skamensky/formql/pkg/formql/verify"
)

// LoadCatalogJSON decodes and validates a compiler catalog from JSON.
func LoadCatalogJSON(data []byte) (*schema.Catalog, error) {
	catalogValue, err := catalog.DecodeCatalogJSON(data)
	if err != nil {
		return nil, err
	}
	return catalogValue, nil
}

// VerifySQL runs the default verification pipeline against SQL text.
func VerifySQL(ctx context.Context, sql string, mode verify.Mode) (verify.Result, error) {
	return verify.DefaultVerifier().Verify(ctx, verify.Request{
		SQL:  sql,
		Mode: normalizeMode(mode),
	})
}

// LoadSnapshot resolves a schema snapshot from a shared catalog provider.
func LoadSnapshot(ctx context.Context, provider catalog.Provider, ref catalog.Ref) (*catalog.Snapshot, error) {
	return provider.Load(ctx, ref)
}

// LoadSchemaInfo resolves frontend-facing schema info from a shared provider.
func LoadSchemaInfo(ctx context.Context, provider catalog.Provider, ref catalog.Ref) (*catalog.Info, error) {
	return catalog.Inspector{Provider: provider}.LoadInfo(ctx, ref)
}

// LoadSchemaInfoJSON resolves schema info from JSON-encoded catalog data.
func LoadSchemaInfoJSON(data []byte) (*catalog.Info, error) {
	catalogValue, err := LoadCatalogJSON(data)
	if err != nil {
		return nil, err
	}
	provider := catalog.StaticProvider{
		Snapshot: &catalog.Snapshot{
			Catalog: catalogValue,
		},
	}
	return LoadSchemaInfo(context.Background(), provider, catalog.Ref{})
}

// LoadCatalogIntrospectionJSON converts raw host introspection metadata into a validated compiler catalog.
func LoadCatalogIntrospectionJSON(data []byte) (*schema.Catalog, error) {
	snapshot, err := livecatalog.SnapshotFromIntrospectionJSON(data)
	if err != nil {
		return nil, err
	}
	return snapshot.Catalog, nil
}

// CompileCatalogJSON compiles a formula using a JSON-encoded schema catalog.
func CompileCatalogJSON(data []byte, formulaText, fieldAlias string) (*formql.Compilation, error) {
	catalogValue, err := LoadCatalogJSON(data)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	compilation, err := formql.Compile(formulaText, catalogValue, alias)
	if err != nil {
		return nil, err
	}
	return compilation, nil
}

// CompileCatalogIntrospectionJSON compiles a formula from raw host introspection metadata.
func CompileCatalogIntrospectionJSON(data []byte, formulaText, fieldAlias string) (*formql.Compilation, error) {
	catalogValue, err := LoadCatalogIntrospectionJSON(data)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	return formql.Compile(formulaText, catalogValue, alias)
}

// CompileDocumentCatalogJSON compiles a multi-field document using a JSON-encoded schema catalog.
func CompileDocumentCatalogJSON(data []byte, documentText string) (*formql.DocumentCompilation, error) {
	catalogValue, err := LoadCatalogJSON(data)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocument(documentText, catalogValue)
}

// CompileDocumentCatalogIntrospectionJSON compiles a multi-field document from raw host introspection metadata.
func CompileDocumentCatalogIntrospectionJSON(data []byte, documentText string) (*formql.DocumentCompilation, error) {
	catalogValue, err := LoadCatalogIntrospectionJSON(data)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocument(documentText, catalogValue)
}

// Compile loads a catalog from a provider and compiles a formula against it.
func Compile(ctx context.Context, provider catalog.Provider, ref catalog.Ref, formulaText, fieldAlias string) (*formql.Compilation, error) {
	snapshot, err := LoadSnapshot(ctx, provider, ref)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	return formql.Compile(formulaText, snapshot.Catalog, alias)
}

// CompileDocument loads a catalog from a provider and compiles a multi-field document against it.
func CompileDocument(ctx context.Context, provider catalog.Provider, ref catalog.Ref, documentText string) (*formql.DocumentCompilation, error) {
	snapshot, err := LoadSnapshot(ctx, provider, ref)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocument(documentText, snapshot.Catalog)
}

// CompileAndVerifyCatalogJSON compiles a formula and verifies its generated SQL.
func CompileAndVerifyCatalogJSON(ctx context.Context, data []byte, formulaText, fieldAlias string, mode verify.Mode) (*formql.Compilation, verify.Result, error) {
	compilation, err := CompileCatalogJSON(data, formulaText, fieldAlias)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

// CompileAndVerifyDocumentCatalogJSON compiles a multi-field document and verifies its generated SQL.
func CompileAndVerifyDocumentCatalogJSON(ctx context.Context, data []byte, documentText string, mode verify.Mode) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocumentCatalogJSON(data, documentText)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

// CompileAndVerifyCatalogIntrospectionJSON compiles a formula from raw host introspection metadata
// and verifies the generated SQL through the shared verifier pipeline.
func CompileAndVerifyCatalogIntrospectionJSON(ctx context.Context, data []byte, formulaText, fieldAlias string, mode verify.Mode) (*formql.Compilation, verify.Result, error) {
	compilation, err := CompileCatalogIntrospectionJSON(data, formulaText, fieldAlias)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

// CompileAndVerifyDocumentCatalogIntrospectionJSON compiles a multi-field document from raw host
// introspection metadata and verifies the generated SQL through the shared verifier pipeline.
func CompileAndVerifyDocumentCatalogIntrospectionJSON(ctx context.Context, data []byte, documentText string, mode verify.Mode) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocumentCatalogIntrospectionJSON(data, documentText)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

// CompileAndVerify loads a catalog from a provider, compiles a formula, and
// runs the shared verification pipeline against the generated SQL.
func CompileAndVerify(ctx context.Context, provider catalog.Provider, ref catalog.Ref, formulaText, fieldAlias string, mode verify.Mode) (*formql.Compilation, verify.Result, error) {
	compilation, err := Compile(ctx, provider, ref, formulaText, fieldAlias)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

// CompileAndVerifyDocument loads a catalog from a provider, compiles a multi-field
// document, and runs the shared verification pipeline against the generated SQL.
func CompileAndVerifyDocument(ctx context.Context, provider catalog.Provider, ref catalog.Ref, documentText string, mode verify.Mode) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocument(ctx, provider, ref, documentText)
	if err != nil {
		return nil, verify.Result{}, err
	}

	result, err := VerifySQL(ctx, compilation.SQL.Query, mode)
	if err != nil {
		return nil, verify.Result{}, err
	}

	return compilation, result, nil
}

func normalizeMode(mode verify.Mode) verify.Mode {
	switch strings.ToLower(strings.TrimSpace(string(mode))) {
	case "", string(verify.ModeSyntax):
		return verify.ModeSyntax
	case string(verify.ModePlan):
		return verify.ModePlan
	default:
		return mode
	}
}
