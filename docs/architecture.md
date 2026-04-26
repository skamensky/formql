# FormQL Architecture

This repo is the authoritative FormQL implementation.

An earlier prototype exists as inspiration only. It is not the specification, and behavior should not be copied forward automatically just because it existed there.

## Pipeline

The Go compiler does not compile directly from AST to SQL.

```
source
  -> AST
  -> typed semantic IR (HIR)
  -> backend lowering
  -> PostgreSQL SQL
```

That split is intentional:

- `AST` is syntax only.
- `HIR` carries resolved columns, resolved relationship paths, types, and warnings.
- the PostgreSQL backend only consumes `HIR`.
- if we later want a bytecode interpreter, that can be another backend from `HIR`; it does not replace `HIR`.

## Packages

- [`cmd/formqlc`](../cmd/formqlc/main.go): CLI entrypoint and developer tooling surface.
- [`pkg/formql/frontend`](../pkg/formql/frontend/parser.go): lexer and parser.
- [`pkg/formql/semantics`](../pkg/formql/semantics/lower.go): typechecker and AST -> HIR lowering.
- [`pkg/formql/ir`](../pkg/formql/ir/ir.go): typed semantic IR.
- [`pkg/formql/backend/postgres`](../pkg/formql/backend/postgres/render.go): PostgreSQL SQL renderer.
- [`pkg/formql/api`](../pkg/formql/api/api.go): shared host-facing API for compile/verify flows.
- [`pkg/formql/catalog`](../pkg/formql/catalog/provider.go): provider/cache/schema-info contracts.
- [`pkg/formql/filemeta`](../pkg/formql/filemeta/filemeta.go): file-scoped metadata parser for table resolution.
- [`pkg/formql/livecatalog`](../pkg/formql/livecatalog/postgres.go): live schema providers built on host-specific introspection sources.
- [`pkg/formql/lsp`](../pkg/formql/lsp/server.go): minimal LSP server built on the same compiler/typechecker modules.
- [`pkg/formql/wasm`](../pkg/formql/wasm/main.go): browser/Node entrypoint over the same Go engine.

## Mockable Interfaces

The compiler is not hardwired to the live DB layer.

- [`schema.Resolver`](../pkg/formql/schema/schema.go): the compiler-facing interface for tables, columns, and relationships.
- [`schema.Explorer`](../pkg/formql/schema/schema.go): extends `Resolver` for tooling surfaces like completion.
- [`catalog.Provider`](../pkg/formql/catalog/provider.go): load a schema snapshot for a logical catalog ref.
- [`catalog.ManagedProvider`](../pkg/formql/catalog/provider.go): provider plus optional lifecycle cleanup for host processes.
- [`catalog.InfoProvider`](../pkg/formql/catalog/provider.go): frontend/editor-facing schema info view.
- [`catalog.Cache`](../pkg/formql/catalog/provider.go): optional cache storage for schema snapshots.
- [`livecatalog.Source`](../pkg/formql/livecatalog/postgres.go): raw host-specific introspection seam used to build provider snapshots.

The tests in [`pkg/formql/formula_test.go`](../pkg/formql/formula_test.go) intentionally use a mocked catalog instead of a database connection.

## Warnings

The lowering phase can emit non-fatal warnings into `HIR`.

Current warning support:

- joins that use a non-indexed source foreign-key column
- joins that target a non-indexed referenced column

Those warnings depend on live catalog metadata from PostgreSQL. If index information is unknown, the compiler stays silent instead of guessing.

## Live Catalog And Schema Info

Live catalog loading is split into two layers:

- host-specific introspection via `livecatalog.Source`
- shared snapshot/info loading via `catalog.Provider` and `catalog.InfoProvider`

The current PostgreSQL-backed source introspects:

- base tables and columns from `information_schema.columns`
- direct foreign-key relationships from `information_schema.table_constraints`
- join index coverage from PostgreSQL index catalogs

Relationship names are currently derived mechanically from foreign-key column names by stripping suffixes like `_id`.

That is an implementation choice, not a language commitment. The long-term traversal surface may be renamed or made more explicit.

That lets different hosts resolve schema differently without changing compiler logic:

- CLI/LSP can use `database/sql`
- the PostgreSQL extension can introspect in-process
- browser callers can use checked-in catalog JSON through the same provider contracts

## File Metadata

Formula and document files do not inherit a base table from the workspace or folder name. File-based tooling resolves the table in this order:

- leading source comments, for example `// formql: table=rental_contract`
- adjacent sidecar JSON, for example `contract_overview.meta.json` with `{"table":"rental_contract"}`

The comment parser accepts `formql:`, `@formql`, and `formql-meta:` directives with `key=value` parameters so the metadata surface can grow without changing the file discovery model. `table` and `base_table` are currently synonyms. Missing or unresolved table metadata is a file-level error in the LSP and a command error for `-formula-file`/`-document-file`.

## LSP

The LSP server is intentionally thin in architecture, but now useful enough to drive editor workflows.

