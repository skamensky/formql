# FormQL

FormQL is an Excel-like formula language with a real compiler pipeline and a relational typechecker. The primary backend today is PostgreSQL SQL, with PL/pgSQL available later when expression lowering is not enough.

An earlier prototype exists as inspiration only. FormQL behavior in this repo is defined by the language design, typed IR, and backend contracts here, not by legacy implementation details.

## Current structure

- CLI: [`cmd/formqlc`](./cmd/formqlc/main.go)
- Core API: [`pkg/formql`](./pkg/formql/formula.go)
- Host API: [`pkg/formql/api`](./pkg/formql/api/api.go)
- Catalog providers + cache: [`pkg/formql/catalog`](./pkg/formql/catalog/provider.go)
- LSP: [`pkg/formql/lsp`](./pkg/formql/lsp/server.go)
- Browser/Node WASM entrypoint: [`pkg/formql/wasm`](./pkg/formql/wasm/main.go)
- VS Code extension: [`editors/vscode`](./editors/vscode/package.json)
- Architecture notes: [`docs/architecture.md`](./docs/architecture.md)
- Sample offline catalog: [`examples/catalogs/rental-agency.formql.schema.json`](./examples/catalogs/rental-agency.formql.schema.json)
- Offline VS Code workspaces:
  [`examples/workspaces/offline-rental-offer`](./examples/workspaces/offline-rental-offer/README.md),
  [`examples/workspaces/offline-rental-contract`](./examples/workspaces/offline-rental-contract/README.md) and
  [`examples/workspaces/offline-resale-sale`](./examples/workspaces/offline-resale-sale/README.md)
- Seeded Postgres dev DB: [`docker-compose.yml`](./docker-compose.yml)

## Pipeline

```text
source
  -> AST
  -> typed semantic IR
  -> backend lowering
  -> PostgreSQL SQL
```

Bytecode is not ruled out, but it would be another backend or lower IR stage. It is not a substitute for the semantic IR that the SQL backend and language tooling need.

## Developer tools

- `formqlc ast`: parse only
- `formqlc document-ast`: parse a comma-separated multi-field document
- `formqlc hir`: parse + typecheck + semantic IR
- `formqlc document-hir`: parse + typecheck a multi-field document
- `formqlc typecheck`: typecheck against a live catalog or schema file
- `formqlc document-typecheck`: typecheck a multi-field document
- `formqlc query`: emit PostgreSQL SQL
- `formqlc document-query`: emit PostgreSQL SQL for a multi-projection document
- `formqlc verify-sql`: verify raw SQL through the shared verifier
- `formqlc verify-query`: compile a formula, then verify the generated SQL through the same verifier
- `formqlc verify-document-query`: compile a document, then verify the generated SQL
- `formqlc catalog`: introspect a live Postgres schema
- `formqlc lsp`: run the language server

The typechecker is intentionally driven by a real catalog contract so the compiler, CLI, and LSP all exercise the same schema-resolution code paths.

File-based commands can resolve their table from source metadata instead of a workspace default:

```formql
// formql: table=rental_contract
actual_total, customer.email
```

The supported file-level forms are a leading comment directive (`formql:`, `@formql`, or `formql-meta:` with `key=value` params) or an adjacent `.meta.json` sidecar such as `contract_overview.meta.json`. The `table`/`base_table` value is required for `-formula-file` and `-document-file`; inline `-formula` and `-document` commands still require `-table`.

## Shared catalog shape

Catalog loading is now centered on one provider contract:

- `catalog.Provider`: load a schema snapshot for a logical `Ref`
- `catalog.ManagedProvider`: same, with optional lifecycle cleanup for CLI/LSP hosts
- `catalog.InfoProvider`: frontend/editor-facing schema info view
- `catalog.Cache` + `catalog.CachingProvider`: optional cache layer, outside compiler logic
- `livecatalog.Source`: raw live-database introspection seam for host-specific implementations

That split keeps compiler ownership in Go while still allowing different hosts to gather schema data differently:

- CLI/LSP can use `database/sql`
- the PostgreSQL extension can introspect in-process without a connection string
- browser callers can use checked-in schema JSON through the same provider and info surfaces

## VS Code

The repo now includes a local VS Code extension in [`editors/vscode`](./editors/vscode/package.json).

- canonical file extension: `.formql`
- diagnostics: parser and typechecker errors
- completions: columns, relationships, and built-ins
- hover: columns and relationships
- jump to definition: offline schema mode, into the schema JSON file

The fastest live demos are the bundled rental-agency workspaces at [`examples/workspaces/offline-rental-offer`](./examples/workspaces/offline-rental-offer/README.md), [`examples/workspaces/offline-rental-contract`](./examples/workspaces/offline-rental-contract/README.md), and [`examples/workspaces/offline-resale-sale`](./examples/workspaces/offline-resale-sale/README.md). They use `go run` plus a checked-in schema file, so users do not need a running database to see the editor features work. The `formulas/` directories are now a broader language corpus, not just a couple of demo snippets: they are compiled in tests and deliberately cover the current builtin/operator surface.

