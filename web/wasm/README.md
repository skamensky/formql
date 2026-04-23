# FormQL WASM

This bundle exposes the shared Go compiler, schema info loader, and verifier to
browser or Node environments through a single `globalThis.FormQL` object.

Build it with:

```bash
make wasm-build
```

Smoke-test it with:

```bash
make wasm-smoke
```

For a browser-like round trip with online SQL verification:

```bash
make web-smoke
```

That target serves the wasm files over HTTP, compiles a formula in wasm, and
posts the generated SQL to the local `formqlweb` verifier backend.

The current JS surface is:

- `FormQL.loadSchemaInfoJSON(catalogJSON, options?)`
- `FormQL.compileCatalogJSON(catalogJSON, formula, options?)`
- `FormQL.compileAndVerifyCatalogJSON(catalogJSON, formula, options?)`
- `FormQL.verifySQL(sql, options?)`

Options currently support:

- `baseTable`
- `namespace`
- `fieldAlias`
- `verifyMode`
- `revision`
