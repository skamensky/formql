# FormQL

FormQL is an Excel-like formula language with a real compiler pipeline and a relational typechecker. The primary backend today is PostgreSQL SQL, with PL/pgSQL available later when expression lowering is not enough.

An earlier prototype exists as inspiration only. FormQL behavior in this repo is defined by the language design, typed IR, and backend contracts here, not by legacy implementation details.

## Current structure

- CLI: [`cmd/formqlc`](./cmd/formqlc/main.go)
- Core API: [`pkg/formql`](./pkg/formql/formula.go)
- LSP: [`pkg/formql/lsp`](./pkg/formql/lsp/server.go)
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
- `formqlc hir`: parse + typecheck + semantic IR
- `formqlc typecheck`: typecheck against a live catalog or schema file
- `formqlc query`: emit PostgreSQL SQL
- `formqlc catalog`: introspect a live Postgres schema
- `formqlc lsp`: run the language server

The typechecker is intentionally driven by a real catalog contract so the compiler, CLI, and LSP all exercise the same schema-resolution code paths.

## VS Code

The repo now includes a local VS Code extension in [`editors/vscode`](./editors/vscode/package.json).

- canonical file extension: `.formql`
- diagnostics: parser and typechecker errors
- completions: columns, relationships, and built-ins
- hover: columns and relationships
- jump to definition: offline schema mode, into the schema JSON file

The fastest live demos are the bundled rental-agency workspaces at [`examples/workspaces/offline-rental-offer`](./examples/workspaces/offline-rental-offer/README.md), [`examples/workspaces/offline-rental-contract`](./examples/workspaces/offline-rental-contract/README.md), and [`examples/workspaces/offline-resale-sale`](./examples/workspaces/offline-resale-sale/README.md). They use `go run` plus a checked-in schema file, so users do not need a running database to see the editor features work. The `formulas/` directories are now a broader language corpus, not just a couple of demo snippets: they are compiled in tests and deliberately cover the current builtin/operator surface.

## Quick start

```bash
make db-up
make catalog BASE_TABLE=rental_contract
make typecheck BASE_TABLE=rental_contract FORMULA='rep_rel.manager_rel.first_name & " @ " & rep_rel.branch_rel.name'
make query BASE_TABLE=resale_sale FORMULA='vehicle_rel.model_name & " / " & STRING(vehicle_rel.model_year)'
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