The LSP does not use a folder naming convention or workspace-level base table. Each `.formql` file must declare its table in source metadata or an adjacent `.meta.json`; if the table cannot be resolved, the server publishes a file-level diagnostic.

## WASM

The repo also ships a browser/Node wasm bundle built from the same Go compiler and catalog packages:

```bash
make wasm-build
make wasm-smoke
```

The JS surface is:

- `FormQL.loadSchemaInfoJSON(catalogJSON, options?)`
- `FormQL.completeCatalogJSON(catalogJSON, source, offset, options?)`
- `FormQL.compileCatalogJSON(catalogJSON, formula, options?)`
- `FormQL.compileDocumentCatalogJSON(catalogJSON, document, options?)`
- `FormQL.compileAndVerifyCatalogJSON(catalogJSON, formula, options?)`
- `FormQL.compileAndVerifyDocumentCatalogJSON(catalogJSON, document, options?)`
- `FormQL.verifySQL(sql, options?)`

The example frontend at `http://127.0.0.1:8090` now uses those wasm APIs for catalog-aware completion and compilation, then calls the backend for SQL verification and query execution against the seeded rental-agency sample database.

Today, `js/wasm` builds support schema info, parsing, typechecking, and SQL generation. SQL verification itself is reported as unavailable in wasm builds, because the current offline verifier backend is not portable to browser runtimes yet.

For browser flows that need real SQL verification, run the local backend:

```bash
make web-backend
```

Then open `http://127.0.0.1:8090`. The playground compiles in wasm and sends the generated SQL to the backend verifier. The full automated round trip is:

```bash
make web-smoke
```

## Quick start

```bash
make db-up
make catalog BASE_TABLE=rental_contract
make typecheck BASE_TABLE=rental_contract FORMULA='rep.manager.first_name & " @ " & rep.branch.name'
make query BASE_TABLE=resale_sale FORMULA='vehicle.model_name & " / " & STRING(vehicle.model_year)'
make document-query BASE_TABLE=rental_contract FORMULA='actual_total, customer.email, vehicle.model_name AS vehicle_model'
go run ./cmd/formqlc document-query -schema examples/catalogs/rental-agency.formql.schema.json -document-file examples/workspaces/offline-rental-contract/documents/contract_overview.formql
make verify-query BASE_TABLE=resale_sale FORMULA='vehicle.model_name & " / " & STRING(vehicle.model_year)'
```

## PostgreSQL extension

The PostgreSQL extension is now a thin wrapper over the same Go compiler and verifier used by the CLI.

- shared Go logic: [`pkg/formql`](./pkg/formql/formula.go), [`pkg/formql/verify`](./pkg/formql/verify/verify.go), [`pkg/formql/api`](./pkg/formql/api/api.go)
- Go C-ABI bridge: [`pkg/formql/capi`](./pkg/formql/capi/main.go)
- PostgreSQL wrapper: [`ext/formql`](./ext/formql/formql.control)

The extension currently exposes SQL verification helpers, a live catalog export, a JSON-catalog compile entrypoint, and a live in-server compile entrypoint:

- `formql_verify_sql_error(sql text) -> text`
- `formql_verify_sql_ok(sql text) -> boolean`
- `formql_verify_sql_diagnostics(sql text) -> jsonb`
- `formql_catalog(base_table text) -> jsonb`
- `formql_compile_catalog(catalog jsonb, formula text, field_alias text default 'result', verify_mode text default 'syntax') -> jsonb`
- `formql_compile_live(base_table text, formula text, field_alias text default 'result', verify_mode text default 'syntax') -> jsonb`
- `formql_compile_document_catalog(catalog jsonb, document text, verify_mode text default 'syntax') -> jsonb`
- `formql_compile_document_live(base_table text, document text, verify_mode text default 'syntax') -> jsonb`

Because the extension uses a Go runtime helper library, PostgreSQL needs enough optional static TLS available to load it lazily:

```bash
GLIBC_TUNABLES=glibc.rtld.optional_static_tls=2048 postgres ...
```

## Open Language Questions

These are current implementation choices that should be treated as provisional, not language law:

- the current relationship traversal spelling is useful, but may not be the long-term surface syntax
- relationship names are currently inferred mechanically from foreign-key metadata; explicit schema naming may be better later
- live catalog introspection currently collapses some Postgres types aggressively, for example arrays to string
- `date` and `timestamp` are currently merged into one semantic type

If we keep any of those, it should be because they survive language design review, not because they were convenient in an earlier prototype.

## Extension choice

FormQL uses `.formql` as the canonical extension.

`.fql` is not claimed by default in this repo because `FQL` is already overloaded by unrelated languages and tools, including an existing VS Code marketplace extension for Ferret Query Language. We can support `.fql` later as an opt-in alias if we decide the ergonomics are worth the ambiguity.
