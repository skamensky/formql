# Agent Handoff

## Status
Current branch: `main`
Current HEAD: `037c2fc` `Add cached schema info to web playground`
Previous major integration commit: `0a23bce` `Unify catalog hosts and add wasm web backend`

Original handoff state: repo was clean at handoff time.

Post-review session update:
- multi-field document support was extended after reviewing this handoff
- implemented scope: document AST/parser, shared document lowering, deterministic aliases, PostgreSQL multi-projection rendering, public Go/API entrypoints, CLI commands, LSP diagnostics, wasm JS methods, web backend endpoint, playground document mode, and PostgreSQL extension bridge functions
- verified during the first pass with `go test ./...`, `make wasm-smoke`, `make web-smoke`, and `git diff --check`; rerun the full command set after any further changes

## Compact Summary
FormQL is no longer split across separate host-specific logic paths for compiler/catalog semantics.

The important architectural move is complete:
- compiler + typechecker + SQL generation live in shared Go code
- schema loading is abstracted through shared provider contracts
- PostgreSQL extension is a thin host bridge, not a second compiler
- wasm/browser entrypoint uses the same Go compiler packages
- browser verification is handled by an online backend because the current offline verifier backend is not portable to `js/wasm`

What now exists:
- shared catalog/provider/cache layer: `pkg/formql/catalog`
- shared host API: `pkg/formql/api`
- live Postgres introspection layer: `pkg/formql/livecatalog`
- Go C bridge for PG extension: `pkg/formql/capi`
- wasm entrypoint: `pkg/formql/wasm`
- local web backend + playground: `cmd/formqlweb`, `web/playground`
- PG extension + Docker E2E path: `ext/formql`, `docker/extension`, `make ext-e2e`

## Verified Before Handoff
These were run successfully in this environment:
- `go test ./...`
- `make wasm-smoke`
- `make ext-e2e`
- `make web-smoke`

Meaning:
- shared Go compiler/verifier code passes unit/integration tests
- wasm bundle builds and runs in Node
- PostgreSQL extension builds, installs, loads, and passes smoke SQL in Docker
- local web backend serves wasm, compiles in wasm, and verifies generated SQL through backend HTTP API

## Key Architecture To Preserve

### 1. Catalog loading is shared and host-neutral
Main interfaces live in `pkg/formql/catalog/provider.go`.

Important types:
- `catalog.Ref`
- `catalog.Snapshot`
- `catalog.Provider`
- `catalog.ManagedProvider`
- `catalog.InfoProvider`
- `catalog.Cache`
- `catalog.CachingProvider`
- `catalog.Info`

Reasoning:
- the compiler should never care whether schema came from JSON, `database/sql`, PostgreSQL SPI, or browser-loaded data
- caching belongs outside compiler logic; hosts opt into it
- frontend/editor schema browsing should use the same provider seam, not a parallel DTO path

### 2. Live introspection is split correctly
`pkg/formql/livecatalog/postgres.go` now separates:
- host-specific raw introspection
- shared conversion of raw metadata into compiler catalog

Important idea:
- extension no longer owns type mapping / relationship derivation logic
- extension gathers raw metadata only
- shared Go code turns raw metadata into `schema.Catalog`

Reasoning:
This was a key correction. Without it, Postgres extension behavior would drift from CLI/browser behavior.

### 3. PostgreSQL extension is a host bridge
Relevant files:
- `ext/formql/src/formql.c`
- `pkg/formql/capi/main.go`
- `ext/formql/sql/formql--0.1.0.sql`

Current SQL API:
- `formql_verify_sql_error(sql text)`
- `formql_verify_sql_ok(sql text)`
- `formql_verify_sql_diagnostics(sql text)`
- `formql_catalog(base_table text)`
- `formql_compile_catalog(catalog jsonb, formula text, field_alias text default 'result', verify_mode text default 'syntax')`
- `formql_compile_live(base_table text, formula text, field_alias text default 'result', verify_mode text default 'syntax')`
- `formql_compile_document_catalog(catalog jsonb, document text, verify_mode text default 'syntax')`
- `formql_compile_document_live(base_table text, document text, verify_mode text default 'syntax')`

Important constraint:
- C wrapper should stay thin
- semantic logic belongs in Go
- if more extension APIs are added, keep them as marshaling/introspection adapters only

### 4. WASM is real, but verification is intentionally split
Relevant files:
- `pkg/formql/wasm/main.go`
- `pkg/formql/verify/default_js.go`
- `cmd/formqlweb/main.go`