- it reloads catalog information through `catalog.Provider`
- it can also run in offline schema mode from a checked-in schema JSON file
- it resolves the base table per document from source or sidecar metadata
- it typechecks single-formula files and comma-separated documents by calling the same compiler modules as the CLI
- it publishes parser/typechecker errors and warning diagnostics
- it serves completions for columns, relationships, and functions
- it serves hover information for columns and relationships
- it serves definition requests for schema-backed symbols when a schema file is available

That is the modularity test: editor features are consumers of the compiler, not a parallel implementation.

The repo also includes a VS Code client in [`editors/vscode`](../editors/vscode/package.json) and bundled example workspaces in [`examples/workspaces/offline-rental-offer`](../examples/workspaces/offline-rental-offer/README.md), [`examples/workspaces/offline-rental-contract`](../examples/workspaces/offline-rental-contract/README.md), and [`examples/workspaces/offline-resale-sale`](../examples/workspaces/offline-resale-sale/README.md).

## WASM

The wasm entrypoint in [`pkg/formql/wasm`](../pkg/formql/wasm/main.go) reuses the same Go compiler, provider, and schema-info packages for frontend or Node usage.

Current wasm support:

- schema info from catalog JSON
- parsing, typechecking, and SQL generation from catalog JSON
- multi-field document compilation through the same browser/Node surface
- shared result shapes for frontend consumers

Current wasm limitation:

- SQL verification is reported as unavailable in `js/wasm` builds, because the current offline verifier backend is not yet portable to browser runtimes

## Developer Workflow

Start the sample database:

```bash
make db-up
```

Inspect the live catalog:

```bash
make catalog BASE_TABLE=rental_contract
```

Typecheck a formula against the live database:

```bash
make typecheck BASE_TABLE=rental_contract FORMULA='rep.manager.first_name & " @ " & rep.branch.name'
```

Generate PostgreSQL SQL:

```bash
make query BASE_TABLE=resale_sale FORMULA='vehicle.model_name & " / " & STRING(vehicle.model_year)'
```

Run the language server:

```bash
make lsp
```

## Join Alias Strategy

The PostgreSQL renderer generates a stable alias for each joined table using FNV-64a over the relationship path:

```
rel_<016x hash of dot-joined path>
```

For example, the path `customer_id__rel.assigned_rep_id__rel` always produces `rel_0eac59992a21af53` regardless of query context.

Sequential counters (`rel_0`, `rel_1`, ...) were considered and rejected:

- **Unstable across queries** — the alias for a given path shifts whenever another join is added or removed elsewhere in the same document. Adding a new projected field can silently rename every alias.
- **Order-dependent** — correct generation requires a guaranteed traversal order over all joins before any alias can be assigned.
- **Breaks golden tests on unrelated changes** — any formula change that adds or removes a join invalidates every downstream alias in the golden file, not just the changed one.

The hash is computed purely from the path, so the same relationship path always gets the same alias in every formula, every document, and every backend that renders from the same IR.

PostgreSQL identifiers are limited to 63 bytes. Relationship paths can grow long enough to exceed this limit after multiple hops. The hash approach keeps every alias at a fixed 21 bytes (`rel_` + 16 hex digits), so alias collisions from truncation are impossible regardless of traversal depth.

## Verification Architecture

FormQL verification should work entirely without a running database while still tracking PostgreSQL parser behavior.

To keep extension work testable and independent from extension build tooling, verification lives behind a small interface contract in [`pkg/formql/verify`](../pkg/formql/verify/verify.go).

### Contract

- `verify.Verifier`: stage-level verifier interface
- `verify.Pipeline`: composition of verifier stages with fail-fast behavior
- `verify.Request`: SQL + bind args + verification mode
- `verify.Result`: normalized success + diagnostics output

This means extension-specific code is just a host bridge into shared Go logic, not a second verifier implementation.

### Recommended staged rollout

1. Use `go-pgquery` as the default offline verifier stage for parser correctness.
2. Keep the PostgreSQL extension as a host bridge into the same verifier stages, not a separate implementation.
3. Keep any custom lints/policy checks advisory, never correctness-authoritative.
4. Compose stages with `verify.Pipeline` so the same test corpus runs in CI, CLI, and extension environments.

### Correctness stance

Verification correctness is defined by PostgreSQL itself.

- Do **not** implement a separate SQL parser as the source of truth in FormQL.
- For offline syntax/analysis checks, delegate to `go-pgquery` (libpg_query via pure-Go runtime).
- A running PostgreSQL instance is optional for additional runtime checks, not required for baseline verification.
- Any FormQL-specific lint or policy pass must remain advisory over parser validity.

## PostgreSQL Extension

The PostgreSQL extension now ships as a thin PGXS module that links a Go C archive built from the same repository code used by the CLI and tests.

Current layering:

1. shared Go compiler and verifier packages in `pkg/formql/...`
2. C-ABI bridge in [`pkg/formql/capi`](../pkg/formql/capi/main.go)
3. thin PostgreSQL wrapper in [`ext/formql/src/formql.c`](../ext/formql/src/formql.c)

The wrapper does not own verification or compilation semantics. It only:

