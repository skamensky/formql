# FormQL VS Code Extension

This extension registers the `formql` language for `.formql` files and starts the FormQL language server for diagnostics, completions, hover, and jump-to-definition.

## Development

```bash
cd editors/vscode
npm install
code .
```

Press `F5` in that `editors/vscode` window to run the `Run FormQL Extension` launch config.

That launch config opens an Extension Development Host with the bundled offline workspace already loaded.

Do not press `F5` while focused on a `.formql` file in your normal VS Code window. In that context, `F5` means "debug this file", which is why VS Code asked about Plain Text debugging.

## Recommended configuration

Offline schema mode is the default recommendation because the language server can jump from a formula symbol directly into the schema file:

```json
{
  "formql.serverPath": "go",
  "formql.serverArgs": ["run", "${workspaceFolder}/../../../cmd/formqlc"],
  "formql.schemaPath": "${workspaceFolder}/schema/opportunity.formql.schema.json"
}
```

Each `.formql` file declares its table with a leading metadata comment such as `// formql: table=opportunity`, or an adjacent `.meta.json` sidecar.

## About `.fql`

This extension intentionally claims `.formql` as the canonical file extension.

`.fql` is a tempting shorthand, but `FQL` is already used by multiple unrelated languages and at least one existing VS Code marketplace extension, so claiming it by default would create unnecessary ambiguity.
