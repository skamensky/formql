# FormQL Architecture

This repo is the authoritative FormQL implementation.

An earlier JavaScript project exists as inspiration only. It is not the specification, and behavior should not be copied forward automatically just because it existed there.

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

Relationship names are normalized from foreign-key columns by stripping `_id`, so `customer_id` becomes `customer` and is used in formulas as `customer_rel`.

That is a bootstrap choice, not a language commitment.

## LSP

The LSP server is intentionally thin.

- it reloads catalog information from the live database
- it typechecks documents by calling the same compiler modules as the CLI
- it publishes parser/typechecker errors and warning diagnostics
- it serves basic completions for columns, relationships, and functions

That is the modularity test: editor features are consumers of the compiler, not a parallel implementation.

## Developer Workflow

Start the sample database:

```bash
make db-up
```

Inspect the live catalog:

```bash
make catalog BASE_TABLE=opportunity
```

Typecheck a formula against the live database:

```bash
make typecheck BASE_TABLE=opportunity FORMULA='IF(customer_rel.email = NULL, "missing", "ok")'
```

Generate PostgreSQL SQL:

```bash
make query BASE_TABLE=opportunity FORMULA='IF(amount > 100, customer_rel.first_name, "low")'
```

Run the language server:

```bash
make lsp BASE_TABLE=opportunity
```
