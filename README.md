# FormQL

FormQL is an Excel-like formula language with a real compiler pipeline and a relational typechecker. The primary backend today is PostgreSQL SQL, with PL/pgSQL available later when expression lowering is not enough.

The important architectural constraint is that the language is not defined by the old JavaScript POC. That codebase is reference material only. FormQL behavior should come from the language design, typed IR, and backend contracts in this repo.

## Current structure

- CLI: [`cmd/formqlc`](./cmd/formqlc/main.go)
- Core API: [`pkg/formql`](./pkg/formql/formula.go)
- Architecture notes: [`docs/architecture.md`](./docs/architecture.md)
- Sample offline catalog: [`examples/catalogs/opportunity-schema.json`](./examples/catalogs/opportunity-schema.json)
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
- `formqlc hir`: parse + typecheck + semantic IR
- `formqlc typecheck`: typecheck against a live catalog or schema file
- `formqlc query`: emit PostgreSQL SQL
- `formqlc catalog`: introspect a live Postgres schema
- `formqlc lsp`: run the language server

The typechecker is intentionally driven by a real catalog contract so the compiler, CLI, and LSP all exercise the same schema-resolution code paths.

## Quick start

```bash
make db-up
make catalog BASE_TABLE=opportunity
make typecheck BASE_TABLE=opportunity FORMULA='IF(customer_rel.email = NULL, "missing", customer_rel.first_name)'
make query BASE_TABLE=opportunity FORMULA='IF(offer_amount > 500000, customer_rel.first_name, "low")'
```

## Design warnings already visible

These are areas inherited from the bootstrap that should be treated as provisional, not language law:

- `*_rel` relationship syntax is useful, but may be more implementation-shaped than language-shaped.
- foreign-key column introspection currently derives relationship names mechanically from columns like `customer_id -> customer`; that may need explicit schema naming later.
- live catalog introspection currently collapses some Postgres types aggressively, for example arrays to string.
- `date` and `timestamp` are currently merged into one semantic type in the bootstrap.

If we keep any of those, it should be because they survive language design review, not because the JS repo happened to do something similar.
