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

// CompileOptions configures compiler behavior for host APIs.
type CompileOptions struct {
	MaxRelationshipDepth int `json:"max_relationship_depth,omitempty"`
}

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
	return CompileCatalogJSONWithOptions(data, formulaText, fieldAlias, CompileOptions{})
}

// CompileCatalogJSONWithOptions compiles a formula using a JSON-encoded schema catalog and explicit options.
func CompileCatalogJSONWithOptions(data []byte, formulaText, fieldAlias string, options CompileOptions) (*formql.Compilation, error) {
	catalogValue, err := LoadCatalogJSON(data)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	compilation, err := formql.CompileWithOptions(formulaText, catalogValue, alias, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
	if err != nil {
		return nil, err
	}
	return compilation, nil
}

// CompileCatalogIntrospectionJSON compiles a formula from raw host introspection metadata.
func CompileCatalogIntrospectionJSON(data []byte, formulaText, fieldAlias string) (*formql.Compilation, error) {
	return CompileCatalogIntrospectionJSONWithOptions(data, formulaText, fieldAlias, CompileOptions{})
}

// CompileCatalogIntrospectionJSONWithOptions compiles a formula from raw host introspection metadata.
func CompileCatalogIntrospectionJSONWithOptions(data []byte, formulaText, fieldAlias string, options CompileOptions) (*formql.Compilation, error) {
	catalogValue, err := LoadCatalogIntrospectionJSON(data)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	return formql.CompileWithOptions(formulaText, catalogValue, alias, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
}

// CompileDocumentCatalogJSON compiles a multi-field document using a JSON-encoded schema catalog.
func CompileDocumentCatalogJSON(data []byte, documentText string) (*formql.DocumentCompilation, error) {
	return CompileDocumentCatalogJSONWithOptions(data, documentText, CompileOptions{})
}

// CompileDocumentCatalogJSONWithOptions compiles a multi-field document using a JSON-encoded schema catalog.
func CompileDocumentCatalogJSONWithOptions(data []byte, documentText string, options CompileOptions) (*formql.DocumentCompilation, error) {
	catalogValue, err := LoadCatalogJSON(data)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocumentWithOptions(documentText, catalogValue, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
}

// CompileDocumentCatalogIntrospectionJSON compiles a multi-field document from raw host introspection metadata.
func CompileDocumentCatalogIntrospectionJSON(data []byte, documentText string) (*formql.DocumentCompilation, error) {
	return CompileDocumentCatalogIntrospectionJSONWithOptions(data, documentText, CompileOptions{})
}

// CompileDocumentCatalogIntrospectionJSONWithOptions compiles a multi-field document from raw host introspection metadata.
func CompileDocumentCatalogIntrospectionJSONWithOptions(data []byte, documentText string, options CompileOptions) (*formql.DocumentCompilation, error) {
	catalogValue, err := LoadCatalogIntrospectionJSON(data)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocumentWithOptions(documentText, catalogValue, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
}

// Compile loads a catalog from a provider and compiles a formula against it.
func Compile(ctx context.Context, provider catalog.Provider, ref catalog.Ref, formulaText, fieldAlias string) (*formql.Compilation, error) {
	return CompileWithOptions(ctx, provider, ref, formulaText, fieldAlias, CompileOptions{})
}

// CompileWithOptions loads a catalog from a provider and compiles a formula against it.
func CompileWithOptions(ctx context.Context, provider catalog.Provider, ref catalog.Ref, formulaText, fieldAlias string, options CompileOptions) (*formql.Compilation, error) {
	snapshot, err := LoadSnapshot(ctx, provider, ref)
	if err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(fieldAlias)
	if alias == "" {
		alias = "result"
	}

	return formql.CompileWithOptions(formulaText, snapshot.Catalog, alias, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
}

// CompileDocument loads a catalog from a provider and compiles a multi-field document against it.
func CompileDocument(ctx context.Context, provider catalog.Provider, ref catalog.Ref, documentText string) (*formql.DocumentCompilation, error) {
	return CompileDocumentWithOptions(ctx, provider, ref, documentText, CompileOptions{})
}

// CompileDocumentWithOptions loads a catalog from a provider and compiles a multi-field document against it.
func CompileDocumentWithOptions(ctx context.Context, provider catalog.Provider, ref catalog.Ref, documentText string, options CompileOptions) (*formql.DocumentCompilation, error) {
	snapshot, err := LoadSnapshot(ctx, provider, ref)
	if err != nil {
		return nil, err
	}
	return formql.CompileDocumentWithOptions(documentText, snapshot.Catalog, formql.Options{
		MaxRelationshipDepth: options.MaxRelationshipDepth,
	})
}

// CompileAndVerifyCatalogJSON compiles a formula and verifies its generated SQL.
func CompileAndVerifyCatalogJSON(ctx context.Context, data []byte, formulaText, fieldAlias string, mode verify.Mode) (*formql.Compilation, verify.Result, error) {
	return CompileAndVerifyCatalogJSONWithOptions(ctx, data, formulaText, fieldAlias, mode, CompileOptions{})
}

// CompileAndVerifyCatalogJSONWithOptions compiles a formula and verifies its generated SQL.
func CompileAndVerifyCatalogJSONWithOptions(ctx context.Context, data []byte, formulaText, fieldAlias string, mode verify.Mode, options CompileOptions) (*formql.Compilation, verify.Result, error) {
	compilation, err := CompileCatalogJSONWithOptions(data, formulaText, fieldAlias, options)
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
	return CompileAndVerifyDocumentCatalogJSONWithOptions(ctx, data, documentText, mode, CompileOptions{})
}

// CompileAndVerifyDocumentCatalogJSONWithOptions compiles a multi-field document and verifies its generated SQL.
func CompileAndVerifyDocumentCatalogJSONWithOptions(ctx context.Context, data []byte, documentText string, mode verify.Mode, options CompileOptions) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocumentCatalogJSONWithOptions(data, documentText, options)
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
	return CompileAndVerifyCatalogIntrospectionJSONWithOptions(ctx, data, formulaText, fieldAlias, mode, CompileOptions{})
}

// CompileAndVerifyCatalogIntrospectionJSONWithOptions compiles a formula from raw host introspection metadata.
func CompileAndVerifyCatalogIntrospectionJSONWithOptions(ctx context.Context, data []byte, formulaText, fieldAlias string, mode verify.Mode, options CompileOptions) (*formql.Compilation, verify.Result, error) {
	compilation, err := CompileCatalogIntrospectionJSONWithOptions(data, formulaText, fieldAlias, options)
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
	return CompileAndVerifyDocumentCatalogIntrospectionJSONWithOptions(ctx, data, documentText, mode, CompileOptions{})
}

// CompileAndVerifyDocumentCatalogIntrospectionJSONWithOptions compiles a multi-field document from raw host introspection metadata.
func CompileAndVerifyDocumentCatalogIntrospectionJSONWithOptions(ctx context.Context, data []byte, documentText string, mode verify.Mode, options CompileOptions) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocumentCatalogIntrospectionJSONWithOptions(data, documentText, options)
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
	return CompileAndVerifyWithOptions(ctx, provider, ref, formulaText, fieldAlias, mode, CompileOptions{})
}

// CompileAndVerifyWithOptions loads a catalog from a provider, compiles a formula, and verifies the SQL.
func CompileAndVerifyWithOptions(ctx context.Context, provider catalog.Provider, ref catalog.Ref, formulaText, fieldAlias string, mode verify.Mode, options CompileOptions) (*formql.Compilation, verify.Result, error) {
	compilation, err := CompileWithOptions(ctx, provider, ref, formulaText, fieldAlias, options)
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
	return CompileAndVerifyDocumentWithOptions(ctx, provider, ref, documentText, mode, CompileOptions{})
}

// CompileAndVerifyDocumentWithOptions loads a catalog from a provider, compiles a document, and verifies the SQL.
func CompileAndVerifyDocumentWithOptions(ctx context.Context, provider catalog.Provider, ref catalog.Ref, documentText string, mode verify.Mode, options CompileOptions) (*formql.DocumentCompilation, verify.Result, error) {
	compilation, err := CompileDocumentWithOptions(ctx, provider, ref, documentText, options)
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
