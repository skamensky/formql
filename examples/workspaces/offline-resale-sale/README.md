# Offline Resale Sale Workspace

This workspace reuses the same rental-agency schema with used-vehicle resale formulas. It is the quickest way to test FormQL against a different slice of the same domain model and confirm that the compiler is not hard-wired to one table.

## Open in VS Code

Open this folder directly:

```bash
code /home/shmuel/repos/shkamensky/formql/examples/workspaces/offline-resale-sale
```

The workspace points the FormQL extension at:

- the local compiler via `go run`
- the shared rental-agency schema in `examples/catalogs`
- per-file table metadata declared with `// formql: table=resale_sale`

Open any file under `formulas/` to test:

- buyer-channel formulas that mix retail, wholesale, and repeat-renter logic
- vehicle lineage paths through purchase and vendor relationships
- explicit `STRING(...)` casting inside string concatenation

Open files under `documents/` to test multi-field query documents for used-car resale reporting.

## Feature Coverage

This workspace complements the contract workspace with resale-specific language coverage:

- alternate inequality spellings: `channel_override.formql`, `trade_route.formql`
- date casting and date arithmetic: `days_to_year_end.formql`, `warranty_window_days.formql`
- absolute deltas and margin logic: `margin_gap_abs.formql`, `margin_band.formql`
- warning-path traversal through manager relationships: `manager_line.formql`
- document examples: `resale_margin_view.formql`