- converts PostgreSQL values to C strings / JSON text
- calls the exported Go bridge functions
- converts returned JSON back into PostgreSQL `text`, `boolean`, or `jsonb`

The Go runtime lives in a separate shared helper library. PostgreSQL backend processes still need enough optional static TLS available to load that helper after startup, so container/runtime environments should set `GLIBC_TUNABLES=glibc.rtld.optional_static_tls=2048` or an equivalent host-level setting. `formql.so` remains the SQL-callable wrapper and links to the helper lazily at call time.

### API surface

Current SQL API:

- `formql_verify_sql_error(sql text) returns text`
- `formql_verify_sql_ok(sql text) returns boolean`
- `formql_verify_sql_diagnostics(sql text) returns jsonb`
- `formql_catalog(base_table text) returns jsonb`
- `formql_compile_catalog(catalog jsonb, formula text, field_alias text default 'result', verify_mode text default 'syntax') returns jsonb`
- `formql_compile_live(base_table text, formula text, field_alias text default 'result', verify_mode text default 'syntax') returns jsonb`
- `formql_compile_document_catalog(catalog jsonb, document text, verify_mode text default 'syntax') returns jsonb`
- `formql_compile_document_live(base_table text, document text, verify_mode text default 'syntax') returns jsonb`

Design notes:

- the scalar verification helpers stay easy to call from SQL and shell scripts
- the compile entrypoints are JSON because compiler output is structured
- the live compile entrypoints resolve their catalog in-process, then call the same Go compiler bridge as every other host
- no execution endpoint in v1
- compiler ownership stays in Go; only schema introspection differs by host

### Build strategy

Use a split build so extension specifics stay isolated and testable:

1. Keep shared verification contract and fixtures in Go (`pkg/formql/verify`).
2. Build extension as a thin adapter layer that maps SQL calls to the shared Go bridge.
3. Prefer PGXS-based extension packaging (`Makefile`, `.control`, `--*.sql`) for compatibility.
4. Keep CI matrix with:
   - Go unit tests (no Postgres required)
   - extension build checks against targeted PostgreSQL majors
   - extension integration tests in containers

### Test strategy

Adopt the same fixture corpus in every layer:

- **Contract tests (Go, offline):** expected diagnostics for valid/invalid SQL.
- **Golden tests (compiler):** formula -> SQL output snapshots.
- **Extension integration tests:** call the extension verification functions against the same SQL fixtures and assert parity with the offline verifier for parser outcomes.
- **Upgrade tests:** install old extension version, upgrade, and verify API compatibility.

Minimum gating policy for the extension phase:

- no merge unless fixture parity passes (offline verifier vs extension verifier)
- no merge unless extension install/upgrade scripts pass on every supported PG major


### Return contract

To keep SQL callsites simple, the extension keeps a scalar-first verification API and adds JSON only where structured output is necessary:

- `formql_verify_sql_error(sql text) returns text`
  - `NULL` means verification passed
  - non-`NULL` means verification failed and contains a user-readable error summary
- `formql_verify_sql_ok(sql text) returns boolean`
  - `TRUE` when verification passed
  - `FALSE` when verification failed
- `formql_verify_sql_diagnostics(sql text) returns jsonb`
  - full machine-readable diagnostics when needed
- `formql_compile_catalog(...) returns jsonb`
  - compiler output plus verification result from the shared Go engine
- `formql_compile_document_catalog(...) returns jsonb`
  - multi-projection compiler output plus verification result from the shared Go engine

### E2E Docker testing requirement (extension build + runtime)

Extension work should not merge without an end-to-end Docker job that:

1. builds a PostgreSQL image with the FormQL extension installed
2. starts a container from that image
3. runs SQL assertions against extension functions (`*_ok`, `*_error`, optional `*_diagnostics`)
4. validates contract behavior:
   - valid SQL -> `*_error IS NULL` and `*_ok = TRUE`
   - invalid SQL -> `*_error IS NOT NULL` and `*_ok = FALSE`

Recommended repo layout for this phase:

```text
docker/
  extension/
    Dockerfile           # PostgreSQL + built/installed extension
    smoke.sql            # E2E SQL assertions

ext/formql/
  Makefile
  formql.control
  sql/
  src/
```

Recommended make targets:

- `make ext-build`       (build extension artifacts)
- `make ext-docker-build` (build image with extension)
- `make ext-e2e`         (run smoke SQL in container)

### File hierarchy

```text
ext/
  formql/
    Makefile
    formql.control
    sql/
      formql--0.1.0.sql
      formql--0.1.0--0.1.1.sql
    src/
      formql.c                 # SQL-callable entrypoints
    test/
      sql/
      expected/

testdata/
  verify/
    valid.sql
    invalid.sql
    diagnostics/*.json

pkg/formql/verify/
  verify.go                    # shared request/result contract
  pgquery.go                   # offline parser verifier

pkg/formql/capi/
  main.go                      # Go C-ABI bridge used by the extension
```

Rationale:

- keeps extension mechanics isolated under `ext/`
- keeps reusable fixtures in repository-level `testdata/verify`
- keeps Go-first contract and offline tests independent from extension toolchain
