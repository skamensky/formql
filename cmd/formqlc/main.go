package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/skamensky/formql/pkg/formql"
	"github.com/skamensky/formql/pkg/formql/api"
	"github.com/skamensky/formql/pkg/formql/catalog"
	"github.com/skamensky/formql/pkg/formql/filemeta"
	"github.com/skamensky/formql/pkg/formql/livecatalog"
	"github.com/skamensky/formql/pkg/formql/lsp"
	"github.com/skamensky/formql/pkg/formql/schema"
	"github.com/skamensky/formql/pkg/formql/verify"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "ast":
		exitIfErr(runAST(os.Args[2:]))
	case "document-ast":
		exitIfErr(runDocumentAST(os.Args[2:]))
	case "hir":
		exitIfErr(runHIR(os.Args[2:]))
	case "document-hir":
		exitIfErr(runDocumentHIR(os.Args[2:]))
	case "query":
		exitIfErr(runQuery(os.Args[2:]))
	case "document-query":
		exitIfErr(runDocumentQuery(os.Args[2:]))
	case "verify-sql":
		exitIfErr(runVerifySQL(os.Args[2:]))
	case "verify-query":
		exitIfErr(runVerifyQuery(os.Args[2:]))
	case "verify-document-query":
		exitIfErr(runVerifyDocumentQuery(os.Args[2:]))
	case "typecheck":
		exitIfErr(runTypecheck(os.Args[2:]))
	case "document-typecheck":
		exitIfErr(runDocumentTypecheck(os.Args[2:]))
	case "catalog":
		exitIfErr(runCatalog(os.Args[2:]))
	case "lsp":
		exitIfErr(runLSP(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: formqlc <ast|document-ast|hir|document-hir|query|document-query|verify-sql|verify-query|verify-document-query|typecheck|document-typecheck|catalog|lsp> [flags]")
}

func runAST(args []string) error {
	fs := flag.NewFlagSet("ast", flag.ContinueOnError)
	formulaText := fs.String("formula", "", "formula source")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *formulaText == "" {
		return fmt.Errorf("ast requires -formula")
	}
	parsed, err := formql.Parse(*formulaText)
	if err != nil {
		return err
	}
	return writeJSON(parsed)
}

func runDocumentAST(args []string) error {
	fs := flag.NewFlagSet("document-ast", flag.ContinueOnError)
	formulaText := fs.String("formula", "", "document source")
	documentText := fs.String("document", "", "document source")
	if err := fs.Parse(args); err != nil {
		return err
	}
	source := *documentText
	if source == "" {
		source = *formulaText
	}
	if source == "" {
		return fmt.Errorf("document-ast requires -document or -formula")
	}
	parsed, err := formql.ParseDocument(source)
	if err != nil {
		return err
	}
	return writeJSON(parsed)
}

func runHIR(args []string) error {
	fs := commonFlagSet("hir")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	plan, err := formql.Lower(config.formulaText, catalog)
	if err != nil {
		return err
	}
	return writeJSON(plan)
}

func runDocumentHIR(args []string) error {
	fs := commonFlagSet("document-hir")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	plan, err := formql.LowerDocument(config.formulaText, catalog)
	if err != nil {
		return err
	}
	return writeJSON(plan)
}

func runQuery(args []string) error {
	fs := commonFlagSet("query")
	field := fs.String("field", "result", "output field alias")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	compilation, err := formql.Compile(config.formulaText, catalog, *field)
	if err != nil {
		return err
	}
	return writeJSON(struct {
		Expression string   `json:"expression"`
		Query      string   `json:"query"`
		Joins      []string `json:"joins"`
		Warnings   any      `json:"warnings,omitempty"`
	}{
		Expression: compilation.SQL.Expression,
		Query:      compilation.SQL.Query,
		Joins:      compilation.SQL.JoinClauses,
		Warnings:   compilation.HIR.Warnings,
	})
}

func runDocumentQuery(args []string) error {
	fs := commonFlagSet("document-query")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	compilation, err := formql.CompileDocument(config.formulaText, catalog)
	if err != nil {
		return err
	}
	return writeJSON(struct {
		Projections any      `json:"projections"`
		Query       string   `json:"query"`
		Joins       []string `json:"joins"`
		Warnings    any      `json:"warnings,omitempty"`
	}{
		Projections: compilation.SQL.Projections,
		Query:       compilation.SQL.Query,
		Joins:       compilation.SQL.JoinClauses,
		Warnings:    compilation.HIR.Warnings,
	})
}

func runTypecheck(args []string) error {
	fs := commonFlagSet("typecheck")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	plan, err := formql.Lower(config.formulaText, catalog)
	if err != nil {
		return err
	}
	return writeJSON(struct {
		Valid        bool   `json:"valid"`
		ResultType   string `json:"result_type"`
		JoinCount    int    `json:"join_count"`
		WarningCount int    `json:"warning_count"`
		Warnings     any    `json:"warnings,omitempty"`
	}{
		Valid:        true,
		ResultType:   string(plan.Expr.Type()),
		JoinCount:    len(plan.Joins),
		WarningCount: len(plan.Warnings),
		Warnings:     plan.Warnings,
	})
}

func runDocumentTypecheck(args []string) error {
	fs := commonFlagSet("document-typecheck")
	if err := fs.Parse(args); err != nil {
		return err
	}
	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}
	plan, err := formql.LowerDocument(config.formulaText, catalog)
	if err != nil {
		return err
	}
	fields := make([]struct {
		Alias      string `json:"alias"`
		ResultType string `json:"result_type"`
	}, 0, len(plan.Fields))
	for _, field := range plan.Fields {
		fields = append(fields, struct {
			Alias      string `json:"alias"`
			ResultType string `json:"result_type"`
		}{
			Alias:      field.Alias,
			ResultType: string(field.ResultType),
		})
	}
	return writeJSON(struct {
		Valid        bool `json:"valid"`
		Fields       any  `json:"fields"`
		JoinCount    int  `json:"join_count"`
		WarningCount int  `json:"warning_count"`
		Warnings     any  `json:"warnings,omitempty"`
	}{
		Valid:        true,
		Fields:       fields,
		JoinCount:    len(plan.Joins),
		WarningCount: len(plan.Warnings),
		Warnings:     plan.Warnings,
	})
}

