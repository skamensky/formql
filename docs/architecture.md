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