Current wasm JS surface:
- `FormQL.loadSchemaInfoJSON(...)`
- `FormQL.compileCatalogJSON(...)`
- `FormQL.compileDocumentCatalogJSON(...)`
- `FormQL.compileAndVerifyCatalogJSON(...)`
- `FormQL.compileAndVerifyDocumentCatalogJSON(...)`
- `FormQL.verifySQL(...)`

Actual behavior:
- compile/type/schema flows work in wasm
- verification in wasm returns `verification_unavailable`
- real SQL verification is done by backend HTTP API

Reasoning:
This is deliberate, not a bug. Current `go-pgquery` backend is not browser-portable. The correct near-term architecture is browser compile + online verify.

### 5. Web backend is now the online verification host for wasm
Relevant files:
- `cmd/formqlweb/main.go`
- `web/playground/index.html`
- `web/playground/smoke.cjs`

Current endpoints:
- `GET /api/health`
- `GET /api/catalog/rental-agency`
- `GET /api/schema-info/rental-agency?base_table=...`
- `POST /api/verify-sql`
- `POST /api/compile-and-verify`
- `POST /api/compile-document-and-verify`
- static `/wasm/...`
- static playground `/`

Important detail:
- backend now uses the shared cached provider layer for rental-agency schema info
- this was added so frontend schema browsing exercises the same abstraction instead of reading raw files ad hoc

## Why The Current Shape Is Good
- one source of truth for compiler/catalog semantics
- hosts differ only in transport/introspection/runtime concerns
- extension is no longer a semantic fork
- future LSP/web/editor work can build on `catalog.Provider` / `catalog.InfoProvider`
- multi-field support can now be added at the language/document layer without redoing host integration

## Multi-Field Query Documents
Multi-field documents are now implemented as a proper language/document layer, not as SQL string splitting.

Supported top-level input shape:
- `complex_formula, regular_field, relation_rel.name`
- `actual_total, customer_rel.email, amount + 1 AS amount_plus_one`

Implemented behavior:
- one base table per document
- multiple selected output fields
- each item is a full independent expression
- pass-through base-table fields and relationship fields work alongside formulas
- top-level commas separate document fields
- commas inside function calls remain argument separators
- optional contextual aliases use `expr AS alias`
- deterministic default aliases are generated for unaliased fields
- joins and warnings are shared across all fields in first-use order
- output field order is preserved

Alias policy:
- `actual_total` defaults to alias `actual_total`
- `customer_rel.email` defaults to alias `customer_email`
- computed expressions default to `result`, then `result_2`, etc. when needed
- explicit aliases use `expr AS alias`
- duplicate explicit aliases are semantic errors

Core APIs:
- single-formula compatibility remains on `Parse`, `Lower`, `RenderSQL`, and `Compile`
- document APIs are `ParseDocument`, `LowerDocument`, `RenderDocumentSQL`, and `CompileDocument`
- host-facing document APIs exist in `pkg/formql/api`

Reasoning:
- it directly unlocks “views” / report-like output
- it is still compatible with single-table scope
- it reuses almost all current expression/typechecking work
- it benefits from the now-shared join/caching/catalog architecture
- it avoids prematurely designing DRY/lambda facilities before basic multi-projection exists

The main thing to avoid is a hack where top-level comma lists are bolted onto the current expression parser without a document abstraction. That would make aliases, bindings, and future view-level features painful.

## Good Document Test Cases
- `actual_total, customer_rel.email`
- `IF(customer_rel.email = NULL, "missing", customer_rel.email), actual_total`
- `rep_rel.manager_rel.first_name & " @ " & rep_rel.branch_rel.name, rep_rel.branch_rel.name`
- `vehicle_rel.model_name, vehicle_rel.model_year, STRING(vehicle_rel.model_year)`

## Files Most Relevant For Next Session
- `pkg/formql/frontend/parser.go`
- `pkg/formql/formula.go`
- `pkg/formql/ir/ir.go`
- `pkg/formql/semantics/lower.go`
- `pkg/formql/backend/postgres/render.go`
- `pkg/formql/api/api.go`
- `pkg/formql/catalog/provider.go`
- `cmd/formqlc/main.go`
- `cmd/formqlweb/main.go`
- `pkg/formql/wasm/main.go`
- `pkg/formql/capi/main.go`
- `pkg/formql/lsp/server.go`

## Operational Commands
Useful commands to keep using:
- `go test ./...`
- `make wasm-smoke`
- `make web-smoke`
- `make ext-e2e`
- `make web-backend`

## Final Note
The repo is in a good place architecturally now. Multi-field document support has validated the language boundary, shared join planning, and backend projection model without overcommitting to reusable bindings or Excel-lambda-like features. The next pressure points are likely alias polish, document-aware editor completions, and higher-level view packaging.