func runVerifySQL(args []string) error {
	fs := flag.NewFlagSet("verify-sql", flag.ContinueOnError)
	sqlText := fs.String("sql", "", "sql text")
	mode := fs.String("mode", string(verify.ModeSyntax), "verification mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sqlText == "" {
		return fmt.Errorf("verify-sql requires -sql")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := api.VerifySQL(ctx, *sqlText, verify.Mode(*mode))
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func runVerifyQuery(args []string) error {
	fs := commonFlagSet("verify-query")
	field := fs.String("field", "result", "output field alias")
	mode := fs.String("mode", string(verify.ModeSyntax), "verification mode")
	if err := fs.Parse(args); err != nil {
		return err
	}

	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}

	compilation, err := formql.Compile(config.formulaText, catalog, *field)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := api.VerifySQL(ctx, compilation.SQL.Query, verify.Mode(*mode))
	if err != nil {
		return err
	}

	return writeJSON(struct {
		Query        string        `json:"query"`
		Verification verify.Result `json:"verification"`
	}{
		Query:        compilation.SQL.Query,
		Verification: result,
	})
}

func runVerifyDocumentQuery(args []string) error {
	fs := commonFlagSet("verify-document-query")
	mode := fs.String("mode", string(verify.ModeSyntax), "verification mode")
	if err := fs.Parse(args); err != nil {
		return err
	}

	config := extractCommonConfig(fs)
	catalog, err := loadCatalog(&config)
	if err != nil {
		return err
	}

	compilation, err := formql.CompileDocument(config.formulaText, catalog)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := api.VerifySQL(ctx, compilation.SQL.Query, verify.Mode(*mode))
	if err != nil {
		return err
	}

	return writeJSON(struct {
		Query        string        `json:"query"`
		Verification verify.Result `json:"verification"`
	}{
		Query:        compilation.SQL.Query,
		Verification: result,
	})
}

func runCatalog(args []string) error {
	fs := flag.NewFlagSet("catalog", flag.ContinueOnError)
	databaseURL := fs.String("database-url", envOr("FORMULA_DATABASE_URL", envOr("DATABASE_URL", "")), "postgres connection string")
	table := fs.String("table", envOr("FORMULA_BASE_TABLE", ""), "base table")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *databaseURL == "" {
		return fmt.Errorf("catalog requires -database-url or FORMULA_DATABASE_URL")
	}
	if *table == "" {
		return fmt.Errorf("catalog requires -table or FORMULA_BASE_TABLE")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	catalog, err := livecatalog.LoadCatalog(ctx, *databaseURL, *table)
	if err != nil {
		return err
	}
	return writeJSON(catalog)
}

func runLSP(args []string) error {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	databaseURL := fs.String("database-url", envOr("FORMULA_DATABASE_URL", envOr("DATABASE_URL", "")), "postgres connection string")
	schemaPath := fs.String("schema", "", "path to schema json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	provider, err := loadLSPProvider(ctx, *databaseURL, *schemaPath)
	if err != nil {
		return err
	}
	defer provider.Close()

	server := lsp.NewServer(os.Stdin, os.Stdout, provider, lsp.Config{
		SchemaPath: *schemaPath,
	})
	return server.Run(ctx)
}

type commonConfig struct {
	formulaText string
	formulaPath string
	schemaPath  string
	databaseURL string
	table       string
}

func commonFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.String("formula", "", "formula source")
	fs.String("formula-file", "", "path to formula/document source")
	fs.String("document", "", "document source")
	fs.String("document-file", "", "path to document source")
	fs.String("schema", "", "path to schema json")
	fs.String("database-url", envOr("FORMULA_DATABASE_URL", envOr("DATABASE_URL", "")), "postgres connection string")
	fs.String("table", envOr("FORMULA_BASE_TABLE", ""), "base table")
	return fs
}

func extractCommonConfig(fs *flag.FlagSet) commonConfig {
	source := fs.Lookup("formula").Value.String()
	if source == "" {
		source = fs.Lookup("document").Value.String()
	}
	sourcePath := fs.Lookup("formula-file").Value.String()
	if sourcePath == "" {
		sourcePath = fs.Lookup("document-file").Value.String()
	}
	return commonConfig{
		formulaText: source,
		formulaPath: sourcePath,
		schemaPath:  fs.Lookup("schema").Value.String(),
		databaseURL: fs.Lookup("database-url").Value.String(),
		table:       fs.Lookup("table").Value.String(),
	}
}

func loadCatalog(config *commonConfig) (*schema.Catalog, error) {
	sourceText, table, err := resolveCommandSource(*config)
	if err != nil {
		return nil, err
	}
	config.formulaText = sourceText

	if config.schemaPath != "" {
		file, err := os.ReadFile(config.schemaPath)
		if err != nil {
			return nil, err
		}
		catalog, err := api.LoadCatalogJSON(file)
		if err != nil {
			return nil, err
		}
		catalog.BaseTable = table
		return catalog, nil
	}
	if config.databaseURL == "" {
		return nil, fmt.Errorf("provide either -schema or -database-url")
	}
	if table == "" {
		return nil, fmt.Errorf("provide -table for inline live catalog commands or file metadata for file-based live catalog commands")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return livecatalog.LoadCatalog(ctx, config.databaseURL, table)
}

func resolveCommandSource(config commonConfig) (string, string, error) {
	if config.formulaPath == "" {
		if config.formulaText == "" {
			return "", "", fmt.Errorf("command requires -formula, -document, -formula-file, or -document-file")
		}
		table := strings.ToLower(strings.TrimSpace(config.table))
		if table == "" {
			return "", "", fmt.Errorf("inline commands require -table; file commands require source or sidecar metadata")
		}
		return config.formulaText, table, nil
	}

	source, err := os.ReadFile(config.formulaPath)
	if err != nil {
		return "", "", err
	}
	text := string(source)

	table, err := resolveFileBaseTable(config.formulaPath, text)
	if err != nil {
		return "", "", err
	}
	return text, table, nil
}

func resolveFileBaseTable(path, text string) (string, error) {
	if metadata, ok, err := filemeta.ParseSource(text); err != nil {
		return "", err
	} else if ok && metadata.BaseTable() != "" {
		return metadata.BaseTable(), nil
	}

	if metadata, _, ok, err := filemeta.LoadSidecar(path); err != nil {
		return "", err
	} else if ok && metadata.BaseTable() != "" {
		return metadata.BaseTable(), nil
	}

	return "", fmt.Errorf("%s is missing FormQL table metadata; add a leading comment like // formql: table=rental_contract or adjacent %s", path, filemeta.SidecarPaths(path)[0])
}

func writeJSON(value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func exitIfErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func envOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value
	}
	return fallback
}

func loadLSPProvider(ctx context.Context, databaseURL, schemaPath string) (catalog.ManagedProvider, error) {
	if schemaPath != "" {
		file, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, err
		}

		var schemaCatalog schema.Catalog
		if err := json.Unmarshal(file, &schemaCatalog); err != nil {
			return nil, err
		}
		if err := schemaCatalog.Validate(); err != nil {
			return nil, err
		}
		return catalog.NoopCloseProvider{
			Provider: catalog.StaticProvider{
				Snapshot: &catalog.Snapshot{Catalog: &schemaCatalog},
			},
		}, nil
	}

	if databaseURL == "" {
		return nil, fmt.Errorf("lsp requires either -schema or -database-url")
	}

	return livecatalog.NewPostgresProvider(ctx, databaseURL)
}
