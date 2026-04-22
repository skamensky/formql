# Offline Opportunity Workspace

This workspace is the fastest way to try FormQL in VS Code without a running database.

## Open in VS Code

Open this folder directly:

```bash
code /home/shmuel/repos/shkamensky/formql/examples/workspaces/offline-opportunity
```

The workspace settings already point the extension at:

- the local compiler via `go run`
- the bundled schema file
- the `opportunity` base table

Open any file under `formulas/` to test:

- completions for columns, relationships, and functions
- hover over columns and relationships
- jump to definition into `schema/opportunity.formql.schema.json`
- parser/typechecker diagnostics
