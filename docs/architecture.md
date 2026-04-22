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
- [`pkg/formql/livecatalog`](../pkg/formql/livecatalog/postgres.go): live schema loader from PostgreSQL.
- [`pkg/formql/lsp`](../pkg/formql/lsp/server.go): minimal LSP server built on the same compiler/typechecker modules.

## Mockable Interfaces

The compiler is not hardwired to the live DB layer.

- [`schema.Resolver`](../pkg/formql/schema/schema.go): the compiler-facing interface for tables, columns, and relationships.
- [`schema.Explorer`](../pkg/formql/schema/schema.go): extends `Resolver` for tooling surfaces like completion.
- [`livecatalog.Provider`](../pkg/formql/livecatalog/postgres.go): a live DB-backed catalog source used by the CLI and LSP.

The tests in [`pkg/formql/formula_test.go`](../pkg/formql/formula_test.go) intentionally use a mocked catalog instead of a database connection.

## Warnings

The lowering phase can emit non-fatal warnings into `HIR`.

Current warning support:

- joins that use a non-indexed source foreign-key column
- joins that target a non-indexed referenced column

Those warnings depend on live catalog metadata from PostgreSQL. If index information is unknown, the compiler stays silent instead of guessing.

## Live Catalog

The live catalog loader introspects:

- base tables and columns from `information_schema.columns`
- direct foreign-key relationships from `information_schema.table_constraints`
- join index coverage from PostgreSQL index catalogs

Relationship names are currently derived mechanically from foreign-key column names by stripping suffixes like `_id`.

That is an implementation choice, not a language commitment. The long-term traversal surface may be renamed or made more explicit.

## LSP

The LSP server is intentionally thin in architecture, but now useful enough to drive editor workflows.

- it reloads catalog information from the live database
- it can also run in offline schema mode from a checked-in schema JSON file
- it typechecks documents by calling the same compiler modules as the CLI
- it publishes parser/typechecker errors and warning diagnostics
- it serves completions for columns, relationships, and functions
- it serves hover information for columns and relationships
- it serves definition requests for schema-backed symbols when a schema file is available

That is the modularity test: editor features are consumers of the compiler, not a parallel implementation.

The repo also includes a VS Code client in [`editors/vscode`](../editors/vscode/package.json) and bundled example workspaces in [`examples/workspaces/offline-rental-offer`](../examples/workspaces/offline-rental-offer/README.md), [`examples/workspaces/offline-rental-contract`](../examples/workspaces/offline-rental-contract/README.md), and [`examples/workspaces/offline-resale-sale`](../examples/workspaces/offline-resale-sale/README.md).

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
make typecheck BASE_TABLE=rental_contract FORMULA='rep_rel.manager_rel.first_name & " @ " & rep_rel.branch_rel.name'
```

Generate PostgreSQL SQL:

```bash
make query BASE_TABLE=resale_sale FORMULA='vehicle_rel.model_name & " / " & STRING(vehicle_rel.model_year)'
```

Run the language server:

```bash
make lsp BASE_TABLE=rental_contract
```

## Verification Architecture (Extension-Ready, Testable in Go)

FormQL verification should work entirely without a running database while still tracking PostgreSQL parser behavior.

To keep extension work testable and independent from extension build tooling, verification lives behind a small interface contract in [`pkg/formql/verify`](../pkg/formql/verify/verify.go).

### Contract

- `verify.Verifier`: stage-level verifier interface
- `verify.Pipeline`: composition of verifier stages with fail-fast behavior
- `verify.Request`: SQL + bind args + verification mode
- `verify.Result`: normalized success + diagnostics output

This means extension-specific code is just one implementation of the verifier interface, not the interface itself.

### Recommended staged rollout

1. Use `go-pgquery` as the default offline verifier stage for parser correctness.
2. Keep optional extension-backed checks as additional stages, not a requirement for verification.
3. Keep any custom lints/policy checks advisory, never correctness-authoritative.
4. Compose stages with `verify.Pipeline` so the same test corpus runs in CI and extension environments.

### Correctness stance

Verification correctness is defined by PostgreSQL itself.

- Do **not** implement a separate SQL parser as the source of truth in FormQL.
- For offline syntax/analysis checks, delegate to `go-pgquery` (libpg_query via pure-Go runtime).
- A running PostgreSQL instance is optional for additional runtime checks, not required for baseline verification.
- Any FormQL-specific lint or policy pass must remain advisory over parser validity.

## Proposed PostgreSQL Extension (Design Only, Not Implemented)

This section defines a first-pass contract for the extension work. It is
intentionally design-only and does not introduce extension code yet.

### API surface

Expose a small SQL API from the extension so app code and tests use a stable contract:

- `formql_verify_sql(sql text) returns jsonb`
  - parser-only verification response
  - returns `{ ok: bool, diagnostics: [{code, message, detail?, hint?, position?}] }`
- `formql_verify_expression(base_table text, formula text) returns jsonb`
  - runs FormQL-to-SQL compilation externally, then verifies generated SQL in-db
  - same response shape
- `formql_version() returns text`
  - extension version + parser compatibility marker

Design notes:

- diagnostics are normalized JSON so Go/CLI/LSP surfaces can consume one shape
- no execution endpoint in v1 (verification only)
- if planning-level checks are added later, expose a mode argument rather than a separate function

### Build strategy

Use a split build so extension specifics stay isolated and testable:

1. Keep shared verification contract and fixtures in Go (`pkg/formql/verify`).
2. Build extension as a thin adapter layer that maps SQL calls to parser/verification engine.
3. Prefer PGXS-based extension packaging (`Makefile`, `.control`, `--*.sql`) for compatibility.
4. Keep CI matrix with:
   - Go unit tests (no Postgres required)
   - extension build checks against targeted PostgreSQL majors
   - extension integration tests in containers

### Test strategy

Adopt the same fixture corpus in every layer:

- **Contract tests (Go, offline):** expected diagnostics for valid/invalid SQL.
- **Golden tests (compiler):** formula -> SQL output snapshots.
- **Extension integration tests:** call `formql_verify_sql` against the same SQL fixtures and assert parity with offline verifier for parser outcomes.
- **Upgrade tests:** install old extension version, upgrade, and verify API compatibility.

Minimum gating policy for the extension phase:

- no merge unless fixture parity passes (offline verifier vs extension verifier)
- no merge unless extension install/upgrade scripts pass on every supported PG major


### Return contract proposal (no JSON parsing required)

To keep the SQL-callsite simple, use a scalar-first API and make JSON optional:

- `formql_verify_sql_error(sql text) returns text`
  - `NULL` means verification passed
  - non-`NULL` means verification failed and contains a user-readable error summary
- `formql_verify_sql_ok(sql text) returns boolean`
  - `TRUE` when verification passed
  - `FALSE` when verification failed
- optional advanced API: `formql_verify_sql_diagnostics(sql text) returns jsonb`
  - full diagnostics only for callers that need machine-readable details

This gives developers a no-JSON happy path while still supporting rich diagnostics.

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

### File hierarchy (proposed)

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
      verify_bridge.c          # bridge to verification engine
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
  pipeline.go (optional split)
  pgquery.go                   # offline parser verifier
```

Rationale:

- keeps extension mechanics isolated under `ext/`
- keeps reusable fixtures in repository-level `testdata/verify`
- keeps Go-first contract and offline tests independent from extension toolchain
